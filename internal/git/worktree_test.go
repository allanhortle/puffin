package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveWorktree(t *testing.T) {
	ctx := context.Background()
	r := repo{t, t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.commit("base.txt", "one\n")
	outside := t.TempDir()
	path, trash := filepath.Join(outside, "tree"), filepath.Join(outside, "trash")
	r.git("worktree", "add", "-q", "-b", "feature", path)

	if err := os.WriteFile(filepath.Join(path, "new.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorktree(ctx, r.dir, path, trash, false); err == nil || !strings.Contains(err.Error(), "modified or untracked") {
		t.Fatalf("uncommitted work should stop it, got %v", err)
	}
	if err := RemoveWorktree(ctx, r.dir, path, trash, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the worktree is still there: %v", err)
	}
	if n := WorktreeCount(ctx, r.dir); n != 1 {
		t.Fatalf("git still counts %d worktrees", n)
	}
}
