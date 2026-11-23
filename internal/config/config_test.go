package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMissingFileGivesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "claude" || cfg.TasksDir == "" {
		t.Errorf("defaults not applied: %+v", cfg)
	}
}

func TestExampleParses(t *testing.T) {
	cfg, err := Load("../../config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := cfg.Repo["owner/name"]
	if !ok {
		t.Fatalf("no owner/name section: %+v", cfg.Repo)
	}
	if !repo.Worktree {
		t.Error("the example repository is not marked for worktrees")
	}
	if len(repo.Copy) == 0 || len(repo.PreRemove) == 0 || len(repo.Panes) != 3 {
		t.Errorf("repo section incomplete: %+v", repo)
	}
	if strings.HasPrefix(cfg.Roots[0], "~") {
		t.Errorf("root not expanded: %s", cfg.Roots[0])
	}
}

func TestRejectsWhatItDoesNotUnderstand(t *testing.T) {
	for name, body := range map[string]string{
		"unknown key": `agnet = "claude"`,
		"bad split":   "[[repo.\"a/b\".panes]]\nsplit = \"diagonal\"",
	} {
		if _, err := Load(write(t, body)); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
