package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/allanhortle/puffin/internal/api"
	"github.com/allanhortle/puffin/internal/client"
	"github.com/allanhortle/puffin/internal/config"
	"github.com/allanhortle/puffin/internal/daemon"
	"github.com/allanhortle/puffin/internal/git"
	"github.com/allanhortle/puffin/internal/paths"
	"github.com/allanhortle/puffin/internal/pr"
	"github.com/allanhortle/puffin/internal/store"
	"github.com/allanhortle/puffin/internal/task"
	"github.com/allanhortle/puffin/internal/tmux"
)

const version = "0.0.1"

const help = `puffin: tasks, worktrees and pull requests, in tmux

  puffin new [prompt...]   make a task: a directory, a tmux session, and an agent on the prompt
      --read REPO          a repository the task's agent may read; repeatable
      --attach REPO        give the task a worktree of REPO straight away; repeatable
      --name NAME          the task's name, instead of one made from the prompt
      -e, --edit           write the prompt in $EDITOR
                           REPO can be "." for the repository you are standing in
  puffin attach REPO [prompt...]
                           give a task a worktree of REPO, with its own window and agent
      --task NAME          the task, instead of the one this tmux session belongs to
      --branch BRANCH      the branch, instead of one named for the task
      -e, --edit           write the prompt in $EDITOR
  puffin pick              one list of your tasks and the reviews asked of you, whatever
                           waits on you first. enter goes to it, starting a review if it is
                           one; ctrl-o opens the pull request; ctrl-r refetches; ctrl-s sweeps
  puffin pr N              make, or return to, the task for a pull request: a worktree on its
                           branch. one of yours to carry on with, or one to review, with an
                           agent started on the review. OWNER/REPO#N if N alone is ambiguous
  puffin here [PATH]       make a task that lives in a directory as it is, here or at PATH:
                           a session and an agent, no worktree, no pull request to follow.
                           for a clone you work on directly. closing it leaves the directory
      --name NAME          the task's name, instead of the directory's
  puffin adopt [PATH]      make a task out of an existing worktree, here or at PATH, by
                           moving it into the task as it stands
      --name NAME          the task's name, instead of the branch's
  puffin tasks [NAME]      every task, or one, with its worktrees and pull requests; --json
  puffin switch NAME       go to a task's session, bringing it back if it has gone
  puffin rename [NAME] NEW give a task and its session another name; the directory and
                           branches keep theirs
  puffin close [NAME]      remove a task's worktrees, session and directory
      --force              even if a worktree has uncommitted work
  puffin sweep             close every task whose pull requests have all merged or closed
  puffin repos             the repositories found under the configured roots; --scan to look again

  puffin start             start the daemon in the background, if it is not running
  puffin stop              stop the daemon and clear the letters
  puffin restart
  puffin toggle            stop it if it is running, start it if it is not
  puffin status            daemon health, last fetch, resolved windows; --json for the raw reply
  puffin refresh           one poll cycle now; --force refetches even if nothing moved
  puffin watch [target]    follow one pull request; target is a number, branch or worktree
  puffin prs               every pull request the daemon holds; --json for other tools to read
  puffin reviews           the pull requests waiting on your review
  puffin open <target>     a pull request number, or "reviews", in the browser
  puffin agent STATE       for an agent's hooks: say what the agent in this pane is doing.
                           working, waiting, done, or off
  puffin daemon            run the daemon in the foreground
  puffin version
`

const (
	// Nothing a client waits on is allowed to wait forever.
	clientTimeout = 5 * time.Second
	// A refresh waits out a whole cycle, which has its own deadline.
	refreshTimeout = 100 * time.Second
	// Making or removing a worktree takes as long as git takes.
	taskTimeout   = 16 * time.Minute
	watchInterval = 2 * time.Second

	// Where a click lands: the pull request itself for a letter, and the
	// dashboard for the review count, which has no single pull request behind
	// it.
	graphiteURL = "https://app.graphite.com"
	// What the tmux config names the review count's click range, and so what
	// arrives as the argument to `open` when it is clicked. A letter's range
	// carries a number, so a word cannot collide with one.
	reviewTarget = "reviews"
)

