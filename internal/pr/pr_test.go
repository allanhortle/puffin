package pr

import (
	"reflect"
	"testing"
	"time"

	"puffin/internal/github"
)

func ptr[T any](v T) *T { return &v }

// An Actions check: a run of a workflow.
func run(name, conclusion string, workflow, runID int64) github.Context {
	suite := &github.CheckSuite{}
	suite.WorkflowRun = &struct {
		DatabaseID *int64 `json:"databaseId"`
		Workflow   *struct {
			DatabaseID *int64 `json:"databaseId"`
		} `json:"workflow"`
	}{DatabaseID: &runID}
	suite.WorkflowRun.Workflow = &struct {
		DatabaseID *int64 `json:"databaseId"`
	}{DatabaseID: &workflow}
	c := github.Context{Name: name, Conclusion: conclusion, CheckSuite: suite}
	if conclusion == "" {
		c.Status = "IN_PROGRESS"
	}
	return c
}

func status(context, state, description string) github.Context {
	return github.Context{Context: context, State: state, Description: description,
		CreatedAt: ptr(time.Unix(1, 0))}
}

func node(mergeState string, contexts ...github.Context) github.Node {
	n := github.Node{Number: 1, State: "OPEN", HeadRefName: "a", BaseRefName: "main",
		Mergeable: "MERGEABLE", MergeStateStatus: mergeState, ReviewDecision: "REVIEW_REQUIRED"}
	n.Repository.NameWithOwner = "o/r"
	rollup := &github.Rollup{State: "SUCCESS"}
	rollup.Contexts.Nodes = contexts
	n.Commits = &struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *github.Rollup `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	}{Nodes: make([]struct {
		Commit struct {
			StatusCheckRollup *github.Rollup `json:"statusCheckRollup"`
		} `json:"commit"`
	}, 1)}
	n.Commits.Nodes[0].Commit.StatusCheckRollup = rollup
	return n
}

func TestStatus(t *testing.T) {
	gate := github.Context{Name: "Graphite / mergeability_check", Status: "IN_PROGRESS",
		CheckSuite: &github.CheckSuite{App: &struct {
			Slug string `json:"slug"`
		}{Slug: "graphite-app"}}}

	cases := []struct {
		name string
		node github.Node
		edit func(*github.Node)
		want string
	}{
		{name: "green and unblocked", node: node("CLEAN", run("test", "SUCCESS", 1, 10)), want: Ready},
		{name: "a failure", node: node("BLOCKED", run("test", "FAILURE", 1, 10)), want: Failing},
		{name: "a failure re-run green is not a failure",
			node: node("CLEAN", run("test", "FAILURE", 1, 10), run("test", "SUCCESS", 1, 11)), want: Ready},
		{name: "the re-run wins even when listed first",
			node: node("CLEAN", run("test", "SUCCESS", 1, 11), run("test", "FAILURE", 1, 10)), want: Ready},
		{name: "the same name in two workflows is two checks",
			node: node("BLOCKED", run("e2e", "FAILURE", 1, 10), run("e2e", "SUCCESS", 2, 11)), want: Failing},
		{name: "cancelled is green", node: node("UNSTABLE", run("test", "CANCELLED", 1, 10)), want: Ready},
		{name: "in flight", node: node("BLOCKED", run("test", "", 1, 10)), want: Building},
		{name: "the stack gate is not a build", node: node("CLEAN", gate), want: Ready},
		{name: "parked on a person outranks a build",
			node: node("BLOCKED", run("test", "", 1, 10),
				status("UI Tests", "PENDING", "3 changes must be accepted as baseline")),
			want: WaitingMe},
		{name: "an unread pending status is a build",
			node: node("BLOCKED", status("UI Tests", "PENDING", "Build 12 in progress")), want: Building},
		{name: "blocked on review", node: node("BLOCKED", run("test", "SUCCESS", 1, 10)), want: WaitingReview},
		{name: "approved and still blocked is mine", node: node("BLOCKED"),
			edit: func(n *github.Node) { n.ReviewDecision = "APPROVED" }, want: WaitingMe},
		{name: "a draft is blocked by itself", node: node("BLOCKED"),
			edit: func(n *github.Node) { n.IsDraft = true }, want: Ready},
		{name: "conflict", node: node("DIRTY"), want: Conflict},
		{name: "changes requested", node: node("BLOCKED"),
			edit: func(n *github.Node) { n.ReviewDecision = "CHANGES_REQUESTED" }, want: ChangesRequested},
		{name: "a failure outranks a conflict", node: node("DIRTY", run("test", "FAILURE", 1, 10)), want: Failing},
		{name: "queued outranks everything open", node: node("BLOCKED", run("test", "FAILURE", 1, 10)),
			edit: func(n *github.Node) { n.IsInMergeQueue = true }, want: Queued},
		{name: "merged", node: node("CLEAN"), edit: func(n *github.Node) { n.State = "MERGED" }, want: Merged},
		{name: "mergeability not computed yet", node: node(""), want: Unknown},
	}
	for _, c := range cases {
		if c.edit != nil {
			c.edit(&c.node)
		}
		if got := Authored(c.node).Status; got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestChecksNameWhatIsWrong(t *testing.T) {
	checks := Authored(node("BLOCKED",
		run("lint", "FAILURE", 1, 10), run("build", "FAILURE", 1, 10), run("test", "SUCCESS", 1, 10),
		status("UI Review", "PENDING", "Awaiting 2 approvals"))).Checks
	want := Checks{Pass: 1, Fail: 2, Awaiting: 1,
		Failing: []string{"build", "lint"}, AwaitingOn: []string{"UI Review"}}
	if !reflect.DeepEqual(checks, want) {
		t.Errorf("got %+v, want %+v", checks, want)
	}
}

func TestDraftChangesOnlyHowReadyRenders(t *testing.T) {
	if l := (PR{Status: Ready, IsDraft: true}).Label(); l.Letter != "D" {
		t.Errorf("a ready draft renders %q", l.Letter)
	}
	if l := (PR{Status: Failing, IsDraft: true}).Label(); l.Letter != "F" {
		t.Errorf("a failing draft renders %q", l.Letter)
	}
}

func TestForkBranchesDoNotCollide(t *testing.T) {
	n := node("CLEAN")
	n.IsCrossRepository = true
	n.HeadRepositoryOwner = &github.Actor{Login: "someone"}
	if head := Authored(n).Head; head != "someone:a" {
		t.Errorf("head = %q", head)
	}
}

func TestStacks(t *testing.T) {
	open := func(repo string, number int, head, base string) PR {
		return PR{Repo: repo, Number: number, State: "OPEN", Head: head, Base: base}
	}
	// 30 was restacked beneath 10, so the lowest number is not the bottom.
	prs := []PR{
		open("o/r", 10, "b", "a"),
		open("o/r", 20, "c", "b"),
		open("o/r", 30, "a", "main"),
		open("o/r", 40, "lone", "main"),
		// The same numbers and branch names in another repository are no relation.
		open("o/other", 10, "b", "a"),
		// An old closed pull request on a branch that now carries 30.
		{Repo: "o/r", Number: 5, State: "CLOSED", Head: "a", Base: "main"},
	}
	s := BuildStacks(prs)

	if got := s.BottomOf(prs[1]).Number; got != 30 {
		t.Errorf("bottom of 20 is %d", got)
	}
	if got := s.StackOf(prs[1]); !reflect.DeepEqual(got, []int{10, 20, 30}) {
		t.Errorf("stack of 20 is %v", got)
	}
	if got := s.StackOf(prs[3]); !reflect.DeepEqual(got, []int{40}) {
		t.Errorf("stack of 40 is %v", got)
	}
	if got := s.BottomOf(prs[4]); got.Repo != "o/other" || got.Number != 10 {
		t.Errorf("the other repository's 10 resolved to %s#%d", got.Repo, got.Number)
	}
	if got, _ := s.ByHead("o/r", "a"); got.Number != 30 {
		t.Errorf("branch a carries %d", got.Number)
	}
	if _, ok := s.ByHead("o/r", "main"); ok {
		t.Error("main carries a pull request")
	}
}

func TestStackSurvivesACycle(t *testing.T) {
	prs := []PR{
		{Repo: "o/r", Number: 1, State: "OPEN", Head: "a", Base: "b"},
		{Repo: "o/r", Number: 2, State: "OPEN", Head: "b", Base: "a"},
	}
	done := make(chan PR, 1)
	go func() { done <- BuildStacks(prs).BottomOf(prs[0]) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("BottomOf did not return")
	}
}
