// Package run executes the commands puffin leans on: tmux, git and gh.
package run

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Timeout bounds every command. Nothing a cycle waits on may wait forever.
const Timeout = 20 * time.Second

var home, _ = os.UserHomeDir()

// Error carries what the command wrote to stderr, which is usually the only
// useful part.
type Error struct {
	Command string
	Stderr  string
	Err     error
}

func (e *Error) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s: %s", e.Command, e.Stderr)
	}
	return fmt.Sprintf("%s: %v", e.Command, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// Output runs a command and returns its stdout.
//
// It runs in the home directory. The daemon inherits its own directory from
// whichever pane started it, often a worktree that is later removed, and a
// child cannot be spawned in a directory that no longer exists.
func Output(ctx context.Context, name string, args ...string) (string, error) {
	return In(ctx, home, Timeout, name, args...)
}

// In is Output with the directory and the deadline chosen by the caller, for
// the few commands that need one or are slow for good reason.
func In(ctx context.Context, dir string, timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	// Stop waiting on the pipes shortly after the deadline kills the command,
	// even if a grandchild still holds them open.
	cmd.WaitDelay = time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return string(out), &Error{
			Command: name + " " + strings.Join(args, " "),
			Stderr:  strings.TrimSpace(stderr.String()),
			Err:     err,
		}
	}
	return string(out), nil
}
