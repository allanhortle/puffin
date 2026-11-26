package main

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/allanhortle/puffin/internal/api"
	"github.com/allanhortle/puffin/internal/config"
	"github.com/allanhortle/puffin/internal/paths"
	"github.com/allanhortle/puffin/internal/task"
)

const (
	nameTimeout = 30 * time.Second
	// More than this of a prompt says nothing more about what to call it.
	namePromptLimit = 4000
)

var issueID = regexp.MustCompile(`\b[A-Za-z]{2,10}-\d+\b`)

// wantsBranch says whether a new task will get a branch, and so is worth
// naming well: one in a repository worked in worktrees, or none at all, which
// falls to the default repository or to a task with no repository. A
// repository worked in place names its own session.
func wantsBranch(repos []api.RepoRef, bare bool) bool {
	if bare {
		return true
	}
	cfg, err := config.Load(paths.Config())
	if err != nil {
		return false
	}
	names := []string{cfg.DefaultRepo}
	if len(repos) > 0 {
		names = names[:0]
		for _, r := range repos {
			names = append(names, r.Name)
		}
	}
	for _, name := range names {
		if name == "" {
			return true
		}
		for full, rc := range cfg.Repo {
			if rc.Worktree && (full == name || strings.HasSuffix(full, "/"+name)) {
				return true
			}
		}
	}
	return false
}

// suggestName asks the configured name command what to call a task, with
// the configured instructions in PUFFIN_NAME_PROMPT and the task on stdin.
// Any failure is no suggestion: the task is then named by the prompt's words.
func suggestName(ctx context.Context, prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	cfg, err := config.Load(paths.Config())
	if err != nil || cfg.NameCommand == "" {
		return ""
	}
	if len(prompt) > namePromptLimit {
		prompt = prompt[:namePromptLimit]
	}
	ctx, cancel := context.WithTimeout(ctx, nameTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.NameCommand)
	// Somewhere with nothing of the project in it to read.
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "PUFFIN_NAME_PROMPT="+cfg.NamePrompt)
	cmd.Stdin = strings.NewReader("Task: " + strconv.Quote(prompt) + "\n")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return cleanName(string(out), prompt)
}

// cleanName makes a branch name of the command's answer: its last line, with
// the issue id the prompt names in front, if the answer left it off. A line
// of prose is no name.
func cleanName(answer, prompt string) string {
	var line string
	for _, l := range strings.Split(answer, "\n") {
		if l = strings.Trim(strings.TrimSpace(l), "`'\"."); l != "" {
			line = l
		}
	}
	if len(strings.Fields(line)) > 1 {
		return ""
	}
	name := task.SlugifyName(line)
	if name == "" {
		return ""
	}
	if id := strings.ToLower(issueID.FindString(prompt)); id != "" && !strings.Contains(name, id) {
		name = task.SlugifyName(id + "-" + name)
	}
	return name
}
