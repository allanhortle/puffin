// Package task makes, opens and removes tasks: a directory, a tmux session,
// and the worktrees mounted inside it.
//
// A task belongs to no repository. One with no worktree is an exploration: an
// agent at the task's root that can read the repositories it was pointed at.
// Attaching a repository gives the task a worktree of it, a window on that
// worktree, and an agent started inside it so the repository's own agent
// settings apply.
package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/allanhortle/puffin/internal/api"
	"github.com/allanhortle/puffin/internal/config"
	"github.com/allanhortle/puffin/internal/git"
	"github.com/allanhortle/puffin/internal/pr"
	"github.com/allanhortle/puffin/internal/run"
	"github.com/allanhortle/puffin/internal/store"
	"github.com/allanhortle/puffin/internal/tmux"
)

const (
	rootWindow = "root"
	// Where a task keeps what puffin wrote for it, inside its directory.
	stateDir = ".puffin"
	// A repository's pre_remove commands stop servers and containers.
	hookTimeout = 5 * time.Minute

	agentToken = "<agent>"
	setupToken = "<setup>"
)

type progressKey struct{}

// WithProgress has every slow step of a task operation reported to report as
// it starts, in a few words a person can read while they wait.
func WithProgress(ctx context.Context, report func(string)) context.Context {
	return context.WithValue(ctx, progressKey{}, report)
}

func step(ctx context.Context, format string, args ...any) {
	if report, ok := ctx.Value(progressKey{}).(func(string)); ok {
		report(fmt.Sprintf(format, args...))
	}
}

type Manager struct {
	cfg   config.Config
	store *store.Store
	tmux  tmux.Client

	// One change to tasks at a time. They are rare, and each one touches the
	// database, the filesystem, git and tmux in turn.
	mu sync.Mutex
	// When each clone's default branch was last fetched, under mu.
	fetched map[string]time.Time
}

// Every connection to the remote costs seconds before any work is done, so
// one is only made when it will tell us something. A default branch fetched
// this recently is fresh enough to start new work from.
const baseFresh = 5 * time.Minute

func New(cfg config.Config, st *store.Store, tm tmux.Client) *Manager {
	return &Manager{cfg: cfg, store: st, tmux: tm, fetched: map[string]time.Time{}}
}

// Scan looks one level into each configured root for clones and records them.
func (m *Manager) Scan(ctx context.Context) ([]store.Repo, error) {
	type clone struct {
		path      string
		worktrees int
	}
	best := map[string]clone{}
	for _, root := range m.cfg.Roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			dir := filepath.Join(root, entry.Name())
			// A worktree's .git is a file; only a clone has the directory.
			if info, err := os.Stat(filepath.Join(dir, ".git")); err != nil || !info.IsDir() {
				continue
			}
			name := git.RepoName(ctx, filepath.Join(dir, ".git"))
			if name == "" {
				continue
			}
			// A repository can be cloned more than once. The clone that work
			// grows from is the one with the most worktrees.
			c := clone{dir, git.WorktreeCount(ctx, dir)}
			if held, ok := best[name]; !ok || c.worktrees > held.worktrees {
				best[name] = c
			}
		}
	}
	for name, c := range best {
		if err := m.store.SaveClone(ctx, name, c.path, git.DefaultBranch(ctx, c.path)); err != nil {
			return nil, err
		}
	}
	return m.store.Repos(ctx)
}

func short(repo string) string {
	_, name, _ := strings.Cut(repo, "/")
	return name
}

// repo resolves what a person typed, `name` or `owner/name`, scanning again
// if nothing known matches.
func (m *Manager) repo(ctx context.Context, name string) (store.Repo, error) {
	match := func(repos []store.Repo) (store.Repo, error) {
		var found []store.Repo
		for _, r := range repos {
			if strings.EqualFold(r.Name, name) || strings.EqualFold(short(r.Name), name) {
				found = append(found, r)
			}
		}
		switch len(found) {
		case 1:
			return found[0], nil
		case 0:
			return store.Repo{}, fmt.Errorf("no clone of %q under the configured roots", name)
		}
		return store.Repo{}, fmt.Errorf("%q names more than one repository; give the owner too", name)
	}
	repos, err := m.store.Repos(ctx)
	if err != nil {
		return store.Repo{}, err
	}
	if r, err := match(repos); err == nil {
		return r, nil
	}
	if repos, err = m.Scan(ctx); err != nil {
		return store.Repo{}, err
	}
	return match(repos)
}

var (
	linearID = regexp.MustCompile(`^[A-Za-z]{2,10}-\d+\b`)
	notSlug  = regexp.MustCompile(`[^a-z0-9]+`)
)

