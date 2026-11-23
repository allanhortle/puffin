// Package pr turns what GitHub says about a pull request into the one thing
// worth showing: whose move it is.
package pr

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/allanhortle/puffin/internal/github"
)

// One status per pull request, and its colour says whose move it is: yellow a
// machine's, blue a reviewer's, red mine, green nobody's, grey nothing to do.
//
// WAITING_ME covers both a Chromatic baseline parked on my approval and a
// merge GitHub will not let through, because a human being in the way is the
// whole of what the letter has to say: which human does not change what I do
// next.
const (
	Ready            = "READY"
	Building         = "BUILDING"
	WaitingReview    = "WAITING_REVIEW"
	WaitingMe        = "WAITING_ME"
	Failing          = "FAILING"
	Conflict         = "CONFLICT"
	ChangesRequested = "CHANGES_REQUESTED"
	Queued           = "QUEUED"
	Merged           = "MERGED"
	Closed           = "CLOSED"
	Unknown          = "UNKNOWN"

	// What a pull request I am reviewing can be in: asking for my review,
	// reviewed and back with its author, or changed by its author since I
	// reviewed it.
	ReviewRequested = "REVIEW_REQUESTED"
	Reviewed        = "REVIEWED"
	ReviewUpdated   = "REVIEW_UPDATED"
)

type Label struct {
	Letter string
	Colour string // a tmux colour name
}

var draft = Label{"D", "colour242"}

var labels = map[string]Label{
	Ready:            {"G", "green"},
	Building:         {"B", "yellow"},
	WaitingReview:    {"W", "blue"},
	WaitingMe:        {"W", "red"},
	Failing:          {"F", "red"},
	Conflict:         {"C", "red"},
	ChangesRequested: {"R", "red"},
	Queued:           {"Q", "yellow"},
	Merged:           {"M", "magenta"},
	Closed:           {"X", "colour242"},
	Unknown:          {"?", "colour242"},
	// A review is a word, not a letter: it is somebody else's pull request,
	// and should not read as one of mine.
	ReviewRequested: {"rev", "red"},
	ReviewUpdated:   {"new", "red"},
	Reviewed:        {"rev", "blue"},
}

// LabelOf is the letter and colour for a status.
func LabelOf(status string) Label {
	if l, ok := labels[status]; ok {
		return l
	}
	return labels[Unknown]
}

// Checks is a commit's checks, counted by what has to happen next.
type Checks struct {
	Pass       int      `json:"pass"`
	Fail       int      `json:"fail"`
	Pending    int      `json:"pending"`  // a machine is still working
	Awaiting   int      `json:"awaiting"` // parked on a person
	Failing    []string `json:"failing,omitempty"`
	AwaitingOn []string `json:"awaiting_on,omitempty"`
}