func main() {
	command := "help"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	args := os.Args[min(2, len(os.Args)):]
	flag := func(name string) bool { return len(args) > 0 && args[0] == name }
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}

	var err error
	switch command {
	case "daemon":
		err = runDaemon()
	case "start":
		err = start()
	case "stop":
		err = stop()
	case "restart":
		if err = stop(); err == nil {
			err = start()
		}
	case "toggle":
		if _, down := ping(); down == nil {
			err = stop()
		} else {
			err = start()
		}
	case "new":
		err = newTask(args)
	case "attach":
		err = attach(args)
	case "tasks":
		err = tasks(args)
	case "pick":
		err = pick(args)
	case "review", "pr":
		err = reviewCommand(arg)
	case "switch":
		err = switchTo(arg)
	case "close":
		err = closeTask(args)
	case "agent":
		agent(arg) // never an error: it runs inside the agent's own hooks
	case "here":
		err = inPlace(args)
	case "adopt":
		err = adopt(args)
	case "rename":
		err = rename(args)
	case "sweep":
		err = sweep()
	case "repos":
		err = repos(flag("--scan"))
	case "status":
		err = status(flag("--json"))
	case "refresh":
		err = refresh(flag("--force"))
	case "prs":
		err = prs(flag("--json"))
	case "reviews":
		err = reviews()
	case "open":
		err = open(arg)
	case "watch":
		err = watch(arg)
	case "version", "--version":
		fmt.Println(version)
	case "help", "--help", "-h":
		fmt.Print(help)
	default:
		fmt.Fprintf(os.Stderr, "puffin: unknown command %q\n\n%s", command, help)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "puffin:", err)
		os.Exit(1)
	}
}

func daemonClient() *client.Client { return client.New(paths.Socket()) }

func timeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

func ping() (api.Status, error) {
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	return daemonClient().Status(ctx)
}

func runDaemon() error {
	cfg, err := config.Load(paths.Config())
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return daemon.Run(ctx, daemon.Options{
		Version: version,
		Config:  cfg,
		DB:      paths.DB(),
		Socket:  paths.Socket(),
		Lock:    paths.Lock(),
	})
}

// start is idempotent, so a tmux config reload can call it freely.
func start() error {
	if _, err := ping(); err == nil {
		return nil
	}
	// One that holds the lock without answering is on its way out. Wait for
	// it, or the new daemon would find the lock taken and give up.
	for patience := time.Now().Add(clientTimeout); daemon.Running(paths.Lock()) && time.Now().Before(patience); {
		time.Sleep(20 * time.Millisecond)
		if _, err := ping(); err == nil {
			return nil
		}
	}
	// Fail here, where the message reaches a terminal, rather than in the log.
	if _, err := config.Load(paths.Config()); err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(paths.Log()), 0o755); err != nil {
		return err
	}
	log, err := os.OpenFile(paths.Log(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer log.Close()

	cmd := exec.Command(self, "daemon")
	cmd.Stdout, cmd.Stderr = log, log
	// Its own session, so it outlives the terminal or tmux hook that started it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	deadline := time.After(clientTimeout)
	for {
		select {
		case <-exited:
			return fmt.Errorf("the daemon exited as it started; see %s", paths.Log())
		case <-deadline:
			return fmt.Errorf("the daemon did not answer within %s; see %s", clientTimeout, paths.Log())
		case <-time.After(50 * time.Millisecond):
			if _, err := ping(); err == nil {
				return nil
			}
		}
	}
}

// stop returns once the daemon has gone, so that a start straight after it
// does not find the old one still holding the lock.
func stop() error {
	ctx, cancel := timeout(2 * clientTimeout)
	defer cancel()
	err := daemonClient().Shutdown(ctx)
	if err != nil && !errors.Is(err, client.ErrNotRunning) {
		return err
	}
	for daemon.Running(paths.Lock()) {
		select {
		case <-ctx.Done():
			return errors.New("the daemon was asked to stop and has not")
		case <-time.After(20 * time.Millisecond):
		}
	}
	return nil
}

func ago(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return time.Since(*t).Round(time.Second).String() + " ago"
}

func status(asJSON bool) error {
	s, err := ping()
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	}

	poll := s.Poll
	fmt.Printf("daemon:     %s\n", paint("green", fmt.Sprintf("running (pid %d, %s)", s.PID, s.Version)))
	fmt.Printf("last cycle: %s\n", ago(poll.LastCycleAt))
	if poll.LastError != "" {
		fmt.Printf("last error: %s\n", paint("red", poll.LastError))
	}
	fmt.Printf("last fetch: %s\n", ago(poll.LastFetchAt))
	rate := "unknown"
	if poll.RateRemaining != nil {
		rate = fmt.Sprintf("%d left", *poll.RateRemaining)
	}
	fmt.Printf("rate limit: %s\n", rate)
	fmt.Printf("prs:        %d open, %d recently closed\n", poll.Open, poll.Closed)
	fmt.Printf("to review:  %d\n", poll.Reviews)
	fmt.Printf("tasks:      %d open, %d worktrees\n", s.Counts.OpenTasks, s.Counts.Worktrees)
	fmt.Printf("database:   %s (schema %d)\n", s.DB, s.SchemaVersion)

	for _, w := range poll.Windows {
		if w.PR == 0 {
			fmt.Printf("  %s %s → %s\n", w.ID, w.Branch, paint("colour242", "no pr"))
			continue
		}
		parts := []string{fmt.Sprintf("#%d %s", w.PR, w.Status)}
		if w.Pinned {
			parts = append(parts, "pinned")
		}
		if len(w.Stack) > 1 {
			parts = append(parts, fmt.Sprintf("stack of %d", len(w.Stack)))
		}
		if w.Bottom != w.PR {
			parts = append(parts, fmt.Sprintf("letter from bottom #%d %s", w.Bottom, w.BottomStatus))
		}
		if w.Unresolved > 0 {
			parts = append(parts, fmt.Sprintf("%d unresolved on the bottom", w.Unresolved))
		}
		fmt.Printf("  %s %s → %s\n", w.ID, w.Branch, strings.Join(parts, ", "))
	}
	return nil
}