// Slugify turns a prompt into a name short enough for a status bar: a leading
// issue id if there is one, then the first few words.
func Slugify(text string) string {
	const maxWords, maxLen = 4, 40
	words := strings.FieldsFunc(notSlug.ReplaceAllString(strings.ToLower(text), " "),
		func(r rune) bool { return r == ' ' })
	keep := maxWords
	if linearID.MatchString(strings.TrimSpace(text)) {
		keep += 2 // the id's two halves do not count against the words
	}
	if len(words) > keep {
		words = words[:keep]
	}
	slug := strings.Join(words, "-")
	for len(slug) > maxLen {
		cut := strings.LastIndex(slug, "-")
		if cut < 0 {
			return slug[:maxLen]
		}
		slug = slug[:cut]
	}
	return slug
}

func (m *Manager) freeSlug(ctx context.Context, want string) (string, error) {
	for n := 1; ; n++ {
		slug := want
		if n > 1 {
			slug = want + "-" + strconv.Itoa(n)
		}
		taken, err := m.store.SlugTaken(ctx, slug)
		if err != nil {
			return "", err
		}
		// A session of that name that is not ours would be taken over.
		if !taken && !m.tmux.HasSession(ctx, slug) {
			if _, err := os.Stat(filepath.Join(m.cfg.TasksDir, slug)); errors.Is(err, os.ErrNotExist) {
				return slug, nil
			}
		}
	}
}

// placed is a repository a task was asked for, and the clone the asker was
// standing in, if they were standing in one.
type placed struct {
	repo store.Repo
	dir  string
}

// place resolves what a task was asked for. A repository named by where the
// asker stands is registered on the spot if it was not known, which is what
// lets one outside the configured roots be worked on at all.
func (m *Manager) place(ctx context.Context, ref api.RepoRef) (placed, error) {
	if ref.Name == "" {
		// A repository with no origin has no name to be configured under,
		// so it can only be worked on where it is.
		return placed{dir: ref.Path}, nil
	}
	r, err := m.repo(ctx, ref.Name)
	if err != nil {
		if ref.Path == "" {
			return placed{}, err
		}
		if err := m.store.SaveClone(ctx, ref.Name, ref.Path, git.DefaultBranch(ctx, ref.Path)); err != nil {
			return placed{}, err
		}
		if r, err = m.repo(ctx, ref.Name); err != nil {
			return placed{}, err
		}
	}
	dir := ref.Path
	if dir == "" {
		dir = r.Path
	}
	return placed{repo: r, dir: dir}, nil
}

// New makes a task. What kind is the repository's to say, not the caller's:
//
//   - A repository configured for worktrees gets a branch and a worktree,
//     with an agent in it on the prompt. Several of them share the task,
//     side by side in its directory, with one agent above them all.
//   - Any other repository is worked on in place: the task is a session in
//     the clone itself. There is one per clone, so asking again goes to it.
//   - Bare, or with no repository at all, the task is an agent at its own
//     root, able to read the repositories and nothing more.
func (m *Manager) New(ctx context.Context, req api.NewTask) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prompt := strings.TrimSpace(req.Prompt)
	refs := req.Repos
	if len(refs) == 0 && m.cfg.DefaultRepo != "" {
		refs = []api.RepoRef{{Name: m.cfg.DefaultRepo}}
	}
	// Resolve everything that can be refused before anything is made.
	var places []placed
	for _, ref := range refs {
		p, err := m.place(ctx, ref)
		if err != nil {
			return store.Task{}, err
		}
		places = append(places, p)
	}
	if !req.Bare {
		for _, p := range places {
			if m.cfg.Repo[p.repo.Name].Worktree {
				continue
			}
			if len(places) > 1 {
				return store.Task{}, fmt.Errorf("%s is worked on in place, so it cannot share a task; set `worktree = true` for it in the config to give it worktrees", filepath.Base(p.dir))
			}
			return m.here(ctx, p.dir, req.Name, prompt)
		}
	}

	want := Slugify(req.Name)
	if want == "" {
		want = Slugify(prompt)
	}
	if want == "" {
		want = "task-" + time.Now().Format("0102-1504")
	}
	slug, err := m.freeSlug(ctx, want)
	if err != nil {
		return store.Task{}, err
	}
	t := store.Task{Slug: slug, Dir: filepath.Join(m.cfg.TasksDir, slug), Flow: "work",
		Prompt: prompt, LinearID: strings.ToUpper(linearID.FindString(prompt))}
	if line, _, _ := strings.Cut(prompt, "\n"); line != "" {
		t.Title = line
	}
	if req.Bare {
		for _, p := range places {
			if p.repo.Name != "" {
				t.Reads = append(t.Reads, p.repo)
			}
		}
		places = nil
	}

	if err := os.MkdirAll(filepath.Join(t.Dir, stateDir), 0o755); err != nil {
		return store.Task{}, err
	}
	if err := m.store.CreateTask(ctx, &t, time.Now()); err != nil {
		return store.Task{}, err
	}

	switch len(places) {
	case 0:
		if err := m.openSession(ctx, t, prompt, false); err != nil {
			return t, err
		}
	case 1:
		// The worktree's window is the session's first, and its agent has
		// the prompt.
		if err := m.attach(ctx, t, places[0].repo, slug, prompt); err != nil {
			return t, err
		}
	default:
		// Work that spans repositories is one piece of work, so it is given
		// to one agent in one window, at the task's root, where each worktree
		// is a directory beside the others. The worktrees get no windows of
		// their own: one is a keystroke away when it is wanted.
		for _, p := range places {
			if _, err := m.checkout(ctx, t, p.repo, slug); err != nil {
				return t, err
			}
		}
		if err := m.openSession(ctx, t, prompt, false); err != nil {
			return t, err
		}
	}
	return m.store.Task(ctx, slug)
}