// PR is a pull request, normalised. Head is the branch name, prefixed with
// `owner:` when the branch lives in a fork, so it is unique within Repo.
type PR struct {
	Repo    string `json:"repo"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	State   string `json:"state"` // OPEN, MERGED or CLOSED
	Head    string `json:"head"`
	Base    string `json:"base"`
	HeadSHA string `json:"head_sha"`
	IsDraft bool   `json:"is_draft"`
	Author  string `json:"author"`

	Mergeable        string `json:"mergeable"`
	MergeStateStatus string `json:"merge_state_status"`
	ReviewDecision   string `json:"review_decision"`
	IsInMergeQueue   bool   `json:"is_in_merge_queue"`
	QueuePosition    *int   `json:"queue_position"`
	Unresolved       int    `json:"unresolved"`
	Checks           Checks `json:"checks"`

	// The commit my last review was submitted on, for a pull request I am
	// reviewing.
	ReviewedSHA string `json:"reviewed_sha,omitempty"`

	Status string `json:"status"`
}

// Label is how the pull request renders. Draftness is deliberately not a
// status: every pull request here starts as a draft, so a draft letter would
// hide red CI on most of the bar. It changes how READY renders and nothing
// else.
func (p PR) Label() Label {
	if p.Status == Ready && p.IsDraft {
		return draft
	}
	return LabelOf(p.Status)
}

// Busy means a machine is working on it, so its state will move soon without
// any notification arriving to say so.
func (p PR) Busy() bool { return p.Checks.Pending > 0 || p.IsInMergeQueue }

func base(n github.Node) PR {
	p := PR{
		Repo:    n.Repository.NameWithOwner,
		Number:  n.Number,
		Title:   n.Title,
		URL:     n.URL,
		State:   n.State,
		Head:    n.HeadRefName,
		Base:    n.BaseRefName,
		HeadSHA: n.HeadRefOid,
		IsDraft: n.IsDraft,
	}
	if n.Author != nil {
		p.Author = n.Author.Login
	}
	// Two forks can both offer a branch called `patch-1`.
	if n.IsCrossRepository && n.HeadRepositoryOwner != nil {
		p.Head = n.HeadRepositoryOwner.Login + ":" + n.HeadRefName
	}
	return p
}

// Authored normalises one of my own pull requests and derives its status.
func Authored(n github.Node) PR {
	p := base(n)
	p.Mergeable = or(n.Mergeable, "UNKNOWN")
	p.MergeStateStatus = or(n.MergeStateStatus, "UNKNOWN")
	p.ReviewDecision = or(n.ReviewDecision, "NONE")
	p.IsInMergeQueue = n.IsInMergeQueue
	if n.MergeQueueEntry != nil {
		p.QueuePosition = &n.MergeQueueEntry.Position
	}
	if n.ReviewThreads != nil {
		for _, t := range n.ReviewThreads.Nodes {
			if !t.IsResolved {
				p.Unresolved++
			}
		}
	}
	p.Checks = summariseChecks(n)
	p.Status = deriveStatus(p)
	return p
}

// ToReview normalises a pull request that is asking for my review. Whose move
// it is has already been answered by it landing in that search at all.
func ToReview(n github.Node) PR {
	p := base(n)
	p.Status = ReviewRequested
	return p
}

// Watched normalises a pull request I have a review task for, fetched by
// number rather than found by a search. requested says it is also, right
// now, asking for my review.
//
// With no request outstanding, the head having moved off the commit I
// reviewed is the only sign that it wants another look. A restack moves it
// too, so this can say "new" when nothing of substance changed; that is the
// cheap side to be wrong on.
func Watched(n github.Node, requested bool) PR {
	p := base(n)
	if n.Reviews != nil && len(n.Reviews.Nodes) > 0 && n.Reviews.Nodes[0].Commit != nil {
		p.ReviewedSHA = n.Reviews.Nodes[0].Commit.Oid
	}
	switch {
	case p.State == "MERGED":
		p.Status = Merged
	case p.State == "CLOSED":
		p.Status = Closed
	case requested || p.ReviewedSHA == "":
		p.Status = ReviewRequested
	case p.ReviewedSHA != p.HeadSHA:
		p.Status = ReviewUpdated
	default:
		p.Status = Reviewed
	}
	return p
}

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func set(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

var (
	failStates = set("FAILURE", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE", "ERROR")
	// CANCELLED is green on purpose. Restacking a graphite branch cancels its
	// in-flight runs, and GitHub's own rollup state calls that a FAILURE even
	// though no check failed.
	greenStates = set("SUCCESS", "SKIPPED", "NEUTRAL", "CANCELLED")
)

// Chromatic parks its UI statuses at PENDING once the build is done and a
// person has to act: accept a baseline, or go collect approvals. Nothing is
// running, so counting it as building hides it behind whatever genuinely is.
// Matched on the description rather than the context name, because the very
// same context is also, legitimately, PENDING while the build runs. An
// unrecognised description therefore falls through to building, which is the
// safe way round: a check we cannot read stays yellow instead of crying wolf.
var awaitingDescriptions = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^awaiting \d+ approval`),
	regexp.MustCompile(`(?i)must be accepted as baseline`),
}

func needsAPerson(c github.Context) bool {
	for _, pattern := range awaitingDescriptions {
		if pattern.MatchString(c.Description) {
			return true
		}
	}
	return false
}

// Graphite's mergeability check sits IN_PROGRESS until every downstack pull
// request has merged, which can be weeks on a parked stack. It says nothing a
// stack's bottom letter does not, and counting it as building would keep the
// busy backstop on for as long as the stack waits.
func isStackGate(c github.Context) bool {
	return c.CheckSuite != nil && c.CheckSuite.App != nil &&
		c.CheckSuite.App.Slug == "graphite-app" &&
		c.Name == "Graphite / mergeability_check"
}

func checkState(c github.Context) string {
	return or(c.Conclusion, or(c.State, c.Status))
}

func contextName(c github.Context) string {
	return or(c.Name, or(c.Context, "?"))
}

