// Package api is what the client and the daemon say to each other: JSON over
// HTTP on a unix socket.
package api

import (
	"encoding/json"
	"time"

	"github.com/allanhortle/puffin/internal/pr"
	"github.com/allanhortle/puffin/internal/store"
)

const (
	StatusPath    = "/v1/status"
	ShutdownPath  = "/v1/shutdown"
	RefreshPath   = "/v1/refresh" // POST; ?force=1 skips the "has anything moved" check
	ReviewsPath   = "/v1/reviews"
	PRPath        = "/v1/pr" // GET; ?target= or ?repo=&branch= or ?repo=&number=
	PRsPath       = "/v1/prs"
	ReposPath     = "/v1/repos"        // GET lists; POST scans the roots again
	TasksPath     = "/v1/tasks"        // GET lists; POST makes one
	AttachPath    = "/v1/tasks/attach" // POST
	EnsurePath    = "/v1/tasks/ensure" // POST ?task=
	ClosePath     = "/v1/tasks/close"  // POST ?task=&force=1
	SweepPath     = "/v1/tasks/sweep"  // POST
	ReviewPath    = "/v1/tasks/review" // POST ?repo=&number=
	RenamePath    = "/v1/tasks/rename" // POST ?task=&to=
	AdoptPath     = "/v1/tasks/adopt"  // POST
	HerePath      = "/v1/tasks/here"   // POST
	AttentionPath = "/v1/attention"    // POST; recount what is waiting on me, now
	NudgePath     = "/v1/nudge"        // POST; something just changed on GitHub, look soon
)

type Status struct {
	Version       string       `json:"version"`
	PID           int          `json:"pid"`
	StartedAt     time.Time    `json:"started_at"`
	DB            string       `json:"db"`
	SchemaVersion int          `json:"schema_version"`
	Counts        store.Counts `json:"counts"`
	Poll          Poll         `json:"poll"`
}

// Poll is the state of the GitHub poller. A cycle and a fetch go stale for
// different reasons: no fetch means nothing has moved on GitHub, no cycle
// means the loop itself has stopped turning, and only the second is a fault.
type Poll struct {
	LastCycleAt   *time.Time `json:"last_cycle_at"`
	LastFetchAt   *time.Time `json:"last_fetch_at"`
	LastError     string     `json:"last_error,omitempty"`
	RateRemaining *int       `json:"rate_remaining"`
	Open          int        `json:"open"`
	Closed        int        `json:"closed"`
	Reviews       int        `json:"reviews"`
	Windows       []Window   `json:"windows"`
}

// Window is a tmux window and the pull request its tab reports on, which is
// the bottom of the stack the window's branch belongs to.
type Window struct {
	ID           string `json:"id"`
	Repo         string `json:"repo"`
	Branch       string `json:"branch"`
	Path         string `json:"path"`
	Pinned       bool   `json:"pinned,omitempty"`
	PR           int    `json:"pr,omitempty"`
	Status       string `json:"status,omitempty"`
	Bottom       int    `json:"bottom,omitempty"`
	BottomStatus string `json:"bottom_status,omitempty"`
	Stack        []int  `json:"stack,omitempty"`
	Unresolved   int    `json:"unresolved,omitempty"`
}

// Cycle is what one pass of the poller did.
type Cycle struct {
	Refetched bool `json:"refetched"`
	Windows   int  `json:"windows"`
	PRs       int  `json:"prs"`
	Reviews   int  `json:"reviews"`
}

// Reviews is the pull requests waiting on my review. Count is the true total;
// PRs is only the first page of it.
type Reviews struct {
	Count int     `json:"count"`
	PRs   []pr.PR `json:"prs"`
}

// PRs is everything the poller holds, for the tools that build their own
// lists out of it: my pull requests, the ones asking for my review, and the
// windows that have one open.
type PRs struct {
	FetchedAt *time.Time `json:"fetched_at"`
	PRs       []pr.PR    `json:"prs"`
	Reviews   Reviews    `json:"reviews"`
	Windows   []Window   `json:"windows"`
}

type PR struct {
	PR        pr.PR      `json:"pr"`
	Stack     []int      `json:"stack"`
	FetchedAt *time.Time `json:"fetched_at"`
}

// NewTask asks for a task. Read names repositories its root agent may read;
// Attach names repositories it gets a worktree of straight away.
type NewTask struct {
	Name   string   `json:"name,omitempty"`
	Prompt string   `json:"prompt,omitempty"`
	Read   []string `json:"read,omitempty"`
	Attach []string `json:"attach,omitempty"`
}

// Attach asks for a worktree of Repo in Task, on Branch, or on a branch named
// for the task when that is empty.
type Attach struct {
	Task   string `json:"task"`
	Repo   string `json:"repo"`
	Branch string `json:"branch,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

// Here asks for a task that lives in the directory at Path, as it is.
type Here struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// Adopt asks for the worktree at Path to become a task.
type Adopt struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// Swept is what a sweep did: the tasks it closed, and why it kept the rest.
type Swept struct {
	Closed []string          `json:"closed"`
	Kept   map[string]string `json:"kept"`
}

// Progress is one line of the reply to anything that changes a task. Such a
// request takes as long as git takes, so its reply is a stream: a line for
// each step as it starts, then a last one, Done, carrying the result or the
// error.
type Progress struct {
	Step   string          `json:"step,omitempty"`
	Done   bool            `json:"done,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type Error struct {
	Error string `json:"error"`
}
