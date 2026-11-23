// Package tmux reads what is open and writes what the status bar shows.
//
// For now a window is matched to a pull request by the directory its panes
// are in, as prstatusd did. Once puffin makes its own sessions, a window will
// carry the id of what it shows and this lookup goes away.
package tmux

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/allanhortle/puffin/internal/git"
	"github.com/allanhortle/puffin/internal/run"
)

// Options puffin writes. The letter and the number travel separately because
// tmux only honours a #[range=] marker written literally in the format option,
// but it will expand a format inside the marker's argument: the number rides
// in its own option and the config turns the letter into a clickable range.
const (
	LetterOption = "@pr"
	NumberOption = "@prnum"
	// How many pull requests are waiting on my review. Server-scoped, because
	// unlike the letter it belongs to no one window. A bare number with no
	// styling: a letter's colour is data, whereas this is always the same
	// colour, which makes it the config's business.
	ReviewOption = "@prreview"
	// How many tasks are waiting on me. Server-scoped and bare, like the
	// review count.
	AttentionOption = "@puffin_attention"
	// Read, never written: `owner/repo#branch`, set on a window by hand to say
	// which pull request it shows.
	PinOption = "@prpin"
)

// ErrNoServer means there is no tmux server to talk to.
var ErrNoServer = errors.New("no tmux server running")

var noServer = regexp.MustCompile(`(?i)no server running|no such file or directory|error connecting to`)

type Client struct {
	// Passed to tmux as -L when set. Empty means whichever server tmux would
	// pick on its own.
	Socket string
}

func (c Client) Run(ctx context.Context, args ...string) (string, error) {
	if c.Socket != "" {
		args = append([]string{"-L", c.Socket}, args...)
	}
	out, err := run.Output(ctx, "tmux", args...)
	var failed *run.Error
	if errors.As(err, &failed) && noServer.MatchString(failed.Stderr) {
		return "", ErrNoServer
	}
	return out, err
}

