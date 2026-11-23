package task

import (
	"testing"

	"github.com/allanhortle/puffin/internal/pr"
	"github.com/allanhortle/puffin/internal/store"
)

func TestRank(t *testing.T) {
	open := func(status string) store.TaskChange {
		return store.TaskChange{Number: 1, State: "OPEN", Status: status}
	}
	merged := store.TaskChange{Number: 2, State: "MERGED", Status: pr.Merged}
	local := store.TaskChange{State: "LOCAL", Status: pr.Unknown}
	threads := open(pr.WaitingReview)
	threads.Unresolved = 2
	draft := open(pr.Ready)
	draft.IsDraft = true

	for name, c := range map[string]struct {
		changes []store.TaskChange
		state   string
		want    int
	}{
		"exploring":                          {nil, store.TaskOpen, Idle},
		"a branch with no pull request":      {[]store.TaskChange{local}, store.TaskOpen, Idle},
		"failing":                            {[]store.TaskChange{open(pr.Failing)}, store.TaskOpen, Mine},
		"the worst of a stack wins":          {[]store.TaskChange{open(pr.Building), open(pr.Conflict), merged}, store.TaskOpen, Mine},
		"unresolved threads are mine":        {[]store.TaskChange{threads}, store.TaskOpen, Mine},
		"ready":                              {[]store.TaskChange{open(pr.Ready), open(pr.WaitingReview)}, store.TaskOpen, Ready},
		"a green draft waits on nobody":      {[]store.TaskChange{draft}, store.TaskOpen, Idle},
		"building":                           {[]store.TaskChange{open(pr.Building), open(pr.WaitingReview)}, store.TaskOpen, Machine},
		"out for review":                     {[]store.TaskChange{open(pr.WaitingReview), merged}, store.TaskOpen, Reviewer},
		"all merged but not yet marked done": {[]store.TaskChange{merged}, store.TaskOpen, Idle},
		"done":                               {[]store.TaskChange{merged}, store.TaskDone, Finished},
	} {
		if got := Rank(store.Task{State: c.state, Changes: c.changes}); got != c.want {
			t.Errorf("%s: rank %d, want %d", name, got, c.want)
		}
	}
}