// What a check has to match to count as the same check across attempts. The
// workflow rather than the run, so a re-run lands on the attempt it replaced,
// and the workflow as well as the name, because two workflows legitimately
// publish a check under one name. A check posted by an app rather than by
// Actions has no workflow and groups by the app instead.
func attemptKey(c github.Context) string {
	if s := c.CheckSuite; s != nil {
		if s.WorkflowRun != nil && s.WorkflowRun.Workflow != nil && s.WorkflowRun.Workflow.DatabaseID != nil {
			return fmt.Sprintf("workflow:%d %s", *s.WorkflowRun.Workflow.DatabaseID, contextName(c))
		}
		if s.App != nil {
			return fmt.Sprintf("app:%s %s", s.App.Slug, contextName(c))
		}
	}
	return "app: " + contextName(c)
}

// Which of two attempts is the later one. Run ids climb, so they order a
// workflow's attempts even while the newest is still queued and carries no
// start time; everything else falls back to when it was posted. Only ever
// compared within one attemptKey, so the two scales never meet.
func attemptOrder(c github.Context) int64 {
	if s := c.CheckSuite; s != nil && s.WorkflowRun != nil && s.WorkflowRun.DatabaseID != nil {
		return *s.WorkflowRun.DatabaseID
	}
	if c.StartedAt != nil {
		return c.StartedAt.UnixMilli()
	}
	if c.CreatedAt != nil {
		return c.CreatedAt.UnixMilli()
	}
	return 0
}

// A commit's rollup keeps every attempt of a check, so a re-run sits alongside
// the run it replaced and a failure that has since been re-run green still
// counts as a failure. GitHub's own rollup state has the same flaw, which is
// why it is only trusted when there are no contexts to read.
func latestAttempts(contexts []github.Context) []github.Context {
	newest := map[string]int{}
	var out []github.Context
	for _, c := range contexts {
		if isStackGate(c) {
			continue
		}
		key := attemptKey(c)
		i, held := newest[key]
		switch {
		case !held:
			newest[key] = len(out)
			out = append(out, c)
		case attemptOrder(c) >= attemptOrder(out[i]):
			out[i] = c
		}
	}
	return out
}

func names(contexts []github.Context) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range contexts {
		if name := contextName(c); !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func summariseChecks(n github.Node) Checks {
	var rollup *github.Rollup
	if n.Commits != nil && len(n.Commits.Nodes) > 0 {
		rollup = n.Commits.Nodes[0].Commit.StatusCheckRollup
	}
	if rollup == nil {
		return Checks{}
	}
	contexts := latestAttempts(rollup.Contexts.Nodes)

	var checks Checks
	if len(contexts) == 0 {
		// Nothing to classify ourselves, so the rollup state is all we have.
		if failStates[rollup.State] {
			checks.Fail = 1
		}
		if rollup.State == "PENDING" {
			checks.Pending = 1
		}
		return checks
	}

	var failing, awaiting []github.Context
	for _, c := range contexts {
		switch state := checkState(c); {
		case failStates[state]:
			failing = append(failing, c)
		case greenStates[state]:
			checks.Pass++
		// Neither green nor failing: in flight, or parked on a human.
		// Splitting the two is the whole point. Pending has to mean a machine
		// is still working, otherwise the busy backstop chases a state that
		// will never move.
		case needsAPerson(c):
			awaiting = append(awaiting, c)
		default:
			checks.Pending++
		}
	}
	checks.Fail = len(failing)
	checks.Awaiting = len(awaiting)
	checks.Failing = names(failing)
	checks.AwaitingOn = names(awaiting)
	return checks
}

func deriveStatus(p PR) string {
	switch {
	case p.State == "MERGED":
		return Merged
	case p.State == "CLOSED":
		return Closed
	case p.IsInMergeQueue:
		return Queued
	case p.Checks.Fail > 0:
		return Failing
	case p.Mergeable == "CONFLICTING" || p.MergeStateStatus == "DIRTY":
		return Conflict
	case p.ReviewDecision == "CHANGES_REQUESTED":
		return ChangesRequested
	// Ahead of BUILDING deliberately. A build clears itself and this does
	// not, so the thing worth showing while both are outstanding is the one
	// wanting me.
	case p.Checks.Awaiting > 0:
		return WaitingMe
	case p.Checks.Pending > 0:
		return Building
	}

	switch p.MergeStateStatus {
	// Nothing is failing or pending by this point, so UNSTABLE can only mean
	// cancelled or neutral runs, which already count as green.
	case "CLEAN", "BEHIND", "HAS_HOOKS", "DRAFT", "UNSTABLE":
		return Ready
	case "BLOCKED":
		if p.IsDraft {
			return Ready // blocked by its own draftness, not by anyone
		}
		if p.ReviewDecision == "REVIEW_REQUIRED" {
			return WaitingReview
		}
		return WaitingMe
	}
	// GitHub computes mergeability lazily and answers UNKNOWN while it works.
	return Unknown
}
