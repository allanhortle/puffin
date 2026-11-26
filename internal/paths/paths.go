// Package paths says where puffin keeps things, following the XDG layout.
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func xdg(env string, fallback ...string) string {
	if dir := os.Getenv(env); dir != "" {
		return filepath.Join(dir, "puffin")
	}
	return filepath.Join(append(append([]string{home()}, fallback...), "puffin")...)
}

func configDir() string { return xdg("XDG_CONFIG_HOME", ".config") }
func dataDir() string   { return xdg("XDG_DATA_HOME", ".local", "share") }
func stateDir() string  { return xdg("XDG_STATE_HOME", ".local", "state") }

func Config() string { return filepath.Join(configDir(), "config.yaml") }
func DB() string     { return filepath.Join(dataDir(), "puffin.db") }
func Tasks() string  { return filepath.Join(dataDir(), "tasks") }
func Log() string    { return filepath.Join(stateDir(), "daemon.log") }
func Lock() string   { return filepath.Join(stateDir(), "daemon.lock") }

// Socket stays under the state directory rather than a temp directory: macOS
// caps a socket path at 104 bytes and its per-user temp paths run long.
func Socket() string { return filepath.Join(stateDir(), "puffin.sock") }

// Expand turns a leading ~ into the home directory.
func Expand(p string) string {
	if p == "~" {
		return home()
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return filepath.Join(home(), rest)
	}
	return p
}
