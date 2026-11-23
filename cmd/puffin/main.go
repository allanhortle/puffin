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
)

const version = "0.0.1"

const help = `puffin: tasks, worktrees and pull requests, in tmux

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
  puffin daemon            run the daemon in the foreground
  puffin version
`

const (
	// Nothing a client waits on is allowed to wait forever.
	clientTimeout = 5 * time.Second
	// A refresh waits out a whole cycle, which has its own deadline.
	refreshTimeout = 100 * time.Second
	watchInterval  = 2 * time.Second

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
			line := paint(label.Colour, label.Letter) + " " +
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

// Keyed by tmux colour name, since that is how a label names its colour.
var ansi = map[string]string{
	"green": "32", "yellow": "33", "blue": "34", "red": "31", "magenta": "35",
	"colour242": "2", "bold": "1",
}

var colourful = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
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
