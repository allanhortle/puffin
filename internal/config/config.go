// Package config reads ~/.config/puffin/config.toml. puffin sits above every
// repository, so what a worktree of one needs is written here, keyed by the
// repository's name, rather than in a file inside it.
package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/BurntSushi/toml"

	"puffin/internal/paths"
)

type Config struct {
	// Directories scanned for the main clone of each repository.
	Roots []string `toml:"roots"`
	// Where each task's directory is made.
	TasksDir string `toml:"tasks_dir"`
	// The command an agent window runs.
	Agent string `toml:"agent"`
	// Keyed by owner/name.
	Repo map[string]Repo `toml:"repo"`
}

// Repo is what a fresh worktree of one repository needs before it is usable,
// and what has to happen before it is taken away.
type Repo struct {
	// Globs, relative to the main clone, copied into a new worktree.
	Copy []string `toml:"copy"`
	// Shell commands run in a new worktree.
	Setup []string `toml:"setup"`
	// Shell commands run in a worktree before it is removed.
	PreRemove []string `toml:"pre_remove"`
	// The window's panes, in order. Empty means a single agent pane.
	Panes []Pane `toml:"panes"`
}

type Pane struct {
	// "<agent>" stands for the configured agent command.
	Command string `toml:"command"`
	// "horizontal" or "vertical"; empty for the window's first pane.
	Split string `toml:"split"`
	Focus bool   `toml:"focus"`
}

// Load reads the file at path. A missing file is not an error: every setting
// has a default.
func Load(path string) (Config, error) {
	cfg := Config{Agent: "claude", TasksDir: paths.Tasks()}
	meta, err := toml.DecodeFile(path, &cfg)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	// A misspelt key would otherwise be silently ignored.
	if extra := meta.Undecoded(); len(extra) > 0 {
		return cfg, fmt.Errorf("%s: unknown key %q", path, extra[0].String())
	}
	for name, repo := range cfg.Repo {
		for i, pane := range repo.Panes {
			switch pane.Split {
			case "", "horizontal", "vertical":
			default:
				return cfg, fmt.Errorf("%s: repo %q pane %d: split must be horizontal or vertical, not %q",
					path, name, i+1, pane.Split)
			}
		}
	}
	cfg.TasksDir = paths.Expand(cfg.TasksDir)
	for i, root := range cfg.Roots {
		cfg.Roots[i] = paths.Expand(root)
	}
	return cfg, nil
}
