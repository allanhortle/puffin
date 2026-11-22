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
	"strconv"
	"sync"
	"syscall"
	"time"

	"puffin/internal/api"
	"puffin/internal/config"
	"puffin/internal/store"
	"puffin/internal/tmux"
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
	poller    *poller
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
	tm := tmux.Client{Socket: opts.Config.TmuxSocket}
	d := &daemon{
		opts:      opts,
		store:     st,
		poller:    newPoller(st, tm),
		startedAt: time.Now().UTC(),
		shutdown:  cancel,
	}

	var polling sync.WaitGroup
	polling.Add(1)
	go func() {
		defer polling.Done()
		d.poller.run(ctx)
	}()
	// A stopped daemon takes its letters and its count down with it rather
	// than leaving stale ones on the bar. Deferred, so it also runs when the
	// server fails; and after the poller has finished, so nothing redraws
	// them.
	defer func() {
		cancel()
		polling.Wait()
		clearing, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err := tm.Clear(clearing); err != nil && !errors.Is(err, tmux.ErrNoServer) {
			slog.Error("clearing the bar failed", "err", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+api.StatusPath, d.status)
	mux.HandleFunc("POST "+api.ShutdownPath, d.stop)
	mux.HandleFunc("POST "+api.RefreshPath, d.refresh)
	mux.HandleFunc("GET "+api.ReviewsPath, d.reviews)
	mux.HandleFunc("GET "+api.PRPath, d.pr)
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

// Running reports whether a daemon holds the lock. This outlasts the socket:
// a daemon on its way out stops listening first and lets go of the lock last,
// and only then can another take its place.
func Running(lock string) bool {
	f, err := acquire(lock)
	if err != nil {
		return errors.Is(err, ErrRunning)
	}
	f.Close()
	return false
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
	poll, err := d.poller.snapshot(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, api.Status{
		Poll:          poll,
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

func (d *daemon) refresh(w http.ResponseWriter, r *http.Request) {
	cycle, err := d.poller.Refresh(r.Context(), r.URL.Query().Get("force") != "")
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, cycle)
}

func (d *daemon) reviews(w http.ResponseWriter, r *http.Request) {
	reviews, err := d.poller.reviews(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, reviews)
}

func (d *daemon) pr(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	number, _ := strconv.Atoi(q.Get("number"))
	found, err := d.poller.find(r.Context(), q.Get("target"), q.Get("repo"), q.Get("branch"), number)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, http.StatusOK, found)
}

func fail(w http.ResponseWriter, err error) {
	var missing errNoMatch
	if errors.As(err, &missing) {
		reply(w, http.StatusNotFound, api.Error{Error: err.Error()})
		return
	}
	slog.Error("request failed", "err", err)
	reply(w, http.StatusInternalServerError, api.Error{Error: err.Error()})
}

func reply(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
