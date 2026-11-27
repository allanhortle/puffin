package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMissingFileGivesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "claude" || cfg.TasksDir == "" {
		t.Errorf("defaults not applied: %+v", cfg)
	}
}

func TestExampleParses(t *testing.T) {
	cfg, err := Load("../../config.example.yaml")
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
	if len(repo.Copy) == 0 || len(repo.PreRemove) == 0 || len(repo.Panes.Work) != 3 || len(repo.Panes.For("review")) == 0 {
		t.Errorf("repo section incomplete: %+v", repo)
	}
	if strings.HasPrefix(cfg.Roots[0], "~") {
		t.Errorf("root not expanded: %s", cfg.Roots[0])
	}
}

func TestRejectsWhatItDoesNotUnderstand(t *testing.T) {
	for name, body := range map[string]string{
		"unknown key":        "agnet: claude",
		"unknown nested key": "repos:\n  a/b:\n    worktre: true",
		"bad split":          "repos:\n  a/b:\n    panes:\n      review:\n        - split: diagonal",
		"unknown task type":  "repos:\n  a/b:\n    panes:\n      explore:\n        - command: x",
		"panes as a list":    "repos:\n  a/b:\n    panes:\n      - command: x",
	} {
		if _, err := Load(write(t, body)); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestAnOldTOMLConfigIsNotSilentlyIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("agent = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(dir, "config.yaml")); err == nil {
		t.Error("a config.toml with no config.yaml was ignored")
	}
}

func TestEmptyFileGivesDefaults(t *testing.T) {
	cfg, err := Load(write(t, ""))
	if err != nil || cfg.Agent != "claude" || len(cfg.Read.Hide) != 0 {
		t.Errorf("empty file: %+v, %v", cfg, err)
	}
}

func TestLayoutsFor(t *testing.T) {
	work := []Pane{{Command: "<agent>"}}
	review := []Pane{{Command: "puffin read"}, {Command: "<agent>"}}
	if got := (Layouts{Work: work}).For("review"); len(got) != 1 {
		t.Errorf("review with no layout of its own = %+v, want work's", got)
	}
	if got := (Layouts{Work: work, Review: review}).For("review"); len(got) != 2 {
		t.Errorf("review = %+v", got)
	}
	if got := (Layouts{Work: work, Review: review}).For("work"); len(got) != 1 {
		t.Errorf("work = %+v", got)
	}
}
