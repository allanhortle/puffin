package task

import (
	"strings"
	"testing"

	"github.com/allanhortle/puffin/internal/config"
	"github.com/allanhortle/puffin/internal/store"
)

// claude's --add-dir takes any number of values, so a prompt after it is
// read as another directory and the agent starts with nothing to do.
func TestPromptComesBeforeTheDirectories(t *testing.T) {
	m := &Manager{cfg: config.Config{Agent: "claude", AgentDirFlag: "--add-dir"}}
	got := m.agent([]store.Repo{{Path: "/clones/a"}, {Path: "/clones/it's"}}, "/task/p.md", true)
	want := `claude "$(cat '/task/p.md')" --add-dir '/clones/a' --add-dir '/clones/it'\''s'`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if bare := m.agent(nil, "", false); bare != "claude" || strings.Contains(bare, " ") {
		t.Errorf("with nothing to add: %q", bare)
	}
}

// A session brought back carries on where its agent left off, and starts a
// fresh one when there was nothing to carry on.
func TestResume(t *testing.T) {
	m := &Manager{cfg: config.Config{Agent: "claude", AgentDirFlag: "--add-dir", AgentResumeFlag: "--continue"}}
	got := m.agent([]store.Repo{{Path: "/clones/a"}}, "", true)
	want := `claude --continue --add-dir '/clones/a' || claude --add-dir '/clones/a'`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if fresh := m.agent(nil, "", false); fresh != "claude" {
		t.Errorf("not resuming: %q", fresh)
	}
	m.cfg.AgentResumeFlag = ""
	if none := m.agent(nil, "", true); none != "claude" {
		t.Errorf("an agent with no resume flag: %q", none)
	}
}