func refresh(force bool) error {
	ctx, cancel := timeout(refreshTimeout)
	defer cancel()
	cycle, err := daemonClient().Refresh(ctx, force)
	if err != nil {
		return err
	}
	suffix := ""
	if cycle.Refetched {
		suffix = " (refetched)"
	}
	fmt.Printf("%d prs, %d windows, %d to review%s\n", cycle.PRs, cycle.Windows, cycle.Reviews, suffix)
	return nil
}

// prs lists my own pull requests, the ones needing something first. With
// --json it is the interface other tools read instead of a state file.
func prs(asJSON bool) error {
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	all, err := daemonClient().PRs(ctx)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(all)
	}
	for _, open := range []bool{true, false} {
		for _, p := range all.PRs {
			if (p.State == "OPEN") != open {
				continue
			}
			label := p.Label()
			line := paint(label.Colour, fmt.Sprintf("%-8s", label.Word)) + " " +
				paint("colour242", fmt.Sprintf("%s#%d", p.Repo, p.Number)) + " " + p.Title
			if p.Unresolved > 0 {
				line += " " + paint("red", fmt.Sprintf("•%d", p.Unresolved))
			}
			fmt.Println(line)
		}
	}
	return nil
}

// reviews is what the count on the status bar is counting, so that it has
// something to act on rather than only nagging.
func reviews() error {
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	r, err := daemonClient().Reviews(ctx)
	if err != nil {
		return err
	}
	if r.Count == 0 {
		fmt.Println("nothing waiting on you")
		return nil
	}
	for _, p := range r.PRs {
		line := paint("yellow", fmt.Sprintf("#%d", p.Number)) + " " + paint("colour242", p.Repo)
		if p.Author != "" {
			line += " " + paint("blue", "@"+p.Author)
		}
		fmt.Println(line, p.Title)
	}
	// The count is the whole total, the list only the first page of it.
	if r.Count > len(r.PRs) {
		fmt.Println(paint("colour242", fmt.Sprintf("… and %d more", r.Count-len(r.PRs))))
	}
	return nil
}

func prURL(p pr.PR) string {
	return fmt.Sprintf("%s/github/pr/%s/%d", graphiteURL, p.Repo, p.Number)
}

// open is where a click on the status bar lands, with whatever tmux read out
// of the range under the pointer.
func open(target string) error {
	if target == "" {
		return errors.New("open needs a pull request number, or \"reviews\"")
	}
	link := graphiteURL
	if target != reviewTarget {
		ctx, cancel := timeout(clientTimeout)
		defer cancel()
		found, err := daemonClient().PR(ctx, url.Values{"target": {target}})
		if err != nil {
			return err
		}
		link = prURL(found.PR)
	}
	return exec.Command("open", link).Run()
}

// here says which pull request to look for when nobody named one: the one
// for the branch checked out in the current directory.
func here(ctx context.Context) (url.Values, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	info := git.Inspect(ctx, dir)
	if info == nil || info.Branch == "" {
		return nil, errors.New("not on a branch")
	}
	return url.Values{"repo": {git.RepoName(ctx, info.CommonDir)}, "branch": {info.Branch}}, nil
}

