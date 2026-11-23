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

type Manager struct {
	cfg   config.Config
	store *store.Store
	tmux  tmux.Client

	// One change to tasks at a time. They are rare, and each one touches the
	// database, the filesystem, git and tmux in turn.
	mu sync.Mutex
}

func New(cfg config.Config, st *store.Store, tm tmux.Client) *Manager {
	return &Manager{cfg: cfg, store: st, tmux: tm}
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

// New makes a task and its session. With nothing attached, the root window
// runs an agent on the prompt. With something attached, the prompt goes to
// the agent in the first worktree instead and the root window is a shell.
func (m *Manager) New(ctx context.Context, req api.NewTask) (store.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prompt := strings.TrimSpace(req.Prompt)
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

	// Resolve everything that can be refused before anything is made.
	t := store.Task{Slug: slug, Dir: filepath.Join(m.cfg.TasksDir, slug), Flow: "work",
		Prompt: prompt, LinearID: strings.ToUpper(linearID.FindString(prompt))}
	if line, _, _ := strings.Cut(prompt, "\n"); line != "" {
		t.Title = line
	}
	for _, name := range req.Read {
		r, err := m.repo(ctx, name)
		if err != nil {
			return store.Task{}, err
		}
		t.Reads = append(t.Reads, r)
	}
	var attach []store.Repo
	for _, name := range req.Attach {
		r, err := m.repo(ctx, name)
		if err != nil {
			return store.Task{}, err
		}
		attach = append(attach, r)
	}

	if err := os.MkdirAll(filepath.Join(t.Dir, stateDir), 0o755); err != nil {
		return store.Task{}, err
	}
	if err := m.store.CreateTask(ctx, &t, time.Now()); err != nil {
		return store.Task{}, err
	}

	rootPrompt := prompt
	if len(attach) > 0 {
		rootPrompt = ""
	}
	if err := m.openSession(ctx, t, len(attach) == 0, rootPrompt); err != nil {
		return t, err
	}
	for i, r := range attach {
		p := ""
		if i == 0 {
			p = prompt
		}
		if err := m.attach(ctx, t, r, slug, p); err != nil {
			return t, err
		}
	}
	return m.store.Task(ctx, slug)
}

// Review makes a task for reviewing a pull request that is asking for my
// review: a worktree on its branch, and an agent started on the review. Asked
// again for the same pull request, it returns the task it already made.
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
	if err := m.openSession(ctx, t, false, ""); err != nil {
		return t, err
	}
	if err := m.attach(ctx, t, r, found.Head, m.cfg.ReviewPrompt); err != nil {
		return t, err
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

func (m *Manager) attach(ctx context.Context, t store.Task, r store.Repo, branch, prompt string) error {
	// A second stack from the same repository gets a numbered directory.
	// Nothing depends on the name: the path is what is stored.
	path := filepath.Join(t.Dir, short(r.Name))
	for n := 2; ; n++ {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			break
		}
		path = filepath.Join(t.Dir, short(r.Name)+"-"+strconv.Itoa(n))
	}

	if err := git.AddWorktree(ctx, r.Path, path, branch, r.DefaultBranch); err != nil {
		return err
	}
	rc := m.cfg.Repo[r.Name]
	if err := copyInto(ctx, r.Path, path, rc.Copy); err != nil {
		return err
	}
	w := store.Worktree{TaskID: t.ID, Repo: r, Path: path, Ref: branch}
	if err := m.store.AddWorktree(ctx, &w, r.DefaultBranch, time.Now()); err != nil {
		return err
	}
	return m.openWindow(ctx, t, w, true, prompt)
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
func (m *Manager) agent(readable []store.Repo, promptFile string) string {
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
	return strings.Join(parts, " ")
}

func savePrompt(t store.Task, name, prompt string) (string, error) {
	if prompt == "" {
		return "", nil
	}
	file := filepath.Join(t.Dir, stateDir, name+".prompt.md")
	return file, os.WriteFile(file, []byte(prompt+"\n"), 0o644)
}

// openSession makes the task's session, with its root window.
func (m *Manager) openSession(ctx context.Context, t store.Task, withAgent bool, prompt string) error {
	pane, err := m.tmux.NewSession(ctx, t.Slug, rootWindow, t.Dir)
	if err != nil {
		return err
	}
	if err := m.tmux.SetSessionOption(ctx, t.Slug, tmux.TaskOption, strconv.FormatInt(t.ID, 10)); err != nil {
		return err
	}
	if !withAgent {
		return nil
	}
	file, err := savePrompt(t, rootWindow, prompt)
	if err != nil {
		return err
	}
	return m.tmux.Type(ctx, pane, m.agent(t.Reads, file))
}

// openWindow makes the window for a worktree, laid out as its repository
// asks. fresh says the worktree was only just made, so its setup still has to
// run.
func (m *Manager) openWindow(ctx context.Context, t store.Task, w store.Worktree, fresh bool, prompt string) error {
	rc := m.cfg.Repo[w.Repo.Name]
	panes := rc.Panes
	if len(panes) == 0 {
		panes = []config.Pane{{Command: agentToken, Focus: true}}
	}
	name := filepath.Base(w.Path)
	file, err := savePrompt(t, name, prompt)
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

	window, pane, err := m.tmux.NewWindow(ctx, t.Slug, name, w.Path)
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
		command := strings.ReplaceAll(p.Command, agentToken, m.agent(nil, file))
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
	if !m.tmux.HasSession(ctx, t.Slug) {
		if err := m.openSession(ctx, t, len(t.Worktrees) == 0, ""); err != nil {
			return err
		}
	}
	open, err := m.tmux.WorktreeWindows(ctx)
	if err != nil {
		return err
	}
	for _, w := range t.Worktrees {
		if _, ok := open[strconv.FormatInt(w.ID, 10)]; ok {
			continue
		}
		if err := m.openWindow(ctx, t, w, false, ""); err != nil {
			return err
		}
	}
	return nil
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
	// it points.
	if rel, err := filepath.Rel(m.cfg.TasksDir, t.Dir); err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("%s is not inside %s; refusing to remove it", t.Dir, m.cfg.TasksDir)
	}

	for _, w := range t.Worktrees {
		if _, err := os.Stat(w.Path); err == nil {
			for _, hook := range m.cfg.Repo[w.Repo.Name].PreRemove {
				if _, err := run.In(ctx, w.Path, hookTimeout, "sh", "-c", hook); err != nil && !force {
					return fmt.Errorf("pre_remove in %s: %w", w.Path, err)
				}
			}
			if err := git.RemoveWorktree(ctx, w.Repo.Path, w.Path, force); err != nil {
				return fmt.Errorf("%s has work in it; commit it, or close with --force: %w", w.Path, err)
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
	if err := os.RemoveAll(t.Dir); err != nil {
		return err
	}
	if err := m.store.MarkSwept(ctx, t.ID, time.Now()); err != nil {
		return err
	}
	// Last, because the request may have come from inside it.
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
	// A branch that was renamed or deleted before it got a pull request is
	// no longer anybody's work, and would otherwise hold its task open for
	// good.
	local, err := m.store.LocalBranches(ctx)
	if err != nil {
		return err
	}
	for _, b := range local {
		if _, err := os.Stat(b.Clone); err != nil || git.BranchExists(ctx, b.Clone, b.Head) {
			continue
		}
		if err := m.store.DeleteChange(ctx, b.ChangeID); err != nil {
			return err
		}
	}
	if err := m.store.LinkStacks(ctx); err != nil {
		return err
	}
	_, err = m.store.CompleteTasks(ctx, time.Now())
	return err
}
