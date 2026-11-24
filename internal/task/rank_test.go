package task

import (
	"strings"
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
	building := open(pr.Building)
	building.Unresolved = 2
	draft := open(pr.Ready)
	draft.IsDraft = true

	for name, c := range map[string]struct {
		changes []store.TaskChange
		state   string
		want    int
	}{
		"exploring":                                    {nil, store.TaskOpen, Idle},
		"a branch with no pull request":                {[]store.TaskChange{local}, store.TaskOpen, Idle},
		"failing":                                      {[]store.TaskChange{open(pr.Failing)}, store.TaskOpen, Mine},
		"the worst of a stack wins":                    {[]store.TaskChange{open(pr.Building), open(pr.Conflict), merged}, store.TaskOpen, Mine},
		"threads wait while it builds":                 {[]store.TaskChange{building}, store.TaskOpen, Machine},
		"unresolved threads are mine":                  {[]store.TaskChange{threads}, store.TaskOpen, Mine},
		"ready":                                        {[]store.TaskChange{open(pr.Ready), open(pr.WaitingReview)}, store.TaskOpen, Ready},
		"a green draft waits on nobody":                {[]store.TaskChange{draft}, store.TaskOpen, Idle},
		"building":                                     {[]store.TaskChange{open(pr.Building), open(pr.WaitingReview)}, store.TaskOpen, Machine},
		"out for review":                               {[]store.TaskChange{open(pr.WaitingReview), merged}, store.TaskOpen, Reviewer},
		"all merged but not yet marked done":           {[]store.TaskChange{merged}, store.TaskOpen, Idle},
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

func TestOnlyTheBottomOfAStackSpeaks(t *testing.T) {
	pull := func(number int, head, base, status string) store.TaskChange {
		return store.TaskChange{Repo: "o/r", Number: number, Head: head, Base: base, State: "OPEN", Status: status}
	}
	stack := []store.TaskChange{
		pull(3, "c", "b", pr.Failing),
		pull(1, "a", "main", pr.Building),
		pull(2, "b", "a", pr.Conflict),
	}
	task := store.Task{State: store.TaskOpen, Changes: stack}
	if got := Headline(Live(task)); got.Number != 1 {
		t.Errorf("headline is #%d, want the bottom", got.Number)
	}
	if got := Rank(task); got != Machine {
		t.Errorf("rank %d: red higher up the stack leaked through", got)
	}

	// Once the bottom merges, the next one up is the bottom.
	stack[1].State, stack[1].Status = "MERGED", pr.Merged
	if got := Headline(Live(task)); got.Number != 2 {
		t.Errorf("after the bottom merged, headline is #%d", got.Number)
	}

	// Two stacks in one task: the more pressing bottom.
	two := store.Task{State: store.TaskOpen, Changes: []store.TaskChange{
		pull(1, "a", "main", pr.WaitingReview), pull(2, "b", "a", pr.Failing),
		pull(5, "x", "main", pr.Conflict),
	}}
	if got := Headline(Live(two)); got.Number != 5 {
		t.Errorf("of two stacks, headline is #%d", got.Number)
	}
}

func TestATaskInPlaceSortsLast(t *testing.T) {
	here := store.Task{State: store.TaskOpen, InPlace: true}
	done := store.Task{State: store.TaskDone}
	if Rank(here) <= Rank(done) {
		t.Errorf("in place ranks %d, a finished task %d", Rank(here), Rank(done))
	}
	here.Agent = AgentDone
	if Rank(here) != Standing {
		t.Errorf("a finished agent moved it to %d", Rank(here))
	}
	here.Agent = AgentWaiting
	if Rank(here) != Mine {
		t.Errorf("an agent asking in place ranks %d, want it at the top", Rank(here))
	}
}

func TestStatusNamesTheAuthorOfAReview(t *testing.T) {
	review := store.Task{Changes: []store.TaskChange{
		{Head: "fix", Number: 7, State: "OPEN", Status: pr.ReviewRequested, Author: "sam"},
	}}
	if got := letter(review); !strings.Contains(got, "review #[fg=white]sam") {
		t.Errorf("a review does not name its author: %q", got)
	}
	mine := store.Task{Changes: []store.TaskChange{{Head: "fix", Number: 7, State: "OPEN", Status: pr.Ready}}}
	if got := letter(mine); strings.Contains(got, "white") {
		t.Errorf("my own pull request names an author: %q", got)
	}
}
