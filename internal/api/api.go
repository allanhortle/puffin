// Package api is what the client and the daemon say to each other: JSON over
// HTTP on a unix socket.
package api

import (
	"time"

	"puffin/internal/pr"
	"puffin/internal/store"
)

const (
	StatusPath   = "/v1/status"
	ShutdownPath = "/v1/shutdown"
	RefreshPath  = "/v1/refresh" // POST; ?force=1 skips the "has anything moved" check
	ReviewsPath  = "/v1/reviews"
	PRPath       = "/v1/pr" // GET; ?target= or ?repo=&branch= or ?repo=&number=
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

type PR struct {
	PR        pr.PR      `json:"pr"`
	Stack     []int      `json:"stack"`
	FetchedAt *time.Time `json:"fetched_at"`
}

type Error struct {
	Error string `json:"error"`
}