// Review makes a task from a pull request. For one asking for my review, that
// is a worktree on its branch and an agent started on the review. For one of
// my own, it is a worktree on its branch to carry on working in. Asked again
// for the same pull request, it returns the task it already made.
func (m *Manager) Review(ctx context.Context, repo string, number int) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, err := m.store.Tasks(ctx)
	if err != nil {
		return store.Task{}, err
	}
	for _, t := range tasks {
		for _, c := range t.Changes {
			if c.Repo == repo && c.Number == number {
				return t, m.ensure(ctx, t)
			}
		}
	}

	// One of my own is work to carry on with, not something to review.
	mine, err := m.store.Changes(ctx, store.RoleAuthor)
	if err != nil {
		return store.Task{}, err
	}
	for _, p := range mine {
		if p.Repo == repo && p.Number == number && p.State == "OPEN" {
			return m.resume(ctx, p)
		}
	}

	asking, err := m.store.Changes(ctx, store.RoleReviewer)
	if err != nil {
		return store.Task{}, err
	}
	var found *pr.PR
	for i := range asking {
		if asking[i].Repo == repo && asking[i].Number == number {
			found = &asking[i]
		}
	}
	if found == nil {
		return store.Task{}, fmt.Errorf("%s#%d is not asking for your review", repo, number)
	}
	if strings.Contains(found.Head, ":") {
		return store.Task{}, fmt.Errorf("%s#%d comes from a fork, which puffin cannot check out yet", repo, number)
	}
	r, err := m.repo(ctx, repo)
	if err != nil {
		return store.Task{}, err
	}
	slug, err := m.freeSlug(ctx, Slugify(fmt.Sprintf("review %d %s", number, found.Title)))
	if err != nil {
		return store.Task{}, err
	}

	t := store.Task{Slug: slug, Dir: filepath.Join(m.cfg.TasksDir, slug), Flow: "review",
		Title: found.Title, Prompt: m.cfg.ReviewPrompt}
	if err := os.MkdirAll(filepath.Join(t.Dir, stateDir), 0o755); err != nil {
		return store.Task{}, err
	}
	if err := m.store.CreateTask(ctx, &t, time.Now()); err != nil {
		return store.Task{}, err
	}
	if err := m.attach(ctx, t, r, found.Head, m.cfg.ReviewPrompt); err != nil {
		return t, err
	}
	return m.store.Task(ctx, slug)
}

// resume makes a task for one of my own open pull requests: a worktree on its
// branch, named for the branch, and an agent with nothing asked of it yet.
// The rest of its stack joins the task on the next poll.
func (m *Manager) resume(ctx context.Context, p pr.PR) (store.Task, error) {
	r, err := m.repo(ctx, p.Repo)
	if err != nil {
		return store.Task{}, err
	}
	slug, err := m.freeSlug(ctx, Slugify(p.Head))
	if err != nil {
		return store.Task{}, err
	}
	t := store.Task{Slug: slug, Dir: filepath.Join(m.cfg.TasksDir, slug), Flow: "work", Title: p.Title}
	if err := os.MkdirAll(filepath.Join(t.Dir, stateDir), 0o755); err != nil {
		return store.Task{}, err
	}
	if err := m.store.CreateTask(ctx, &t, time.Now()); err != nil {
		return store.Task{}, err
	}
	if err := m.attach(ctx, t, r, p.Head, ""); err != nil {
		// Nothing was made but the row and an empty directory. Take them
		// back, so the failure leaves no half task in the list.
		m.store.MarkSwept(ctx, t.ID, time.Now())
		os.RemoveAll(t.Dir)
		if strings.Contains(err.Error(), "already used by worktree") {
			return store.Task{}, fmt.Errorf("%s is checked out in another worktree; make that one the task with `puffin adopt PATH`: %w", p.Head, err)
		}
		return store.Task{}, err
	}
	return m.store.Task(ctx, slug)
}

