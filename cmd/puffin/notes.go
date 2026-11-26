package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/allanhortle/puffin/internal/github"
	"github.com/allanhortle/puffin/internal/pr"
)

const (
	// hunk keeps a note only as long as its window, so they are read out
	// while it runs. One written in the last moment before quitting is lost.
	noteInterval = time.Second
	noteTimeout  = 5 * time.Second
	postTimeout  = 2 * time.Minute
)

// hunkNote is a note of mine as `hunk session comment list` gives it. The
// range is of the new file, or of the old one for a note on a removed line.
type hunkNote struct {
	FilePath string `json:"filePath"`
	NewRange []int  `json:"newRange"`
	OldRange []int  `json:"oldRange"`
	Body     string `json:"body"`
}

func (n hunkNote) note() (github.Note, bool) {
	out := github.Note{Path: n.FilePath, Body: n.Body}
	lines := n.NewRange
	if len(lines) == 0 {
		lines, out.Old = n.OldRange, true
	}
	if len(lines) == 0 {
		return out, false
	}
	out.StartLine, out.Line = lines[0], lines[len(lines)-1]
	return out, true
}

// hunk shows the diff of two commits, or trees, in hunk, and returns the
// notes I had written in it when it closed.
func hunk(dir, from, to string) ([]github.Note, error) {
	show := exec.Command("hunk", "diff", from, to)
	show.Dir = dir
	show.Stdin, show.Stdout, show.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := show.Start(); err != nil {
		return nil, err
	}

	var mu sync.Mutex
	var seen []hunkNote
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		session := ""
		tick := time.NewTicker(noteInterval)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
			}
			if session == "" {
				if session = hunkSession(show.Process.Pid, dir); session == "" {
					continue
				}
			}
			var reply struct {
				Comments []hunkNote `json:"comments"`
			}
			// A failure is hunk closing under the question: what was read
			// before it stands.
			if hunkJSON(&reply, "session", "comment", "list", session, "--type", "user", "--json") == nil {
				mu.Lock()
				seen = reply.Comments
				mu.Unlock()
			}
		}
	}()
	err := show.Wait()
	close(done)
	wg.Wait()

	var notes []github.Note
	for _, n := range seen {
		if note, ok := n.note(); ok {
			notes = append(notes, note)
		}
	}
	return notes, err
}

func hunkJSON(out any, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), noteTimeout)
	defer cancel()
	raw, err := exec.CommandContext(ctx, "hunk", args...).Output()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// hunkSession is the session of the hunk this process started: the one with
// its pid, or failing that the newest one in the same clone, since hunk may
// be started through a wrapper.
func hunkSession(pid int, dir string) string {
	var reply struct {
		Sessions []struct {
			SessionID  string    `json:"sessionId"`
			Pid        int       `json:"pid"`
			RepoRoot   string    `json:"repoRoot"`
			LaunchedAt time.Time `json:"launchedAt"`
		} `json:"sessions"`
	}
	if hunkJSON(&reply, "session", "list", "--json") != nil {
		return ""
	}
	found, newest := "", time.Time{}
	for _, s := range reply.Sessions {
		if s.Pid == pid {
			return s.SessionID
		}
		if s.RepoRoot == dir && s.LaunchedAt.After(newest) {
			found, newest = s.SessionID, s.LaunchedAt
		}
	}
	return found
}

func noteLine(n github.Note) string {
	where := fmt.Sprintf("%s:%d", n.Path, n.Line)
	if n.Old {
		where += " (removed)"
	}
	return paint("colour242", where) + "  " + strings.ReplaceAll(n.Body, "\n", "\n    ")
}

// postNotes offers to put the notes on the pull request as a pending
// review: a draft that only I see until I submit it. Declined, or failed,
// they are printed, since hunk has already forgotten them.
func postNotes(h github.History, notes []github.Note, oldLines bool) error {
	for _, n := range notes {
		fmt.Println(noteLine(n))
	}
	fmt.Printf("\nadd %s to your pending review of %s#%d? [y/N] ", plural(len(notes), "note", "notes"), h.Repo, h.Number)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		return nil
	}

	ctx, cancel := timeout(postTimeout)
	defer cancel()
	gh := github.New()
	var failed []string
	onFile := 0
	for _, n := range notes {
		onLine, err := gh.Comment(ctx, h.ID, n, oldLines || !n.Old)
		switch {
		case err != nil:
			failed = append(failed, fmt.Sprintf("%s: %v", noteLine(n), err))
		case !onLine:
			onFile++
		}
	}
	if posted := len(notes) - len(failed); posted > 0 {
		fmt.Printf("added %d, pending until you submit the review: %s\n", posted, prURL(pr.PR{Repo: h.Repo, Number: h.Number}))
	}
	if onFile > 0 {
		fmt.Printf("%d went on the file and not the line: github only takes a line the pull request's whole diff shows\n", onFile)
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d not added:\n%s", len(failed), strings.Join(failed, "\n"))
	}
	return nil
}

// approve approves a pull request, and with it submits whatever notes are
// waiting in my pending review.
func approve(args []string) error {
	flags, rest, err := parse(args, "-m", "--message")
	if err != nil {
		return err
	}
	target := ""
	if len(rest) > 0 {
		target = rest[0]
	}
	ctx, cancel := timeout(postTimeout)
	defer cancel()
	repo, number, _, err := readTarget(ctx, target)
	if err != nil {
		return err
	}
	body := last(append(flags["-m"], flags["--message"]...))
	done, err := github.New().Approve(ctx, repo, number, body)
	if err != nil {
		return err
	}
	fmt.Printf("approved %s#%d at %s: %s\n", repo, number, short(done.Head), done.Title)
	if done.Notes > 0 {
		fmt.Printf("%s went out with it\n", plural(done.Notes, "pending note", "pending notes"))
	}
	// So the list stops asking for the review now and not at the next poll.
	daemonClient().Refresh(ctx, true)
	return nil
}
