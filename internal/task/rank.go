package task

import (
	"context"
	"strconv"

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
	// An agent that has stopped to ask something is waiting on me, whatever
	// the pull requests say.
	if t.Agent == AgentWaiting {
		return Mine
	}
	live := Live(t)
	if len(live) == 0 {
		return Idle // exploring, or open with more to do than what has merged
	}
	return RankChange(Headline(live))
}

// What an agent can report itself to be.
const (
	AgentWorking = "working"
	AgentWaiting = "waiting" // stopped to ask for permission, or a question
	AgentDone    = "done"    // finished its turn
)

// Tasks is every live task, with what its agents are doing.
func (m *Manager) Tasks(ctx context.Context) ([]store.Task, error) {
	tasks, err := m.store.Tasks(ctx)
	if err != nil {
		return nil, err
	}
	agents, err := m.tmux.Agents(ctx)
	if err != nil {
		return tasks, nil // the list is still worth having without them
	}
	// One task can have several agents. The one that needs me speaks for it,
	// then one still working, then one that has finished.
	order := []string{AgentWaiting, AgentWorking, AgentDone}
	for i := range tasks {
		states := agents[tasks[i].Slug]
		for _, want := range order {
			for _, state := range states {
				if state == want && tasks[i].Agent == "" {
					tasks[i].Agent = state
				}
			}
		}
	}
	return tasks, nil
}

// Attention is how many things are waiting on me: tasks, and requests for my
// review that have no task yet. It is the number of marked rows in the list.
func (m *Manager) Attention(ctx context.Context) (int, error) {
	tasks, err := m.Tasks(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, t := range tasks {
		if Rank(t) == Mine {
			n++
		}
	}
	asking, err := m.store.Changes(ctx, store.RoleReviewer)
	if err != nil {
		return n, err
	}
	tasked := map[string]bool{}
	for _, t := range tasks {
		for _, c := range t.Changes {
			tasked[c.Repo+"#"+strconv.Itoa(c.Number)] = true
		}
	}
	for _, p := range asking {
		if p.Status == pr.ReviewRequested && !tasked[p.Repo+"#"+strconv.Itoa(p.Number)] {
			n++
		}
	}
	return n, nil
}

// ShowAttention recounts and updates the status bar now.
func (m *Manager) ShowAttention(ctx context.Context) error {
	n, err := m.Attention(ctx)
	if err != nil {
		return err
	}
	return m.tmux.SetAttention(ctx, n)
}