// Attach gives a task a worktree of a repository, on a branch named for the
// task unless another is asked for.
func (m *Manager) Attach(ctx context.Context, req api.Attach) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, err := m.store.Task(ctx, req.Task)
	if err != nil {
		return store.Task{}, fmt.Errorf("%q: %w", req.Task, err)
	}
	if t.InPlace {
		return t, fmt.Errorf("%q lives in %s, which is not puffin's to put a worktree in; make a task for that with `puffin new --attach`", t.Slug, t.Dir)
	}
	r, err := m.repo(ctx, req.Repo)
	if err != nil {
		return store.Task{}, err
	}
	branch := req.Branch
	if branch == "" {
		branch = t.Slug
	}
	if err := m.ensure(ctx, t); err != nil {
		return t, err
	}
	if err := m.attach(ctx, t, r, branch, strings.TrimSpace(req.Prompt)); err != nil {
		return t, err
	}
	return m.store.Task(ctx, t.Slug)
}

// attach gives a task a worktree and a window on it, with an agent on the
// prompt.
func (m *Manager) attach(ctx context.Context, t store.Task, r store.Repo, branch, prompt string) error {
	w, err := m.checkout(ctx, t, r, branch)
	if err != nil {
		return err
	}
	step(ctx, "opening its window")
	return m.openWindow(ctx, t, w, true, prompt, false)
}

// checkout gives a task a worktree of a repository on branch, and no more.
func (m *Manager) checkout(ctx context.Context, t store.Task, r store.Repo, branch string) (store.Worktree, error) {
	// A second stack from the same repository gets a numbered directory.
	// Nothing depends on the name: the path is what is stored.
	path := filepath.Join(t.Dir, short(r.Name))
	for n := 2; ; n++ {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			break
		}
		path = filepath.Join(t.Dir, short(r.Name)+"-"+strconv.Itoa(n))
	}

	if time.Since(m.fetched[r.Path]) > baseFresh {
		step(ctx, "fetching %s", short(r.Name))
		git.FetchBase(ctx, r.Path, r.DefaultBranch)
		m.fetched[r.Path] = time.Now()
	}
	// A branch named for the task is being made now, so there is nothing of
	// it on origin to fetch. Any other name may be a branch that exists.
	if branch != t.Slug {
		step(ctx, "fetching %s", branch)
		git.FetchBranch(ctx, r.Path, branch)
	}
	step(ctx, "checking out %s in %s", branch, short(r.Name))
	if err := git.AddWorktree(ctx, r.Path, path, branch, r.DefaultBranch); err != nil {
		return store.Worktree{}, err
	}
	rc := m.cfg.Repo[r.Name]
	if len(rc.Copy) > 0 {
		step(ctx, "copying files into %s", filepath.Base(path))
	}
	if err := copyInto(ctx, r.Path, path, rc.Copy); err != nil {
		return store.Worktree{}, err
	}
	w := store.Worktree{TaskID: t.ID, Repo: r, Path: path, Ref: branch}
	return w, m.store.AddWorktree(ctx, &w, r.DefaultBranch, time.Now())
}

// copyInto copies what a fresh checkout lacks, such as untracked env files,
// from the main clone to the same place in the worktree.
func copyInto(ctx context.Context, clone, worktree string, globs []string) error {
	for _, glob := range globs {
		matches, err := filepath.Glob(filepath.Join(clone, strings.TrimSuffix(glob, "/")))
		if err != nil {
			return fmt.Errorf("copy %q: %w", glob, err)
		}
		for _, from := range matches {
			rel, err := filepath.Rel(clone, from)
			if err != nil {
				return err
			}
			to := filepath.Join(worktree, rel)
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return err
			}
			if _, err := run.In(ctx, clone, hookTimeout, "cp", "-R", from, to); err != nil {
				return err
			}
		}
	}
	return nil
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// agent is the command line that starts an agent: the prompt, read from a
// file so that no quoting of the prompt itself is ever needed, then the
// directories it may read beyond its own.
//
// The prompt goes first. A flag like claude's --add-dir takes any number of
// values, so a prompt placed after it is read as one more directory.
func (m *Manager) agent(readable []store.Repo, promptFile string, resume bool) string {
	parts := []string{m.cfg.Agent}
	if promptFile != "" {
		parts = append(parts, `"$(cat `+quote(promptFile)+`)"`)
	}
	if m.cfg.AgentDirFlag != "" {
		for _, r := range readable {
			if r.Path != "" {
				parts = append(parts, m.cfg.AgentDirFlag, quote(r.Path))
			}
		}
	}
	plain := strings.Join(parts, " ")
	// With no conversation here to carry on, the agent fails, and a fresh
	// one starts in its place.
	if resume && promptFile == "" && m.cfg.AgentResumeFlag != "" {
		return m.cfg.Agent + " " + m.cfg.AgentResumeFlag + strings.TrimPrefix(plain, m.cfg.Agent) + " || " + plain
	}
	return plain
}

