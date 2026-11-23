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

func rankChange(c store.TaskChange) int {
	switch {
	case c.State == "MERGED" || c.State == "CLOSED":
		return Finished
	case c.Unresolved > 0:
		return Mine
	}
	switch c.Status {
	case pr.Failing, pr.Conflict, pr.ChangesRequested, pr.WaitingMe:
		return Mine
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

// Rank is the most pressing thing among a task's pull requests.
func Rank(t store.Task) int {
	if t.State != store.TaskOpen {
		return Finished
	}
	rank := Finished
	live := false
	for _, c := range t.Changes {
		if r := rankChange(c); r < rank {
			rank = r
		}
		live = live || (c.State != "MERGED" && c.State != "CLOSED")
	}
	if !live {
		return Idle // still open, so there is more to do than what has merged
	}
	return rank
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
