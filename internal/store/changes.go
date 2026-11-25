package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/allanhortle/puffin/internal/pr"
)

const (
	RoleAuthor   = "author"
	RoleReviewer = "reviewer"
)

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func null(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func repoID(ctx context.Context, tx execer, name string) (int64, error) {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO repo (name_with_owner) VALUES (?) ON CONFLICT DO NOTHING`, name); err != nil {
		return 0, err
	}
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM repo WHERE name_with_owner = ?`, name).Scan(&id)
	return id, err
}

func recordStatus(ctx context.Context, tx execer, at string, changeID int64, task sql.NullInt64, from, to string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO event (at, task_id, change_id, kind, from_value, to_value) VALUES (?, ?, ?, 'status', ?, ?)`,
		at, task, changeID, null(from), to)
	return err
}

// ApplyFetch makes the change table agree with one complete answer from
// GitHub: authored is every pull request of mine it returned, reviews every
// one asking for my review. Each status that moved is recorded as an event.
//
// A pull request missing from the answer has aged out of the search. Its row
// stays, as the record of it, and a review that is no longer requested becomes
// REVIEWED. With no task attached it is archived: kept, and shown nowhere.
func (s *Store) ApplyFetch(ctx context.Context, now time.Time, authored, reviews []pr.PR) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	at := stamp(now)
	seen := map[int64]bool{}

	apply := func(role string, list []pr.PR) error {
		// A branch can carry an old closed pull request as well as a live one.
		// The row belongs to the open one, then to the newest.
		ranked := append([]pr.PR(nil), list...)
		sort.SliceStable(ranked, func(i, j int) bool {
			if (ranked[i].State == "OPEN") != (ranked[j].State == "OPEN") {
				return ranked[i].State == "OPEN"
			}
			return ranked[i].Number > ranked[j].Number
		})
		for _, p := range ranked {
			repo, err := repoID(ctx, tx, p.Repo)
			if err != nil {
				return err
			}
			snapshot, err := json.Marshal(p)
			if err != nil {
				return err
			}

			var id int64
			var task sql.NullInt64
			var old string
			err = tx.QueryRowContext(ctx,
				`SELECT id, task_id, status FROM change WHERE repo_id = ? AND head_ref = ?`,
				repo, p.Head).Scan(&id, &task, &old)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				res, err := tx.ExecContext(ctx, `
					INSERT INTO change (repo_id, head_ref, base_ref, pr_number, role, author, title,
					                    state, is_draft, status, snapshot, last_reviewed_sha, fetched_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					repo, p.Head, p.Base, p.Number, role, null(p.Author), p.Title,
					p.State, p.IsDraft, p.Status, string(snapshot), null(p.ReviewedSHA), at)
				if err != nil {
					return err
				}
				if id, err = res.LastInsertId(); err != nil {
					return err
				}
			case err != nil:
				return err
			case seen[id]:
				continue // an older pull request on a branch already claimed
			default:
				if _, err := tx.ExecContext(ctx, `
					UPDATE change
					SET base_ref = ?, pr_number = ?, role = ?, author = ?, title = ?, state = ?,
					    is_draft = ?, status = ?, snapshot = ?, fetched_at = ?, archived_at = NULL,
					    last_reviewed_sha = coalesce(?, last_reviewed_sha)
					WHERE id = ?`,
					p.Base, p.Number, role, null(p.Author), p.Title, p.State,
					p.IsDraft, p.Status, string(snapshot), at, null(p.ReviewedSHA), id); err != nil {
					return err
				}
			}
			seen[id] = true
			if old != p.Status {
				if err := recordStatus(ctx, tx, at, id, task, old, p.Status); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := apply(RoleAuthor, authored); err != nil {
		return err
	}
	if err := apply(RoleReviewer, reviews); err != nil {
		return err
	}

	type row struct {
		id     int64
		task   sql.NullInt64
		role   string
		status string
	}
	var unseen []row
	rows, err := tx.QueryContext(ctx,
		`SELECT id, task_id, role, status FROM change WHERE pr_number IS NOT NULL`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.task, &r.role, &r.status); err != nil {
			rows.Close()
			return err
		}
		if !seen[r.id] {
			unseen = append(unseen, r)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, r := range unseen {
		if r.role == RoleReviewer && r.status == pr.ReviewRequested {
			if _, err := tx.ExecContext(ctx,
				`UPDATE change SET status = ? WHERE id = ?`, pr.Reviewed, r.id); err != nil {
				return err
			}
			if err := recordStatus(ctx, tx, at, r.id, r.task, r.status, pr.Reviewed); err != nil {
				return err
			}
		}
		if !r.task.Valid {
			if _, err := tx.ExecContext(ctx,
				`UPDATE change SET archived_at = coalesce(archived_at, ?) WHERE id = ?`, at, r.id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// WatchedReviews is every pull request a review task owns: the ones that have
// to be asked after by number, because once reviewed they are in no search.
func (s *Store) WatchedReviews(ctx context.Context) ([]pr.PR, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT r.name_with_owner, c.pr_number FROM change c JOIN repo r ON r.id = c.repo_id
		WHERE c.role = 'reviewer' AND c.task_id IS NOT NULL AND c.pr_number IS NOT NULL
		  AND c.state = 'OPEN' ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pr.PR
	for rows.Next() {
		var p pr.PR
		if err := rows.Scan(&p.Repo, &p.Number); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Changes is every pull request held in one role, oldest row first. An
// archived one is history, and not among them.
func (s *Store) Changes(ctx context.Context, role string) ([]pr.PR, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT snapshot, status FROM change
		 WHERE role = ? AND pr_number IS NOT NULL AND archived_at IS NULL ORDER BY id`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pr.PR
	for rows.Next() {
		var snapshot, status string
		if err := rows.Scan(&snapshot, &status); err != nil {
			return nil, err
		}
		var p pr.PR
		if err := json.Unmarshal([]byte(snapshot), &p); err != nil {
			return nil, err
		}
		// The column is the truth: a review that stopped being requested was
		// moved on without a fresh snapshot.
		p.Status = status
		out = append(out, p)
	}
	return out, rows.Err()
}

// Meta reads one of the daemon's bookkeeping values into out. It reports
// false, leaving out alone, when the key has never been written.
func (s *Store) Meta(ctx context.Context, key string, out any) (bool, error) {
	var value string
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(value), out)
}

func (s *Store) SetMeta(ctx context.Context, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = excluded.value`,
		key, string(encoded))
	return err
}