func watch(target string) error {
	c := daemonClient()
	ask := func(q url.Values) (api.PR, error) {
		ctx, cancel := timeout(clientTimeout)
		defer cancel()
		return c.PR(ctx, q)
	}

	query := url.Values{"target": {target}}
	if target == "" {
		ctx, cancel := timeout(clientTimeout)
		q, err := here(ctx)
		cancel()
		if err != nil {
			return err
		}
		query = q
	}
	found, err := ask(query)
	if err != nil {
		return err
	}
	p := found.PR
	// From here on, by the one thing about it that cannot change.
	query = url.Values{"repo": {p.Repo}, "number": {strconv.Itoa(p.Number)}}

	fmt.Println(paint("bold", p.Title))
	heading := p.Head + " → " + p.Base
	if p.IsDraft {
		heading += "  (draft)"
	}
	fmt.Println(paint("colour242", heading))
	fmt.Println(paint("colour242", p.URL))
	fmt.Println(paint("colour242", prURL(p)))

	const bell = "\a"
	clock := func() string { return paint("colour242", time.Now().Format("15:04:05")) }
	var printed *time.Time
	previous := ""
	for {
		if found.FetchedAt != nil && (printed == nil || !printed.Equal(*found.FetchedAt)) {
			printed = found.FetchedAt
			p = found.PR
			fmt.Println(clock(), watchLine(p, found.Stack))
			if previous != "" && previous != p.Status {
				fmt.Print(bell)
			}
			previous = p.Status
			if p.State != "OPEN" {
				fmt.Printf("\nPR is %s, exiting.\n%s", paint(pr.LabelOf(p.State).Colour, p.State), bell)
				return nil
			}
		}
		time.Sleep(watchInterval)
		if found, err = ask(query); err != nil {
			fmt.Println(clock(), paint("yellow", err.Error()+", exiting")+bell)
			return nil
		}
	}
}

func watchLine(p pr.PR, stack []int) string {
	checks := p.Checks
	var parts []string
	switch {
	case checks.Fail > 0:
		parts = append(parts, paint("red", fmt.Sprintf("⨯%d", checks.Fail)))
	case checks.Pending > 0:
		parts = append(parts, paint("yellow", fmt.Sprintf("%2d", checks.Pending))+":"+
			paint("green", strconv.Itoa(checks.Pass)))
	default:
		parts = append(parts, paint("green", strconv.Itoa(checks.Pass)))
	}

	threads := "green"
	if p.Unresolved > 0 {
		threads = "red"
	}
	parts = append(parts, paint(threads, fmt.Sprintf("•%d", p.Unresolved)))

	// The colour is read from the same table as the tab's letter, so this
	// view cannot end up disagreeing with it about whose move it is.
	status := p.Status
	if p.IsInMergeQueue && p.QueuePosition != nil {
		status = fmt.Sprintf("%s#%d", pr.Queued, *p.QueuePosition)
	}
	parts = append(parts, paint(pr.LabelOf(p.Status).Colour, status))

	if len(stack) > 1 {
		numbers := make([]string, len(stack))
		for i, n := range stack {
			numbers[i] = strconv.Itoa(n)
			if n == p.Number {
				numbers[i] = "[" + numbers[i] + "]"
			}
		}
		parts = append(parts, paint("colour242", "stack "+strings.Join(numbers, " ")))
	}
	if len(checks.Failing) > 0 {
		parts = append(parts, paint("red", strings.Join(checks.Failing, ", ")))
	}
	if len(checks.AwaitingOn) > 0 {
		parts = append(parts, paint("red", "needs you: "+strings.Join(checks.AwaitingOn, ", ")))
	}
	return strings.Join(parts, " ")
}

// parse splits arguments into flags and the rest. A flag named in valued
// takes the argument after it; any other flag is a switch. Everything after
// `--` is the rest.
func parse(args []string, valued ...string) (map[string][]string, []string, error) {
	takes := map[string]bool{}
	for _, name := range valued {
		takes[name] = true
	}
	flags := map[string][]string{}
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return flags, append(rest, args[i+1:]...), nil
		case !strings.HasPrefix(a, "-") || a == "-":
			rest = append(rest, a)
		case takes[a]:
			if i+1 == len(args) {
				return nil, nil, fmt.Errorf("%s needs a value", a)
			}
			i++
			flags[a] = append(flags[a], args[i])
		default:
			flags[a] = append(flags[a], "")
		}
	}
	return flags, rest, nil
}