func lines(out string) []string {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// Attached reports whether anyone is looking.
func (c Client) Attached(ctx context.Context) (bool, error) {
	out, err := c.Run(ctx, "list-clients", "-F", "#{client_name}")
	if err != nil {
		return false, err
	}
	return len(lines(out)) > 0, nil
}

func (c Client) WindowIDs(ctx context.Context) ([]string, error) {
	out, err := c.Run(ctx, "list-windows", "-a", "-F", "#{window_id}")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

type Window struct {
	ID      string
	Session string
	Path    string // the worktree root; empty for a pinned window outside any repo
	Branch  string
	Repo    string // owner/name
	Pinned  bool
}

// Windows is every window that shows a branch, ordered by id.
func (c Client) Windows(ctx context.Context) ([]Window, error) {
	// Space separated and split from the left, because a path may contain
	// spaces and nothing else in the line can.
	out, err := c.Run(ctx, "list-panes", "-a", "-F", "#{window_id} #{pane_active} #{pane_current_path}")
	if err != nil {
		return nil, err
	}
	var order []string
	panes := map[string][]string{}
	for _, line := range lines(out) {
		id, rest, _ := strings.Cut(line, " ")
		active, dir, _ := strings.Cut(rest, " ")
		if id == "" || dir == "" {
			continue
		}
		if _, seen := panes[id]; !seen {
			order = append(order, id)
		}
		// The active pane defines the window's worktree. If it has wandered
		// somewhere without a repo, any other pane that has one will do.
		if active == "1" {
			panes[id] = append([]string{dir}, panes[id]...)
		} else {
			panes[id] = append(panes[id], dir)
		}
	}

	infos := map[string]*git.Info{}
	repos := map[string]string{}
	windows := map[string]Window{}
	for _, id := range order {
		for _, dir := range panes[id] {
			info, seen := infos[dir]
			if !seen {
				info = git.Inspect(ctx, dir)
				infos[dir] = info
			}
			if info == nil || info.Branch == "" {
				continue
			}
			repo, seen := repos[info.CommonDir]
			if !seen {
				repo = git.RepoName(ctx, info.CommonDir)
				repos[info.CommonDir] = repo
			}
			windows[id] = Window{ID: id, Path: info.Root, Branch: info.Branch, Repo: repo}
			break
		}
	}

	// A session that starts in one repo can open its pull request in another,
	// and no pane ever moves there, so the directory has nothing to say about
	// it. The pin outranks the directory.
	out, err = c.Run(ctx, "list-windows", "-a", "-F", "#{window_id}\t#{session_name}\t#{"+PinOption+"}")
	if err == nil {
		for _, line := range lines(out) {
			id, rest, _ := strings.Cut(line, "\t")
			session, pin, _ := strings.Cut(rest, "\t")
			if w, ok := windows[id]; ok {
				w.Session = session
				windows[id] = w
			}
			repo, branch, ok := strings.Cut(pin, "#")
			if !ok || branch == "" || strings.Count(repo, "/") != 1 {
				continue
			}
			windows[id] = Window{ID: id, Session: session, Path: windows[id].Path, Branch: branch, Repo: repo, Pinned: true}
		}
	}

	list := make([]Window, 0, len(windows))
	for _, w := range windows {
		list = append(list, w)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list, nil
}

// Rendered is what one window's tab shows.
type Rendered struct {
	Letter string // with its tmux styling
	Number int
}

// A count is shown only when there is something to count.
func count(n int) string {
	if n > 0 {
		return strconv.Itoa(n)
	}
	return ""
}

// Push writes the two counts and a letter for every window in ids, clearing
// the ones absent from rendered, in a single tmux invocation.
func (c Client) Push(ctx context.Context, ids []string, rendered map[string]Rendered, reviews, attention int) error {
	args := []string{"set", "-g", ReviewOption, count(reviews), ";", "set", "-g", AttentionOption, count(attention)}
	for _, id := range ids {
		if r, ok := rendered[id]; ok {
			args = append(args, ";", "set", "-w", "-t", id, LetterOption, r.Letter)
			args = append(args, ";", "set", "-w", "-t", id, NumberOption, strconv.Itoa(r.Number))
		} else {
			args = append(args, ";", "set", "-uw", "-t", id, LetterOption)
			args = append(args, ";", "set", "-uw", "-t", id, NumberOption)
		}
	}
	if _, err := c.Run(ctx, args...); err != nil {
		return err
	}
	// On its own and unchecked: with nobody attached there is no client to
	// redraw, tmux calls that an error, and the options are set regardless.
	c.Run(ctx, "refresh-client", "-S")
	return nil
}

// Clear takes every letter and the review count off the bar.
func (c Client) Clear(ctx context.Context) error {
	ids, err := c.WindowIDs(ctx)
	if err != nil {
		return err
	}
	return c.Push(ctx, ids, nil, 0, 0)
}

// Options that bind tmux to puffin's rows. Live tmux state is never stored:
// these are how a session or window says what it is showing.
const (
	TaskOption = "@puffin_task"
	// What the agent in a window is doing: working, waiting or done. Set by
	// the agent's own hooks, and read back live; it is never stored.
	AgentOption = "@puffin_agent"
	// What a session shows in front of its name, already styled: its task's
	// pull request letter, then the most pressing of its windows' agents.
	StatusOption   = "@puffin_status"
	WorktreeOption = "@puffin_worktree"
)

// exact stops tmux matching a session name by prefix.
func exact(session string) string { return "=" + session }

func (c Client) HasSession(ctx context.Context, name string) bool {
	_, err := c.Run(ctx, "has-session", "-t", exact(name))
	return err == nil
}

// NewSession makes a detached session with one window and returns that
// window's pane.
func (c Client) NewSession(ctx context.Context, name, window, dir string) (string, error) {
	out, err := c.Run(ctx, "new-session", "-d", "-s", name, "-n", window, "-c", dir, "-P", "-F", "#{pane_id}")
	return strings.TrimSpace(out), err
}

func (c Client) RenameSession(ctx context.Context, from, to string) error {
	_, err := c.Run(ctx, "rename-session", "-t", exact(from), to)
	return err
}

// KillSession ends a session. Anyone looking at it is moved to the session
// they were in before, or failing that to any other, because tmux by default
// detaches a client whose session is destroyed, and closing a task should
// not throw its user out of tmux.
func (c Client) KillSession(ctx context.Context, name string) error {
	if out, err := c.Run(ctx, "list-clients", "-t", exact(name), "-F", "#{client_name}"); err == nil {
		for _, client := range lines(out) {
			if _, err := c.Run(ctx, "switch-client", "-c", client, "-l"); err != nil {
				c.Run(ctx, "switch-client", "-c", client, "-n")
			}
		}
	}
	_, err := c.Run(ctx, "kill-session", "-t", exact(name))
	return err
}

func (c Client) SetSessionOption(ctx context.Context, session, key, value string) error {
	// The target is a pane, so the trailing colon is what makes this name a
	// session rather than a window to look for.
	_, err := c.Run(ctx, "set-option", "-t", exact(session)+":", key, value)
	return err
}

func (c Client) SetWindowOption(ctx context.Context, window, key, value string) error {
	_, err := c.Run(ctx, "set-option", "-w", "-t", window, key, value)
	return err
}

// NewWindow adds a window to a session without switching to it, and returns
// the window and its pane.
func (c Client) NewWindow(ctx context.Context, session, name, dir string) (window, pane string, err error) {
	out, err := c.Run(ctx, "new-window", "-d", "-t", exact(session)+":", "-n", name, "-c", dir,
		"-P", "-F", "#{window_id} #{pane_id}")
	window, pane, _ = strings.Cut(strings.TrimSpace(out), " ")
	return window, pane, err
}

// Split divides a pane and returns the new one: side by side when horizontal,
// stacked otherwise.
func (c Client) Split(ctx context.Context, pane, dir string, horizontal bool) (string, error) {
	direction := "-v"
	if horizontal {
		direction = "-h"
	}
	out, err := c.Run(ctx, "split-window", "-d", direction, "-t", pane, "-c", dir, "-P", "-F", "#{pane_id}")
	return strings.TrimSpace(out), err
}

// Type enters a command at a pane's shell. Typed rather than given to tmux as
// the pane's command, so the shell is still there when the command exits and
// it runs with everything the shell's startup files set up.
func (c Client) Type(ctx context.Context, pane, command string) error {
	if _, err := c.Run(ctx, "send-keys", "-t", pane, "-l", command); err != nil {
		return err
	}
	_, err := c.Run(ctx, "send-keys", "-t", pane, "Enter")
	return err
}

func (c Client) SelectPane(ctx context.Context, pane string) error {
	_, err := c.Run(ctx, "select-pane", "-t", pane)
	return err
}

// Switch moves the current client to a session.
func (c Client) Switch(ctx context.Context, session string) error {
	_, err := c.Run(ctx, "switch-client", "-t", exact(session))
	return err
}

// CurrentSession is the name of the session the caller is running inside.
func (c Client) CurrentSession(ctx context.Context) (string, error) {
	out, err := c.Run(ctx, "display-message", "-p", "#{session_name}")
	return strings.TrimSpace(out), err
}

// WorktreeWindows maps the value of each window's worktree option to the
// window carrying it.
func (c Client) WorktreeWindows(ctx context.Context) (map[string]string, error) {
	out, err := c.Run(ctx, "list-windows", "-a", "-F", "#{window_id} #{"+WorktreeOption+"}")
	if errors.Is(err, ErrNoServer) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	found := map[string]string{}
	for _, line := range lines(out) {
		if window, value, ok := strings.Cut(line, " "); ok && value != "" {
			found[value] = window
		}
	}
	return found, nil
}

// SetAgent records what the agent in a pane's window is doing. An empty
// state clears it.
func (c Client) SetAgent(ctx context.Context, pane, state string) error {
	args := []string{"set-option", "-w", "-t", pane, AgentOption, state}
	if state == "" {
		args = []string{"set-option", "-uw", "-t", pane, AgentOption}
	}
	_, err := c.Run(ctx, args...)
	return err
}

// SetStatus sets what a session shows in front of its name. Empty clears it.
func (c Client) SetStatus(ctx context.Context, session, styled string) error {
	args := []string{"set-option", "-t", exact(session) + ":", StatusOption, styled}
	if styled == "" {
		args = []string{"set-option", "-u", "-t", exact(session) + ":", StatusOption}
	}
	if _, err := c.Run(ctx, args...); err != nil {
		return err
	}
	c.Run(ctx, "refresh-client", "-S")
	return nil
}

// Statuses is what every session currently shows for its agents, including
// the sessions that show nothing.
func (c Client) Statuses(ctx context.Context) (map[string]string, error) {
	out, err := c.Run(ctx, "list-sessions", "-F", "#{session_name}\t#{"+StatusOption+"}")
	if errors.Is(err, ErrNoServer) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	found := map[string]string{}
	for _, line := range lines(out) {
		session, status, _ := strings.Cut(line, "\t")
		found[session] = status
	}
	return found, nil
}

// Agents is the state of every agent that has reported one, by session.
func (c Client) Agents(ctx context.Context) (map[string][]string, error) {
	out, err := c.Run(ctx, "list-windows", "-a", "-F", "#{"+AgentOption+"} #{session_name}")
	if errors.Is(err, ErrNoServer) {
		return map[string][]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	found := map[string][]string{}
	for _, line := range lines(out) {
		if state, session, ok := strings.Cut(line, " "); ok && state != "" {
			found[session] = append(found[session], state)
		}
	}
	return found, nil
}

// SetAttention shows how many tasks are waiting on me, without waiting for
// the next poll.
func (c Client) SetAttention(ctx context.Context, n int) error {
	if _, err := c.Run(ctx, "set", "-g", AttentionOption, count(n)); err != nil {
		return err
	}
	c.Run(ctx, "refresh-client", "-S")
	return nil
}
