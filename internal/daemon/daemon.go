// Package daemon is the long-lived process: the one writer to the database,
// and in time the home of the GitHub poller and the tmux event stream.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"puffin/internal/api"
	"puffin/internal/config"
	"puffin/internal/store"
)

// ErrRunning means another daemon already holds the lock.
var ErrRunning = errors.New("a daemon is already running")

type Options struct {
	Version string
	Config  config.Config
	DB      string
	Socket  string
	Lock    string
}

type daemon struct {
	opts      Options
	store     *store.Store
	startedAt time.Time
	shutdown  context.CancelFunc
}

// Run serves until ctx is cancelled or a client asks it to stop.
func Run(ctx context.Context, opts Options) error {
	// An flock rather than a pid file: the kernel drops it when the process
	// dies, however it dies, so there is never a stale one to reason about.
	lock, err := acquire(opts.Lock)
	if err != nil {
		return err
	}
	defer lock.Close()

	st, err := store.Open(ctx, opts.DB)
	if err != nil {
		return err
	}
	defer st.Close()

	// Holding the lock, any socket still on disk belongs to a daemon that is
	// gone.
	if err := os.Remove(opts.Socket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	listener, err := net.Listen("unix", opts.Socket)
	if err != nil {
		return err
	}
	defer os.Remove(opts.Socket)
	if err := os.Chmod(opts.Socket, 0o600); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	d := &daemon{opts: opts, store: st, startedAt: time.Now().UTC(), shutdown: cancel}

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+api.StatusPath, d.status)
	mux.HandleFunc("POST "+api.ShutdownPath, d.stop)
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	slog.Info("listening", "socket", opts.Socket, "db", opts.DB)

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	slog.Info("stopping")
	grace, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	return server.Shutdown(grace)
}

func acquire(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrRunning
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return f, nil
}

func (d *daemon) status(w http.ResponseWriter, r *http.Request) {
	version, err := d.store.SchemaVersion(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	counts, err := d.store.Counts(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, api.Status{
		Version:       d.opts.Version,
		PID:           os.Getpid(),
		StartedAt:     d.startedAt,
		DB:            d.store.Path,
		SchemaVersion: version,
		Counts:        counts,
	})
}

func (d *daemon) stop(w http.ResponseWriter, _ *http.Request) {
	reply(w, http.StatusAccepted, struct{}{})
	d.shutdown()
}

func fail(w http.ResponseWriter, err error) {
	slog.Error("request failed", "err", err)
	reply(w, http.StatusInternalServerError, api.Error{Error: err.Error()})
}

func reply(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