// promptFile is where a prompt is kept for the agent to read as it starts. A
// task that lives in a directory it does not own keeps nothing there, so its
// prompts are kept beside the task directories instead.
func (m *Manager) promptFile(t store.Task, name string) string {
	if t.InPlace {
		return filepath.Join(m.cfg.TasksDir, ".prompts", t.Slug+"."+name+".md")
	}
	return filepath.Join(t.Dir, stateDir, name+".prompt.md")
}

func (m *Manager) savePrompt(t store.Task, name, prompt string) (string, error) {
	if prompt == "" {
		return "", nil
	}
	file := m.promptFile(t, name)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	return file, os.WriteFile(file, []byte(prompt+"\n"), 0o644)
}

func (m *Manager) bind(ctx context.Context, t store.Task) error {
	return m.tmux.SetSessionOption(ctx, t.Slug, tmux.TaskOption, strconv.FormatInt(t.ID, 10))
}

// openSession makes the session of a task that has no worktree: one window,
// with an agent at the task's root.
func (m *Manager) openSession(ctx context.Context, t store.Task, prompt string, resume bool) error {
	window := rootWindow
	if t.InPlace {
		window = filepath.Base(t.Dir)
	}
	pane, err := m.tmux.NewSession(ctx, t.Slug, window, t.Dir)
	if err != nil {
		return err
	}
	if err := m.bind(ctx, t); err != nil {
		return err
	}
	file, err := m.savePrompt(t, rootWindow, prompt)
	if err != nil {
		return err
	}
	return m.tmux.Type(ctx, pane, m.agent(t.Reads, file, resume))
}

// openWindow makes the window for a worktree, laid out as its repository
// asks. fresh says the worktree was only just made, so its setup still has to
// run. resume has its agent carry on the conversation it had there.
func (m *Manager) openWindow(ctx context.Context, t store.Task, w store.Worktree, fresh bool, prompt string, resume bool) error {
	rc := m.cfg.Repo[w.Repo.Name]
	panes := rc.Panes
	if len(panes) == 0 {
		panes = []config.Pane{{Command: agentToken, Focus: true}}
	}
	name := filepath.Base(w.Path)
	file, err := m.savePrompt(t, name, prompt)
	if err != nil {
		return err
	}
	setup := ""
	if fresh {
		setup = strings.Join(rc.Setup, " && ")
	}
	placed := false
	for _, p := range panes {
		placed = placed || strings.Contains(p.Command, setupToken)
	}

	// The task's first window is what makes its session.
	var window, pane string
	if m.tmux.HasSession(ctx, t.Slug) {
		window, pane, err = m.tmux.NewWindow(ctx, t.Slug, name, w.Path)
	} else if pane, err = m.tmux.NewSession(ctx, t.Slug, name, w.Path); err == nil {
		window = pane // a pane names its window well enough to set an option on it
		err = m.bind(ctx, t)
	}
	if err != nil {
		return err
	}
	if err := m.tmux.SetWindowOption(ctx, window, tmux.WorktreeOption, strconv.FormatInt(w.ID, 10)); err != nil {
		return err
	}
	focus := ""
	for i, p := range panes {
		if i > 0 {
			if pane, err = m.tmux.Split(ctx, pane, w.Path, p.Split == "horizontal"); err != nil {
				return err
			}
		}
		command := strings.ReplaceAll(p.Command, agentToken, m.agent(nil, file, resume))
		command = strings.TrimSpace(strings.ReplaceAll(command, setupToken, setup))
		if i == 0 && !placed && setup != "" {
			command = strings.TrimSuffix(setup+" ; "+command, " ; ")
		}
		if command != "" {
			if err := m.tmux.Type(ctx, pane, command); err != nil {
				return err
			}
		}
		if p.Focus {
			focus = pane
		}
	}
	if focus != "" {
		return m.tmux.SelectPane(ctx, focus)
	}
	return nil
}

// Ensure brings a task's session back if it has gone, as it will have after
// a reboot, with a window for each worktree. Agents start without a prompt:
// what they were asked is in their own history.
func (m *Manager) Ensure(ctx context.Context, slug string) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.store.Task(ctx, slug)
	if err != nil {
		return store.Task{}, fmt.Errorf("%q: %w", slug, err)
	}
	return t, m.ensure(ctx, t)
}

