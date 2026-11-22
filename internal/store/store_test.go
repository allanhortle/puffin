package store

import (
	"context"
	"path/filepath"
	"testing"
)

func open(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenMigratesOnce(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "puffin.db")

	first := open(t, path)
	v, err := first.SchemaVersion(ctx)
	if err != nil || v < 1 {
		t.Fatalf("schema version = %d, %v", v, err)
	}
	first.Close()

	// Opening again must not try to create the tables a second time.
	second := open(t, path)
	if again, _ := second.SchemaVersion(ctx); again != v {
		t.Fatalf("schema version moved from %d to %d", v, again)
	}
	if _, err := second.Counts(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestConstraints(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "puffin.db"))
	exec := func(q string, args ...any) error {
		_, err := s.DB.ExecContext(ctx, q, args...)
		return err
	}
	const now = "2026-01-01T00:00:00Z"
	const newTask = `INSERT INTO task (slug, dir, flow, created_at) VALUES (?, ?, ?, ?)`

	if err := exec(newTask, "explore", "/tasks/explore", "work", now); err != nil {
		t.Fatal(err)
	}
	if err := exec(newTask, "explore", "/tasks/explore", "work", now); err == nil {
		t.Error("two live tasks shared a slug")
	}
	if err := exec(newTask, "other", "/tasks/other", "holiday", now); err == nil {
		t.Error("an unknown flow was accepted")
	}

	// Once a task is swept its directory is gone, so the slug is free again.
	if err := exec(`UPDATE task SET state = 'discarded', swept_at = ? WHERE slug = 'explore'`, now); err != nil {
		t.Fatal(err)
	}
	if err := exec(newTask, "explore", "/tasks/explore", "work", now); err != nil {
		t.Errorf("a swept task still held its slug: %v", err)
	}

	if err := exec(`INSERT INTO worktree (task_id, repo_id, path, created_at) VALUES (1, 99, '/x', ?)`, now); err == nil {
		t.Error("a worktree pointed at a repo that does not exist")
	}
}