func last(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

func tmuxClient() (tmux.Client, error) {
	cfg, err := config.Load(paths.Config())
	return tmux.Client{Socket: cfg.TmuxSocket}, err
}

// goTo moves this terminal to a task's session when it is inside tmux, and
// says how to get there when it is not.
func goTo(slug string) error {
	if os.Getenv("TMUX") == "" {
		fmt.Printf("%s is ready: tmux attach -t %s\n", slug, slug)
		return nil
	}
	tm, err := tmuxClient()
	if err != nil {
		return err
	}
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	return tm.Switch(ctx, slug)
}

// currentTask is the task this terminal's tmux session belongs to: a task's
// session carries the task's name.
func currentTask() (string, error) {
	if os.Getenv("TMUX") == "" {
		return "", errors.New("not inside tmux, so name the task")
	}
	tm, err := tmuxClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	return tm.CurrentSession(ctx)
}

func wantsEditor(flags map[string][]string) bool {
	_, short := flags["-e"]
	_, long := flags["--edit"]
	return short || long
}

// edit opens the editor on a prompt and returns what was saved. A long
// prompt, or one pasted from somewhere, has no business on a command line.
func edit(start string) (string, error) {
	file, err := os.CreateTemp("", "puffin-prompt-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(start); err != nil {
		return "", err
	}
	file.Close()

	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	// Through the shell, because $EDITOR is allowed to carry arguments.
	cmd := exec.Command("sh", "-c", editor+` "$1"`, "sh", file.Name())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", editor, err)
	}
	saved, err := os.ReadFile(file.Name())
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(saved))
	if prompt == "" {
		return "", errors.New("the prompt was empty, so nothing was made")
	}
	return prompt, nil
}

// repoNames turns "." into the repository the current directory belongs to,
// so a task can be started from wherever the work is without naming it.
func repoNames(names []string) ([]string, error) {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = name
		if name != "." {
			continue
		}
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		ctx, cancel := timeout(clientTimeout)
		info := git.Inspect(ctx, dir)
		if info != nil {
			out[i] = git.RepoName(ctx, info.CommonDir)
		}
		cancel()
		if info == nil || out[i] == "" {
			return nil, errors.New("\".\" needs to be inside a clone that has an origin")
		}
	}
	return out, nil
}

func newTask(args []string) error {
	flags, rest, err := parse(args, "--read", "--attach", "--name")
	if err != nil {
		return err
	}
	prompt := strings.Join(rest, " ")
	if wantsEditor(flags) {
		if prompt, err = edit(prompt); err != nil {
			return err
		}
	}
	read, err := repoNames(flags["--read"])
	if err != nil {
		return err
	}
	attached, err := repoNames(flags["--attach"])
	if err != nil {
		return err
	}
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	t, err := daemonClient().NewTask(ctx, api.NewTask{
		Name:   last(flags["--name"]),
		Prompt: prompt,
		Read:   read,
		Attach: attached,
	})
	if err != nil {
		return err
	}
	return goTo(t.Slug)
}

func attach(args []string) error {
	flags, rest, err := parse(args, "--task", "--branch")
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return errors.New("attach needs a repository")
	}
	names, err := repoNames(rest[:1])
	if err != nil {
		return err
	}
	prompt := strings.Join(rest[1:], " ")
	if wantsEditor(flags) {
		if prompt, err = edit(prompt); err != nil {
			return err
		}
	}
	task := last(flags["--task"])
	if task == "" {
		if task, err = currentTask(); err != nil {
			return err
		}
	}
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	t, err := daemonClient().Attach(ctx, api.Attach{
		Task:   task,
		Repo:   names[0],
		Branch: last(flags["--branch"]),
		Prompt: prompt,
	})
	if err != nil {
		return err
	}
	w := t.Worktrees[len(t.Worktrees)-1]
	fmt.Printf("%s: %s on %s\n", t.Slug, w.Path, w.Ref)
	return nil
}

var taskColours = map[string]string{store.TaskOpen: "green", store.TaskDone: "magenta", store.TaskDiscarded: "colour242"}

// ranked is every task, the ones waiting on me first.
func ranked() ([]store.Task, error) {
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	all, err := daemonClient().Tasks(ctx)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool { return task.Rank(all[i]) < task.Rank(all[j]) })
	return all, nil
}

func tasks(args []string) error {
	flags, rest, err := parse(args)
	if err != nil {
		return err
	}
	all, err := ranked()
	if err != nil {
		return err
	}
	if _, asJSON := flags["--json"]; asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(all)
	}
	if len(rest) > 0 {
		for _, t := range all {
			if t.Slug == rest[0] {
				printTask(t, true)
				return nil
			}
		}
		return fmt.Errorf("%q: %w", rest[0], store.ErrNoTask)
	}
	for _, t := range all {
		printTask(t, false)
	}
	return nil
}

