package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"puffin/internal/api"
	"puffin/internal/client"
	"puffin/internal/config"
	"puffin/internal/daemon"
	"puffin/internal/paths"
)

const version = "0.0.1"

const help = `puffin: tasks, worktrees and pull requests, in tmux

  puffin start      start the daemon in the background, if it is not running
  puffin stop       stop the daemon
  puffin status     what the daemon holds; --json for the raw reply
  puffin daemon     run the daemon in the foreground
  puffin version
`

// Nothing a client waits on is allowed to wait forever.
const clientTimeout = 5 * time.Second

func main() {
	command := "help"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	args := os.Args[min(2, len(os.Args)):]

	var err error
	switch command {
	case "daemon":
		err = runDaemon()
	case "start":
		err = start()
	case "stop":
		err = stop()
	case "status":
		err = status(len(args) > 0 && args[0] == "--json")
	case "version", "--version":
		fmt.Println(version)
	case "help", "--help", "-h":
		fmt.Print(help)
	default:
		fmt.Fprintf(os.Stderr, "puffin: unknown command %q\n\n%s", command, help)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "puffin:", err)
		os.Exit(1)
	}
}

func runDaemon() error {
	cfg, err := config.Load(paths.Config())
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return daemon.Run(ctx, daemon.Options{
		Version: version,
		Config:  cfg,
		DB:      paths.DB(),
		Socket:  paths.Socket(),
		Lock:    paths.Lock(),
	})
}

// start is idempotent, so a tmux config reload can call it freely.
func start() error {
	c := client.New(paths.Socket())
	if _, err := ping(c); err == nil {
		return nil
	}
	// Fail here, where the message reaches a terminal, rather than in the log.
	if _, err := config.Load(paths.Config()); err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(paths.Log()), 0o755); err != nil {
		return err
	}
	log, err := os.OpenFile(paths.Log(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer log.Close()

	cmd := exec.Command(self, "daemon")
	cmd.Stdout, cmd.Stderr = log, log
	// Its own session, so it outlives the terminal or tmux hook that started it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	deadline := time.After(clientTimeout)
	for {
		select {
		case <-exited:
			return fmt.Errorf("the daemon exited as it started; see %s", paths.Log())
		case <-deadline:
			return fmt.Errorf("the daemon did not answer within %s; see %s", clientTimeout, paths.Log())
		case <-time.After(50 * time.Millisecond):
			if _, err := ping(c); err == nil {
				return nil
			}
		}
	}
}

func stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()
	err := client.New(paths.Socket()).Shutdown(ctx)
	if errors.Is(err, client.ErrNotRunning) {
		return nil
	}
	return err
}

func status(asJSON bool) error {
	s, err := ping(client.New(paths.Socket()))
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	}
	fmt.Printf("puffin %s, pid %d, up %s\n", s.Version, s.PID,
		time.Since(s.StartedAt).Round(time.Second))
	fmt.Printf("%s (schema %d)\n", s.DB, s.SchemaVersion)
	fmt.Printf("%d repos, %d open tasks, %d worktrees, %d changes\n",
		s.Counts.Repos, s.Counts.OpenTasks, s.Counts.Worktrees, s.Counts.Changes)
	return nil
}

func ping(c *client.Client) (api.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()
	return c.Status(ctx)
}
