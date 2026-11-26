// Package config reads ~/.config/puffin/config.yaml. puffin sits above every
// repository, so what a worktree of one needs is written here, keyed by the
// repository's name, rather than in a file inside it.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/allanhortle/puffin/internal/paths"
)

type Config struct {
	// Directories scanned for the main clone of each repository.
	Roots []string `yaml:"roots"`
	// Where each task's directory is made.
	TasksDir string `yaml:"tasks_dir"`
	// The command an agent window runs.
	Agent string `yaml:"agent"`
	// The flag that lets the agent read a directory outside the one it starts
	// in. Empty means the agent has no such flag.
	AgentDirFlag string `yaml:"agent_dir_flag"`
	// The flag that has the agent carry on its last conversation in the
	// directory it starts in, for a session brought back. Empty means it
	// always starts afresh.
	AgentResumeFlag string `yaml:"agent_resume_flag"`
	// What names a new task, and so its branch, from its prompt: a shell
	// command given an instruction and the prompt on stdin, which answers
	// with the name. Empty names a task by its prompt's first words.
	NameCommand string `yaml:"name_command"`
	// The instructions the name command is run with, as PUFFIN_NAME_PROMPT.
	// The default command gives it to the model as its system prompt.
	NamePrompt string `yaml:"name_prompt"`
	// What the agent in a new review task is asked to do.
	ReviewPrompt string `yaml:"review_prompt"`
	// The name of the tmux server to talk to, as given to `tmux -L`. Empty
	// means whichever server tmux would pick on its own.
	TmuxSocket string `yaml:"tmux_socket"`
	// The repository a new task is for when it is not started inside one.
	DefaultRepo string `yaml:"default_repo"`
	// Keyed by owner/name.
	Repo map[string]Repo `yaml:"repos"`
	// How `puffin read` shows a description.
	Read Read `yaml:"read"`
}

// Repo is what a fresh worktree of one repository needs before it is usable,
// and what has to happen before it is taken away.
type Repo struct {
	// Whether work in this repository happens on branches, each in a
	// worktree of its own. Without it the repository is worked on in place:
	// a task there is a session in the clone itself, and nothing is made.
	Worktree bool `yaml:"worktree"`
	// Globs, relative to the main clone, copied into a new worktree.
	Copy []string `yaml:"copy"`
	// Shell commands run in a new worktree, in the pane whose command is
	// "<setup>", or the first pane when none is.
	Setup []string `yaml:"setup"`
	// Shell commands run in a worktree before it is removed.
	PreRemove []string `yaml:"pre_remove"`
	// The window's panes, in order. Empty means a single agent pane.
	Panes []Pane `yaml:"panes"`
}

type Pane struct {
	// "<agent>" stands for the configured agent command, and "<setup>" for the
	// repository's setup commands, which run only when the worktree is new.
	Command string `yaml:"command"`
	// "horizontal" or "vertical"; empty for the window's first pane.
	Split string `yaml:"split"`
	Focus bool   `yaml:"focus"`
	// What the pane runs instead in a task made to review a pull request,
	// such as "puffin read". The same tokens stand.
	Review string `yaml:"review"`
}

// Read is how `puffin read` shows a description.
type Read struct {
	// Blocks a description carries that are not worth reading, each named by
	// the HTML comments around it: "CURSOR_SUMMARY" hides everything from
	// <!-- CURSOR_SUMMARY --> to <!-- /CURSOR_SUMMARY -->.
	Hide []string `yaml:"hide"`
}

// Load reads the file at path. A missing file is not an error: every setting
// has a default.
func Load(path string) (Config, error) {
	cfg := Config{Agent: "claude", AgentDirFlag: "--add-dir", AgentResumeFlag: "--continue", NameCommand: DefaultNameCommand, NamePrompt: DefaultNamePrompt, ReviewPrompt: "/code-review", TasksDir: paths.Tasks(),
		Read: Read{Hide: []string{"CURSOR_SUMMARY"}}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// The config was TOML once. Running on defaults instead of it would
		// be a quiet surprise.
		old := filepath.Join(filepath.Dir(path), "config.toml")
		if _, err := os.Stat(old); err == nil {
			return cfg, fmt.Errorf("%s is no longer read: puffin's config is %s now, in YAML; see config.example.yaml", old, path)
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	// A misspelt key would otherwise be silently ignored.
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return cfg, fmt.Errorf("%s: %w", path, err)
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

// DefaultNameCommand asks a small, quick model, with none of the user's
// settings, hooks, tools or MCP servers, and keeps no record of having asked.
const DefaultNameCommand = `claude -p --model haiku --system-prompt "$PUFFIN_NAME_PROMPT" --no-session-persistence --tools "" --setting-sources "" --strict-mcp-config`

// DefaultNamePrompt is the auto_name system prompt from workmux.
const DefaultNamePrompt = `You output a single git branch name. Nothing else.

Your ENTIRE response must be one line: the branch name. Do not think out
loud, restate the task, explain your choice, or add any preamble, quotes,
punctuation, or trailing text. The first and only thing you write is the
branch name itself.

Rules:
- kebab-case (lowercase, hyphen-separated)
- 2-4 words, 40 characters MAX (hard limit)
- name the core task/feature, not the implementation details
- no feat/, fix/, chore/ prefixes
- if the prompt is a Linear dump, prefix with the lowercase identifier:
  hoth-123-short-title, prod-456-short-title

Examples:
Task: "I need permission to read that file. Once you grant it I can examine
the workmux config and help shorten the auto-naming prompt." -> shorten-auto-name-prompt
Task: "PROD-5259 build a team reporting page for admins" -> prod-5259-team-reporting
`