func changeLabel(c store.TaskChange) pr.Label {
	return pr.PR{Status: c.Status, IsDraft: c.IsDraft}.Label()
}

func printTask(t store.Task, detail bool) {
	line := paint(taskColours[t.State], fmt.Sprintf("%-5s", t.State)) + " " + paint("bold", t.Slug)
	if t.Title != "" && t.Title != t.Slug {
		line += " " + paint("colour242", t.Title)
	}
	fmt.Println(line)
	switch {
	case t.InPlace:
		fmt.Println("      " + paint("colour242", "in place, "+t.Dir))
	case len(t.Worktrees) == 0:
		fmt.Println("      " + paint("colour242", "exploring, no worktree"))
	}
	for _, w := range t.Worktrees {
		ref := w.Ref
		if ref == "" {
			ref = "detached"
		}
		fmt.Printf("      %s %s\n", filepath.Base(w.Path), paint("colour242", w.Repo.Name+" on "+ref))
	}
	for _, c := range t.Changes {
		if c.Number == 0 {
			fmt.Printf("      %s\n", paint("colour242", fmt.Sprintf("%-*s %s", wordWidth, "local", c.Head)))
			continue
		}
		label := changeLabel(c)
		line := fmt.Sprintf("      %s #%d %s", paint(label.Colour, fmt.Sprintf("%-*s", wordWidth, label.Word)),
			c.Number, paint("colour242", c.Head))
		if c.Unresolved > 0 {
			line += " " + paint("red", fmt.Sprintf("•%d", c.Unresolved))
		}
		fmt.Println(line)
	}
	if !detail {
		return
	}
	for _, r := range t.Reads {
		fmt.Println("      " + paint("colour242", "reads "+r.Name))
	}
	fmt.Println("      " + paint("colour242", t.Dir))
	if t.Prompt != "" {
		fmt.Println()
		fmt.Println(t.Prompt)
	}
}

const (
	taskRow   = "task"
	reviewRow = "review"
)

func column(s string, shown, width int) string {
	return s + strings.Repeat(" ", max(width-shown, 1))
}

// The longest status word, and the whole status column: the word, a space,
// two for threads, a space, three for the rest of the stack.
const (
	wordWidth    = 8
	lettersWidth = wordWidth + 1 + 2 + 1 + 3
)

// pickLine is one task as the list shows it. The first field says what Enter
// should act on, and is not displayed.
//
// A task shows one letter however many pull requests it has: the most
// pressing, which is the one that decides where the task sorts, then how many
// more there are. A stack of eight is one piece of work, not eight letters.
func pickLine(t store.Task) string {
	live := task.Live(t)
	if t.State != store.TaskOpen {
		live = t.Changes // a finished task shows how it ended
	}
	// With no pull request there is still something to say about a task:
	// where it lives, or that it has not got as far as one.
	word, colour := "explore", "colour242"
	if t.InPlace {
		word = "here"
	}
	threads, more := "", ""
	if len(live) > 0 {
		first := task.Headline(live)
		word = "local"
		if first.Number != 0 {
			label := changeLabel(first)
			word, colour = label.Word, label.Colour
			if first.Unresolved > 0 {
				threads = strconv.Itoa(first.Unresolved)
			}
		}
		if len(live) > 1 {
			more = "+" + strconv.Itoa(len(live)-1)
		}
	}
	// Three columns of fixed width, so every row's name starts in the same
	// place: the word, its unresolved threads, the rest of its stack.
	letters := paint(colour, fmt.Sprintf("%-*s", wordWidth, word)) + " " +
		paint("red", fmt.Sprintf("%-2s", threads)) + " " + paint("colour242", fmt.Sprintf("%-3s", more))
	shown := lettersWidth
	name := t.Slug
	if t.State != store.TaskOpen {
		name += " (" + t.State + ")"
	}
	title := t.Title
	if title == t.Slug {
		title = ""
	}
	return strings.Join([]string{taskRow + " " + t.Slug,
		marker(task.Rank(t) == task.Mine) + column(letters, shown, lettersWidth) + agentIcon(t.Agent) + " " + paint("bold", name),
		paint("colour242", title)}, "\t")
}

// marker flags a row that is waiting on me. It is the same mark as the count
// in the status bar, and the count is the number of rows that carry it.
func marker(mine bool) string {
	if mine {
		return paint("red", "▲") + " "
	}
	return "  "
}

func agentIcon(state string) string {
	if icon, ok := task.Icons[state]; ok {
		return paint(icon.Colour, icon.Glyph)
	}
	return " "
}