func (m *Manager) ensure(ctx context.Context, t store.Task) error {
	step(ctx, "opening %s", t.Slug)
	// A task with no worktree, or with several, is one agent at its root. A
	// task with one worktree is that worktree's window.
	several := len(t.Worktrees) > 1
	if (len(t.Worktrees) == 0 || several) && !m.tmux.HasSession(ctx, t.Slug) {
		if err := m.openSession(ctx, t, "", true); err != nil {
			return err
		}
	}
	if several {
		return nil
	}
	open, err := m.tmux.WorktreeWindows(ctx)
	if err != nil {
		return err
	}
	for _, w := range t.Worktrees {
		if _, ok := open[strconv.FormatInt(w.ID, 10)]; ok {
			continue
		}
		if err := m.openWindow(ctx, t, w, false, "", true); err != nil {
			return err
		}
	}
	return nil
}

// Reopen brings back a task that finished, or was swept, with more to do and
// no pull request for it yet. It stays open until one opens. A swept task
// gets its worktrees again, at the same paths on the same branches, made
// afresh from the default branch if they are gone, and its agents carry on
// the conversations they had there.
func (m *Manager) Reopen(ctx context.Context, slug string) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.store.Task(ctx, slug)
	swept := false
	if errors.Is(err, store.ErrNoTask) {
		if t, err = m.store.SweptTask(ctx, slug); err == nil {
			swept = true
			if m.tmux.HasSession(ctx, t.Slug) {
				return t, fmt.Errorf("%q is in use by another session", t.Slug)
			}
		}
	}
	if err != nil {
		return store.Task{}, fmt.Errorf("%q: %w", slug, err)
	}
	if t.InPlace {
		return t, fmt.Errorf("%q works in place and has nothing to reopen; switch to it", t.Slug)
	}
	step(ctx, "reopening %s", t.Slug)
	if err := m.store.Reopen(ctx, t, time.Now()); err != nil {
		return t, err
	}
	if !swept {
		return t, m.ensure(ctx, t)
	}

	gone, err := m.store.RemovedWorktrees(ctx, t.ID)
	if err != nil {
		return t, err
	}
	// A worktree removed and made again more than once is one place.
	seen := map[string]bool{}
	var made []store.Worktree
	for _, old := range gone {
		if seen[old.Path] || old.Ref == "" || old.Repo.Path == "" {
			continue
		}
		seen[old.Path] = true
		w, err := m.checkout(ctx, t, old.Repo, old.Ref)
		if err != nil {
			return t, err
		}
		made = append(made, w)
	}
	if t, err = m.store.Task(ctx, t.Slug); err != nil {
		return t, err
	}
	if len(made) == 1 {
		step(ctx, "opening its window")
		return t, m.openWindow(ctx, t, made[0], true, "", true)
	}
	if err := m.openSession(ctx, t, "", true); err != nil {
		return t, err
	}
	return t, nil
}

// Here makes a task out of a directory as it is: a session and an agent in
// it, and nothing else. The directory is not puffin's: nothing in it is made,
// moved or removed, now or when the task is closed. A directory has one such
// task, so asking for it again returns the one there is.
func (m *Manager) Here(ctx context.Context, req api.Here) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.here(ctx, req.Path, req.Name, "")
}

func (m *Manager) here(ctx context.Context, dir, name, prompt string) (store.Task, error) {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return store.Task{}, fmt.Errorf("%s is not a directory", dir)
	}
	if rel, err := filepath.Rel(m.cfg.TasksDir, dir); err == nil && !strings.HasPrefix(rel, "..") {
		return store.Task{}, fmt.Errorf("%s belongs to a task already", dir)
	}
	tasks, err := m.store.Tasks(ctx)
	if err != nil {
		return store.Task{}, err
	}
	for _, t := range tasks {
		if t.InPlace && t.Dir == dir {
			return t, m.ensure(ctx, t)
		}
	}
	want := Slugify(name)
	if want == "" {
		want = Slugify(filepath.Base(dir))
	}
	if want == "" {
		return store.Task{}, fmt.Errorf("%s has nothing in it a name can be made from; give one with --name", dir)
	}
	slug, err := m.freeSlug(ctx, want)
	if err != nil {
		return store.Task{}, err
	}
	t := store.Task{Slug: slug, Dir: dir, Flow: "work", InPlace: true, Prompt: prompt}
	if line, _, _ := strings.Cut(prompt, "\n"); line != "" {
		t.Title = line
	}
	if err := m.store.CreateTask(ctx, &t, time.Now()); err != nil {
		return store.Task{}, err
	}
	if err := m.openSession(ctx, t, prompt, false); err != nil {
		return t, err
	}
	return m.store.Task(ctx, slug)
}

