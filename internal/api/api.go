// Package api is what the client and the daemon say to each other: JSON over
// HTTP on a unix socket.
package api

import (
	"time"

	"puffin/internal/store"
)

const (
	StatusPath   = "/v1/status"
	ShutdownPath = "/v1/shutdown"
)

type Status struct {
	Version       string       `json:"version"`
	PID           int          `json:"pid"`
	StartedAt     time.Time    `json:"started_at"`
	DB            string       `json:"db"`
	SchemaVersion int          `json:"schema_version"`
	Counts        store.Counts `json:"counts"`
}

type Error struct {
	Error string `json:"error"`
}