// reviewLine is a pull request asking for my review that has no task yet.
func reviewLine(p pr.PR) string {
	name := fmt.Sprintf("%s#%d", p.Repo[strings.Index(p.Repo, "/")+1:], p.Number)
	return strings.Join([]string{fmt.Sprintf("%s %s %d", reviewRow, p.Repo, p.Number),
		marker(true) + column(paint("red", "review"), len("review"), lettersWidth) + "  " + name, paint("colour242", p.Title+" @"+p.Author)}, "\t")
}

// pickLines is the one list: my tasks and the reviews asked of me, whatever
// is waiting on me first. A review request counts as waiting on me, and
// follows the tasks that are.
func pickLines() ([]string, error) {
	all, err := ranked()
	if err != nil {
		return nil, err
	}
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	asked, err := daemonClient().Reviews(ctx)
	if err != nil {
		return nil, err
	}
	tasked := map[string]bool{}
	for _, t := range all {
		for _, c := range t.Changes {
			tasked[fmt.Sprintf("%s#%d", c.Repo, c.Number)] = true
		}
	}
	var lines []string
	reviews := func() {
		for _, p := range asked.PRs {
			if !tasked[fmt.Sprintf("%s#%d", p.Repo, p.Number)] {
				lines = append(lines, reviewLine(p))
			}
		}
	}
	placed := false
	for _, t := range all {
		if !placed && task.Rank(t) > task.Mine {
			reviews()
			placed = true
		}
		lines = append(lines, pickLine(t))
	}
	if !placed {
		reviews()
	}
	return lines, nil
}

// pick is the one list, in a finder. Enter goes to a task, or starts the
// review of a pull request and goes to that. ctrl-o opens the pull request
// in the browser instead, and ctrl-r asks GitHub again.
func pick(args []string) error {
	flags, _, err := parse(args)
	if err != nil {
		return err
	}
	colourful = os.Getenv("NO_COLOR") == "" // fzf reads the colours, whatever stdout is
	lines, err := pickLines()
	if err != nil {
		return err
	}
	if _, list := flags["--list"]; list {
		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
	fzf := []string{"--ansi", "--no-sort", "--reverse", "--delimiter", "\t", "--with-nth", "2..",
		"--expect", "ctrl-o",
		"--bind", "ctrl-r:reload(" + q(self) + " refresh >/dev/null 2>&1; " + q(self) + " pick --list)",
		// What the sweep closed, and what it kept and why, becomes the header:
		// a sweep that silently did nothing would look like a key that failed.
		"--bind", "ctrl-s:transform-header(" + q(self) + " sweep 2>&1)+reload(" + q(self) + " pick --list)"}
	if len(lines) == 0 {
		fzf = append(fzf, "--header", "nothing open. start with: puffin new [--attach REPO] what it is for")
	}
	cmd := exec.Command("fzf", fzf...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 130 || exit.ExitCode() == 1) {
		return nil // dismissed, or nothing matched
	}
	if err != nil {
		return fmt.Errorf("fzf: %w", err)
	}
	key, chosen, _ := strings.Cut(strings.TrimRight(string(out), "\n"), "\n")
	id, _, _ := strings.Cut(chosen, "\t")
	fields := strings.Fields(id)
	browser := key == "ctrl-o"

	switch {
	case len(fields) == 2 && fields[0] == taskRow:
		if !browser {
			return switchTo(fields[1])
		}
		ctx, cancel := timeout(clientTimeout)
		defer cancel()
		all, err := daemonClient().Tasks(ctx)
		if err != nil {
			return err
		}
		for _, t := range all {
			for _, c := range t.Changes {
				if t.Slug == fields[1] && c.Number != 0 && c.State == "OPEN" {
					return exec.Command("open", prURL(pr.PR{Repo: c.Repo, Number: c.Number})).Run()
				}
			}
		}
		return fmt.Errorf("%s has no open pull request", fields[1])
	case len(fields) == 3 && fields[0] == reviewRow:
		number, _ := strconv.Atoi(fields[2])
		if browser {
			return exec.Command("open", prURL(pr.PR{Repo: fields[1], Number: number})).Run()
		}
		return review(fields[1], number)
	}
	return nil
}

// review starts, or returns to, the review of a pull request.
func review(repo string, number int) error {
	fmt.Printf("checking out %s#%d…\n", repo, number) // a large checkout says nothing while it works
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	t, err := daemonClient().Review(ctx, repo, number)
	if err != nil {
		return err
	}
	return goTo(t.Slug)
}

func reviewCommand(target string) error {
	repo, n, named := strings.Cut(target, "#")
	if !named {
		repo, n = "", target
	}
	number, err := strconv.Atoi(n)
	if err != nil || (named && strings.Count(repo, "/") != 1) {
		return errors.New("give a pull request as a number, or as owner/repo#number")
	}
	if !named {
		// A number alone: ask the daemon which repository it means.
		ctx, cancel := timeout(clientTimeout)
		found, err := daemonClient().PR(ctx, url.Values{"target": {n}})
		cancel()
		if err != nil {
			return err
		}
		repo = found.PR.Repo
	}
	return review(repo, number)
}

func switchTo(slug string) error {
	if slug == "" {
		return errors.New("switch needs a task")
	}
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	if _, err := daemonClient().Ensure(ctx, slug); err != nil {
		return err
	}
	return goTo(slug)
}

func closeTask(args []string) error {
	flags, rest, err := parse(args)
	if err != nil {
		return err
	}
	slug := ""
	if len(rest) > 0 {
		slug = rest[0]
	} else if slug, err = currentTask(); err != nil {
		return err
	}
	_, force := flags["--force"]
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	return daemonClient().Close(ctx, slug, force)
}

// agent records what the agent in this pane is doing, for the agent's own
// hooks to call. It must never get in the agent's way, so it is quick,
// silent, and does nothing at all outside tmux.
func agent(state string) {
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		return
	}
	switch state {
	case task.AgentWorking, task.AgentWaiting, task.AgentDone:
	case "off":
		state = ""
	default:
		return
	}
	tm, err := tmuxClient()
	if err != nil {
		return
	}
	ctx, cancel := timeout(2 * time.Second)
	defer cancel()
	// The daemon turns this into what the session shows beside its name, and
	// recounts what is waiting on me.
	if tm.SetAgent(ctx, pane, state) == nil {
		daemonClient().Recount(ctx)
	}
}

