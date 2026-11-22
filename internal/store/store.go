// Package store owns the SQLite database. The daemon is its only writer.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	DB   *sql.DB
	Path string
}

// Open opens the database at path, creating it and applying any migrations it
// has not yet seen.
func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// The pragmas ride in the DSN so every pooled connection gets them;
	// foreign_keys in particular is per connection and off by default.
	dsn := "file:" + path +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db, Path: path}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

// SchemaVersion is the number of the last migration applied.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	var v int
	err := s.DB.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v)
	return v, err
}

// A migration is a file named NNNN_description.sql. The number is stored in
// user_version, in the same transaction as the migration itself.
func (s *Store) migrate(ctx context.Context) error {
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	current, err := s.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	for _, name := range names {
		prefix, _, _ := strings.Cut(filepath.Base(name), "_")
		n, err := strconv.Atoi(prefix)
		if err != nil {
			return fmt.Errorf("%s: name must start with a number", name)
		}
		if n <= current {
			continue
		}
		body, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", n)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Counts is the size of each table a person would ask about.
type Counts struct {
	Repos     int `json:"repos"`
	OpenTasks int `json:"open_tasks"`
	Worktrees int `json:"worktrees"`
	Changes   int `json:"changes"`
}

func (s *Store) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	err := s.DB.QueryRowContext(ctx, `
		SELECT
		  (SELECT count(*) FROM repo),
		  (SELECT count(*) FROM task WHERE state = 'open'),
		  (SELECT count(*) FROM worktree WHERE removed_at IS NULL),
		  (SELECT count(*) FROM change)`,
	).Scan(&c.Repos, &c.OpenTasks, &c.Worktrees, &c.Changes)
	return c, err
}