// Adopt turns a worktree that was made some other way into a task. The
// worktree is moved into the task's directory as it stands, uncommitted and
// untracked files included, so nothing has to be installed again. The task
// is named for the branch unless a name is given.
func (m *Manager) Adopt(ctx context.Context, req api.Adopt) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := git.Inspect(ctx, req.Path)
	if info == nil {
		return store.Task{}, fmt.Errorf("%s is not inside a git worktree", req.Path)
	}
	if info.Branch == "" {
		return store.Task{}, fmt.Errorf("%s has no branch checked out", info.Root)
	}
	if rel, err := filepath.Rel(m.cfg.TasksDir, info.Root); err == nil && !strings.HasPrefix(rel, "..") {
		return store.Task{}, fmt.Errorf("%s already belongs to a task", info.Root)
	}
	name := git.RepoName(ctx, info.CommonDir)
	if name == "" {
		return store.Task{}, fmt.Errorf("%s has no origin to say which repository it is", info.Root)
	}
	r, err := m.repo(ctx, name)
	if err != nil {
		return store.Task{}, err
	}
	if info.Root == filepath.Dir(info.CommonDir) {
		return store.Task{}, fmt.Errorf("%s is a clone, not a worktree; a clone stays where it is", info.Root)
	}

	want := Slugify(req.Name)
	if want == "" {
		want = Slugify(info.Branch)
	}
	slug, err := m.freeSlug(ctx, want)
	if err != nil {
		return store.Task{}, err
	}
	t := store.Task{Slug: slug, Dir: filepath.Join(m.cfg.TasksDir, slug), Flow: "work"}
	if err := os.MkdirAll(filepath.Join(t.Dir, stateDir), 0o755); err != nil {
		return store.Task{}, err
	}
	path := filepath.Join(t.Dir, short(r.Name))
	// Run from the clone the worktree belongs to, which need not be the one
	// puffin makes new worktrees from.
	step(ctx, "moving %s into %s", info.Root, slug)
	if err := git.MoveWorktree(ctx, filepath.Dir(info.CommonDir), info.Root, path); err != nil {
		os.RemoveAll(t.Dir) // holds nothing but the directory just made
		return store.Task{}, err
	}
	if err := m.store.CreateTask(ctx, &t, time.Now()); err != nil {
		return store.Task{}, err
	}
	w := store.Worktree{TaskID: t.ID, Repo: r, Path: path, Ref: info.Branch}
	if err := m.store.AddWorktree(ctx, &w, r.DefaultBranch, time.Now()); err != nil {
		return t, err
	}
	if err := m.openWindow(ctx, t, w, false, "", false); err != nil {
		return t, err
	}
	return m.store.Task(ctx, slug)
}

// Rename gives a task, and its session, another name. The directory and the
// branches stay as they are: moving a directory would cost each agent in it
// its history, which is kept by path, and a branch may already be pushed.
func (m *Manager) Rename(ctx context.Context, from, to string) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.store.Task(ctx, from)
	if err != nil {
		return store.Task{}, fmt.Errorf("%q: %w", from, err)
	}
	slug := Slugify(to)
	if slug == "" {
		return t, fmt.Errorf("%q has nothing in it a name can be made from", to)
	}
	if slug == t.Slug {
		return t, nil
	}
	if taken, err := m.store.SlugTaken(ctx, slug); err != nil {
		return t, err
	} else if taken || m.tmux.HasSession(ctx, slug) {
		return t, fmt.Errorf("%q is already in use", slug)
	}
	if m.tmux.HasSession(ctx, t.Slug) {
		if err := m.tmux.RenameSession(ctx, t.Slug, slug); err != nil {
			return t, err
		}
	}
	if err := m.store.RenameTask(ctx, t.ID, slug); err != nil {
		return t, err
	}
	return m.store.Task(ctx, slug)
}

// Close removes a task: its worktrees, its session and its directory. A
// worktree with uncommitted work stops it, unless forced, and whatever was
// removed before that stays removed.
func (m *Manager) Close(ctx context.Context, slug string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.store.Task(ctx, slug)
	if err != nil {
		return fmt.Errorf("%q: %w", slug, err)
	}
	return m.close(ctx, t, force)
}

