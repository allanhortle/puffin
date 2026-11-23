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
	got := m.agent([]store.Repo{{Path: "/clones/a"}, {Path: "/clones/it's"}}, "/task/p.md")
	want := `claude "$(cat '/task/p.md')" --add-dir '/clones/a' --add-dir '/clones/it'\''s'`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if bare := m.agent(nil, ""); bare != "claude" || strings.Contains(bare, " ") {
		t.Errorf("with nothing to add: %q", bare)
	}
}
