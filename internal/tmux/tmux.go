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

	"puffin/internal/git"
	"puffin/internal/run"
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
	ID     string
	Path   string // the worktree root; empty for a pinned window outside any repo
	Branch string
	Repo   string // owner/name
	Pinned bool
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
	out, err = c.Run(ctx, "list-windows", "-a", "-F", "#{window_id} #{"+PinOption+"}")
	if err == nil {
		for _, line := range lines(out) {
			id, pin, _ := strings.Cut(line, " ")
			repo, branch, ok := strings.Cut(pin, "#")
			if !ok || branch == "" || strings.Count(repo, "/") != 1 {
				continue
			}
			windows[id] = Window{ID: id, Path: windows[id].Path, Branch: branch, Repo: repo, Pinned: true}
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

// Push writes the review count and a letter for every window in ids, clearing
// the ones absent from rendered, in a single tmux invocation.
func (c Client) Push(ctx context.Context, ids []string, rendered map[string]Rendered, reviews int) error {
	count := ""
	if reviews > 0 {
		count = strconv.Itoa(reviews)
	}
	args := []string{"set", "-g", ReviewOption, count}
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
	return c.Push(ctx, ids, nil, 0)
}