func (m *Manager) close(ctx context.Context, t store.Task, force bool) error {
	// Never trust a stored path enough to delete it without checking where
	// it points. A task that lives in a directory it does not own has
	// nothing of its own on disk, and that directory is left alone.
	owned := !t.InPlace
	if rel, err := filepath.Rel(m.cfg.TasksDir, t.Dir); owned && (err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel)) {
		return fmt.Errorf("%s is not inside %s; refusing to remove it", t.Dir, m.cfg.TasksDir)
	}

	for _, w := range t.Worktrees {
		if _, err := os.Stat(w.Path); err == nil {
			for _, hook := range m.cfg.Repo[w.Repo.Name].PreRemove {
				step(ctx, "running %s", hook)
				if _, err := run.In(ctx, w.Path, hookTimeout, "sh", "-c", hook); err != nil && !force {
					return fmt.Errorf("pre_remove in %s: %w", w.Path, err)
				}
			}
			step(ctx, "removing the %s worktree", filepath.Base(w.Path))
			if err := git.RemoveWorktree(ctx, w.Repo.Path, w.Path, force); err != nil {
				// The reason first and short: this is read in one line of the
				// picker as often as in a terminal.
				if strings.Contains(err.Error(), "modified or untracked") {
					return fmt.Errorf("uncommitted work in %s; commit it, or close with --force", filepath.Base(w.Path))
				}
				return fmt.Errorf("could not remove %s: %w", filepath.Base(w.Path), err)
			}
		}
		if err := m.store.RemoveWorktree(ctx, w.ID, time.Now()); err != nil {
			return err
		}
		// Succeeds only for a branch git considers merged.
		if w.Ref != "" {
			git.DeleteBranch(ctx, w.Repo.Path, w.Ref)
		}
	}

	// Done if it shipped anything, discarded if it never got that far.
	if t.State == store.TaskOpen {
		state := store.TaskDiscarded
		for _, c := range t.Changes {
			if c.Number != 0 {
				state = store.TaskDone
			}
		}
		if err := m.store.SetTaskState(ctx, t.ID, t.State, state, time.Now()); err != nil {
			return err
		}
	}
	if !owned {
		os.Remove(m.promptFile(t, rootWindow))
	}
	if owned {
		step(ctx, "removing %s", t.Dir)
		if err := os.RemoveAll(t.Dir); err != nil {
			return err
		}
	}
	if err := m.store.MarkSwept(ctx, t.ID, time.Now()); err != nil {
		return err
	}
	// Last, because the request may have come from inside it.
	step(ctx, "closing the session")
	if m.tmux.HasSession(ctx, t.Slug) {
		return m.tmux.KillSession(ctx, t.Slug)
	}
	return nil
}

// Sweep closes every task that is done, and reports which it closed and
// which it could not.
func (m *Manager) Sweep(ctx context.Context) (api.Swept, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := api.Swept{Closed: []string{}, Kept: map[string]string{}}
	tasks, err := m.store.Tasks(ctx)
	if err != nil {
		return out, err
	}
	for _, t := range tasks {
		if t.State != store.TaskDone {
			continue
		}
		step(ctx, "closing %s", t.Slug)
		if err := m.close(ctx, t, false); err != nil {
			out.Kept[t.Slug] = err.Error()
			continue
		}
		out.Closed = append(out.Closed, t.Slug)
	}
	return out, nil
}

// Reconcile brings the rows back in line with what is on disk and on GitHub:
// which branch each worktree is on, which pull requests each task owns, and
// which tasks have nothing left open.
func (m *Manager) Reconcile(ctx context.Context) error {
	worktrees, err := m.store.LiveWorktrees(ctx)
	if err != nil {
		return err
	}
	for _, w := range worktrees {
		info := git.Inspect(ctx, w.Path)
		if info == nil {
			continue // removed by hand; closing the task tidies the row
		}
		if err := m.store.SetWorktreeRef(ctx, w, info.Branch); err != nil {
			return err
		}
	}
	// A branch with no pull request is no longer anybody's work, and would
	// otherwise hold its task open for good, when it was renamed or deleted,
	// or when nothing has it checked out and it has nothing of its own: the
	// branch a task started on, left behind once the work moved to a stack.
	// The branch itself is left alone; only puffin stops counting it.
	local, err := m.store.LocalBranches(ctx)
	if err != nil {
		return err
	}
	checkedOut := map[string]map[string]bool{}
	for _, b := range local {
		if _, err := os.Stat(b.Clone); err != nil {
			continue
		}
		if git.BranchExists(ctx, b.Clone, b.Head) {
			if _, ok := checkedOut[b.Clone]; !ok {
				branches, err := git.CheckedOut(ctx, b.Clone)
				if err != nil {
					continue
				}
				checkedOut[b.Clone] = branches
			}
			base := b.DefaultBranch
			if base == "" {
				base = git.DefaultBranch(ctx, b.Clone)
			}
			if checkedOut[b.Clone][b.Head] || git.HasOwnCommits(ctx, b.Clone, base, b.Head) {
				continue
			}
		}
		if err := m.store.DeleteChange(ctx, b.ChangeID); err != nil {
			return err
		}
	}
	if err := m.store.LinkStacks(ctx); err != nil {
		return err
	}
	if err := m.store.ReleaseKeptOpen(ctx); err != nil {
		return err
	}
	_, err = m.store.CompleteTasks(ctx, time.Now())
	return err
}
