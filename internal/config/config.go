// Package config reads ~/.config/puffin/config.toml. puffin sits above every
// repository, so what a worktree of one needs is written here, keyed by the
// repository's name, rather than in a file inside it.
package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/BurntSushi/toml"

	"github.com/allanhortle/puffin/internal/paths"
)

type Config struct {
	// Directories scanned for the main clone of each repository.
	Roots []string `toml:"roots"`
	// Where each task's directory is made.
	TasksDir string `toml:"tasks_dir"`
	// The command an agent window runs.
	Agent string `toml:"agent"`
	// The flag that lets the agent read a directory outside the one it starts
	// in. Empty means the agent has no such flag.
	AgentDirFlag string `toml:"agent_dir_flag"`
	// The flag that has the agent carry on its last conversation in the
	// directory it starts in, for a session brought back. Empty means it
	// always starts afresh.
	AgentResumeFlag string `toml:"agent_resume_flag"`
	// What names a new task, and so its branch, from its prompt: a shell
	// command given an instruction and the prompt on stdin, which answers
	// with the name. Empty names a task by its prompt's first words.
	NameCommand string `toml:"name_command"`
	// The instructions the name command is run with, as PUFFIN_NAME_PROMPT.
	// The default command gives it to the model as its system prompt.
	NamePrompt string `toml:"name_prompt"`
	// What the agent in a new review task is asked to do.
	ReviewPrompt string `toml:"review_prompt"`
	// The name of the tmux server to talk to, as given to `tmux -L`. Empty
	// means whichever server tmux would pick on its own.
	TmuxSocket string `toml:"tmux_socket"`
	// The repository a new task is for when it is not started inside one.
	DefaultRepo string `toml:"default_repo"`
	// Keyed by owner/name.
	Repo map[string]Repo `toml:"repo"`
}

// Repo is what a fresh worktree of one repository needs before it is usable,
// and what has to happen before it is taken away.
type Repo struct {
	// Whether work in this repository happens on branches, each in a
	// worktree of its own. Without it the repository is worked on in place:
	// a task there is a session in the clone itself, and nothing is made.
	Worktree bool `toml:"worktree"`
	// Globs, relative to the main clone, copied into a new worktree.
	Copy []string `toml:"copy"`
	// Shell commands run in a new worktree, in the pane whose command is
	// "<setup>", or the first pane when none is.
	Setup []string `toml:"setup"`
	// Shell commands run in a worktree before it is removed.
	PreRemove []string `toml:"pre_remove"`
	// The window's panes, in order. Empty means a single agent pane.
	Panes []Pane `toml:"panes"`
}

type Pane struct {
	// "<agent>" stands for the configured agent command, and "<setup>" for the
	// repository's setup commands, which run only when the worktree is new.
	Command string `toml:"command"`
	// "horizontal" or "vertical"; empty for the window's first pane.
	Split string `toml:"split"`
	Focus bool   `toml:"focus"`
}

// Load reads the file at path. A missing file is not an error: every setting
// has a default.
func Load(path string) (Config, error) {
	cfg := Config{Agent: "claude", AgentDirFlag: "--add-dir", AgentResumeFlag: "--continue", NameCommand: DefaultNameCommand, NamePrompt: DefaultNamePrompt, ReviewPrompt: "/code-review", TasksDir: paths.Tasks()}
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