func inPlace(args []string) error {
	flags, rest, err := parse(args, "--name")
	if err != nil {
		return err
	}
	path := "."
	if len(rest) > 0 {
		path = rest[0]
	}
	if path, err = filepath.Abs(path); err != nil {
		return err
	}
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	t, err := daemonClient().Here(ctx, api.Here{Path: path, Name: last(flags["--name"])})
	if err != nil {
		return err
	}
	return goTo(t.Slug)
}

func adopt(args []string) error {
	flags, rest, err := parse(args, "--name")
	if err != nil {
		return err
	}
	path := "."
	if len(rest) > 0 {
		path = rest[0]
	}
	if path, err = filepath.Abs(path); err != nil {
		return err
	}
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	t, err := daemonClient().Adopt(ctx, api.Adopt{Path: path, Name: last(flags["--name"])})
	if err != nil {
		return err
	}
	return goTo(t.Slug)
}

func rename(args []string) error {
	var from, to string
	var err error
	switch len(args) {
	case 1:
		to = args[0]
		if from, err = currentTask(); err != nil {
			return err
		}
	case 2:
		from, to = args[0], args[1]
	default:
		return errors.New("rename needs a new name, and the task unless you are in its session")
	}
	ctx, cancel := timeout(clientTimeout)
	defer cancel()
	t, err := daemonClient().Rename(ctx, from, to)
	if err != nil {
		return err
	}
	fmt.Println(t.Slug)
	return nil
}

func sweep() error {
	ctx, cancel := timeout(taskTimeout)
	defer cancel()
	swept, err := daemonClient().Sweep(ctx)
	if err != nil {
		return err
	}
	for _, slug := range swept.Closed {
		fmt.Println("closed", slug)
	}
	for slug, why := range swept.Kept {
		fmt.Println(paint("yellow", "kept "+slug+": ") + why)
	}
	if len(swept.Closed)+len(swept.Kept) == 0 {
		fmt.Println("nothing is done")
	}
	return nil
}

func repos(scan bool) error {
	ctx, cancel := timeout(refreshTimeout)
	defer cancel()
	found, err := daemonClient().Repos(ctx, scan)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Printf("none known; set roots in %s and run `puffin repos --scan`\n", paths.Config())
	}
	for _, r := range found {
		fmt.Printf("%s %s\n", r.Name, paint("colour242", r.Path+" ("+r.DefaultBranch+")"))
	}
	return nil
}

// Keyed by tmux colour name, since that is how a label names its colour.
var ansi = map[string]string{
	"green": "32", "yellow": "33", "blue": "34", "red": "31", "magenta": "35",
	"colour242": "2", "bold": "1",
}

var colourful = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}()

func paint(colour, s string) string {
	code, ok := ansi[colour]
	if !ok || !colourful {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}
