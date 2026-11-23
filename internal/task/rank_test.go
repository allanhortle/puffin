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
		"a leftover branch does not outrank the stack": {[]store.TaskChange{local, open(pr.Building), open(pr.Building)}, store.TaskOpen, Machine},
		"done": {[]store.TaskChange{merged}, store.TaskDone, Finished},
	} {
		if got := Rank(store.Task{State: c.state, Changes: c.changes}); got != c.want {
			t.Errorf("%s: rank %d, want %d", name, got, c.want)
		}
	}
}

func TestAgentChangesWhoseMoveItIs(t *testing.T) {
	review := []store.TaskChange{{Number: 1, State: "OPEN", Status: pr.ReviewRequested}}
	updated := []store.TaskChange{{Number: 1, State: "OPEN", Status: pr.ReviewUpdated}}
	failing := []store.TaskChange{{Number: 1, State: "OPEN", Status: pr.Failing}}
	waiting := []store.TaskChange{{Number: 1, State: "OPEN", Status: pr.WaitingReview}}
	for name, c := range map[string]struct {
		changes []store.TaskChange
		agent   string
		want    int
	}{
		"a review nobody has started":           {review, "", Mine},
		"a review an agent is doing":            {review, AgentWorking, Machine},
		"a pushed-since review an agent is on":  {updated, AgentWorking, Machine},
		"a review whose agent has finished":     {review, AgentDone, Mine},
		"a review whose agent is asking":        {review, AgentWaiting, Mine},
		"my failing pull request, agent or not": {failing, AgentWorking, Mine},
		"an agent asking outranks a quiet task": {waiting, AgentWaiting, Mine},
	} {
		got := Rank(store.Task{State: store.TaskOpen, Changes: c.changes, Agent: c.agent})
		if got != c.want {
			t.Errorf("%s: rank %d, want %d", name, got, c.want)
		}
	}
}
