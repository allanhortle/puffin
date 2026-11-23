package task

import (
	"context"

	"github.com/allanhortle/puffin/internal/pr"
	"github.com/allanhortle/puffin/internal/store"
)

// Whose move a task is, most pressing first. It is the order of the picker,
// and Mine is what the status bar counts.
const (
	Mine     = iota // something of mine is red: failing, conflicted, sent back, or waiting on me
	Ready           // a pull request could merge
	Idle            // nothing is waiting on anyone: exploring, or no pull request yet
	Machine         // checks are running, or it is in the merge queue
	Reviewer        // out for review
	Finished        // every pull request merged or closed; ready to sweep
)

// RankChange is how pressing one pull request, or branch, is.
func RankChange(c store.TaskChange) int {
	switch {
	case c.State == "MERGED" || c.State == "CLOSED":
		return Finished
	case c.Unresolved > 0:
		return Mine
	}
	switch c.Status {
	case pr.Failing, pr.Conflict, pr.ChangesRequested, pr.WaitingMe, pr.ReviewRequested, pr.ReviewUpdated:
		return Mine
	case pr.Reviewed:
		return Reviewer // back with its author, as my own is when out for review
	case pr.Ready:
		// A draft that is green is waiting on nobody until I say it is ready.
		if c.IsDraft {
			return Idle
		}
		return Ready
	case pr.Building, pr.Queued:
		return Machine
	case pr.WaitingReview:
		return Reviewer
	}
	return Idle
}

// Live is what a task is still waiting on: its pull requests that have not
// merged or closed. A branch with no pull request counts only when there is
// nothing else, because next to real pull requests it is usually the branch
// the task started on, left behind once the work was split into a stack.
func Live(t store.Task) []store.TaskChange {
	var prs, local []store.TaskChange
	for _, c := range t.Changes {
		switch {
		case c.State == "MERGED" || c.State == "CLOSED":
		case c.Number == 0:
			local = append(local, c)
		default:
			prs = append(prs, c)
		}
	}
	if len(prs) > 0 {
		return prs
	}
	return local
}

// Headline is the most pressing of a task's live changes: the one that
// decides where it sorts, and the one worth a letter.
func Headline(live []store.TaskChange) store.TaskChange {
	first := live[0]
	for _, c := range live[1:] {
		if RankChange(c) < RankChange(first) {
			first = c
		}
	}
	return first
}

// Rank is how pressing a task is.
func Rank(t store.Task) int {
	if t.State != store.TaskOpen {
		return Finished
	}
	live := Live(t)
	if len(live) == 0 {
		return Idle // exploring, or open with more to do than what has merged
	}
	return RankChange(Headline(live))
}

// Attention is how many tasks are waiting on me.
func (m *Manager) Attention(ctx context.Context) (int, error) {
	tasks, err := m.store.Tasks(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, t := range tasks {
		if Rank(t) == Mine {
			n++
		}
	}
	return n, nil
}
