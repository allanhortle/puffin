package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const (
	TaskOpen      = "open"
	TaskDone      = "done"
	TaskDiscarded = "discarded"
)

// ErrNoTask means no live task has that slug.
var ErrNoTask = errors.New("no such task")

type Repo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"` // owner/name
	Path          string `json:"path,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type Worktree struct {
	ID     int64  `json:"id"`
	TaskID int64  `json:"task_id"`
	Repo   Repo   `json:"repo"`
	Path   string `json:"path"`
	Ref    string `json:"ref,omitempty"`
}

// TaskChange is a branch a task owns and, once it has one, its pull request.
type TaskChange struct {
	Repo   string `json:"repo"`
	Head   string `json:"head"`
	Number int    `json:"number,omitempty"`
	State  string `json:"state"`
	Status string `json:"status"`
	// Review threads nobody has resolved.
	Unresolved int  `json:"unresolved,omitempty"`
	IsDraft    bool `json:"is_draft,omitempty"`
}

type Task struct {
	ID        int64        `json:"id"`
	Slug      string       `json:"slug"`
	Dir       string       `json:"dir"`
	Title     string       `json:"title,omitempty"`
	Flow      string       `json:"flow"`
	State     string       `json:"state"`
	LinearID  string       `json:"linear_id,omitempty"`
	Prompt    string       `json:"prompt,omitempty"`
	CreatedAt string       `json:"created_at"`
	Reads     []Repo       `json:"reads"`
	Worktrees []Worktree   `json:"worktrees"`
	Changes   []TaskChange `json:"changes"`
}

// SaveClone records where a repository's main clone is.
func (s *Store) SaveClone(ctx context.Context, name, path, defaultBranch string) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO repo (name_with_owner, main_clone_path, default_branch) VALUES (?, ?, ?)
		ON CONFLICT (name_with_owner) DO UPDATE
		SET main_clone_path = excluded.main_clone_path, default_branch = excluded.default_branch`,
		name, path, defaultBranch)
	return err
}

// Repos is every repository with a known clone.
func (s *Store) Repos(ctx context.Context) ([]Repo, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, name_with_owner, main_clone_path, coalesce(default_branch, '')
		FROM repo WHERE main_clone_path IS NOT NULL ORDER BY name_with_owner`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.Name, &r.Path, &r.DefaultBranch); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SlugTaken reports whether a task that still has its directory uses slug.
func (s *Store) SlugTaken(ctx context.Context, slug string) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM task WHERE slug = ? AND swept_at IS NULL`, slug).Scan(&n)
	return n > 0, err
}

// CreateTask inserts t, with the repositories its root agent may read, and
// fills in its id.
func (s *Store) CreateTask(ctx context.Context, t *Task, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	t.CreatedAt = stamp(now)
	t.State = TaskOpen
	res, err := tx.ExecContext(ctx, `
		INSERT INTO task (slug, dir, title, flow, linear_id, prompt, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.Slug, t.Dir, t.Title, t.Flow, null(t.LinearID), t.Prompt, t.CreatedAt)
	if err != nil {
		return err
	}
	if t.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	for _, r := range t.Reads {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_repo (task_id, repo_id) VALUES (?, ?)`, t.ID, r.ID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO event (at, task_id, kind, to_value) VALUES (?, ?, 'task', ?)`,
		t.CreatedAt, t.ID, TaskOpen); err != nil {
		return err
	}
	return tx.Commit()
}

// Tasks is every task that still has its directory, oldest first.
func (s *Store) Tasks(ctx context.Context) ([]Task, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, slug, dir, title, flow, state, coalesce(linear_id, ''), prompt, created_at
		FROM task WHERE swept_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	for rows.Next() {
		t := Task{Reads: []Repo{}, Worktrees: []Worktree{}, Changes: []TaskChange{}}
		if err := rows.Scan(&t.ID, &t.Slug, &t.Dir, &t.Title, &t.Flow, &t.State,
			&t.LinearID, &t.Prompt, &t.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	byID := map[int64]*Task{}
	for i := range tasks {
		byID[tasks[i].ID] = &tasks[i]
	}

	reads, err := s.DB.QueryContext(ctx, `
		SELECT tr.task_id, r.id, r.name_with_owner, coalesce(r.main_clone_path, ''), coalesce(r.default_branch, '')
		FROM task_repo tr JOIN repo r ON r.id = tr.repo_id ORDER BY r.name_with_owner`)
	if err != nil {
		return nil, err
	}
	for reads.Next() {
		var task int64
		var r Repo
		if err := reads.Scan(&task, &r.ID, &r.Name, &r.Path, &r.DefaultBranch); err != nil {
			reads.Close()
			return nil, err
		}
		if t := byID[task]; t != nil {
			t.Reads = append(t.Reads, r)
		}
	}
	if err := reads.Close(); err != nil {
		return nil, err
	}

	worktrees, err := s.LiveWorktrees(ctx)
	if err != nil {
		return nil, err
	}
	for _, w := range worktrees {
		if t := byID[w.TaskID]; t != nil {
			t.Worktrees = append(t.Worktrees, w)
		}
	}

	changes, err := s.DB.QueryContext(ctx, `
		SELECT c.task_id, r.name_with_owner, c.head_ref, coalesce(c.pr_number, 0), c.state, c.status,
		       coalesce(json_extract(c.snapshot, '$.unresolved'), 0), c.is_draft
		FROM change c JOIN repo r ON r.id = c.repo_id
		WHERE c.task_id IS NOT NULL ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	defer changes.Close()
	for changes.Next() {
		var task int64
		var c TaskChange
		if err := changes.Scan(&task, &c.Repo, &c.Head, &c.Number, &c.State, &c.Status,
			&c.Unresolved, &c.IsDraft); err != nil {
			return nil, err
		}
		if t := byID[task]; t != nil {
			t.Changes = append(t.Changes, c)
		}
	}
	return tasks, changes.Err()
}

// Task is the live task with that slug.
func (s *Store) Task(ctx context.Context, slug string) (Task, error) {
	tasks, err := s.Tasks(ctx)
	if err != nil {
		return Task{}, err
	}
	for _, t := range tasks {
		if t.Slug == slug {
			return t, nil
		}
	}
	return Task{}, ErrNoTask
}

// LiveWorktrees is every worktree that has not been removed.
func (s *Store) LiveWorktrees(ctx context.Context) ([]Worktree, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT w.id, w.task_id, w.path, coalesce(w.current_ref, ''),
		       r.id, r.name_with_owner, coalesce(r.main_clone_path, ''), coalesce(r.default_branch, '')
		FROM worktree w JOIN repo r ON r.id = w.repo_id
		WHERE w.removed_at IS NULL ORDER BY w.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Worktree
	for rows.Next() {
		var w Worktree
		if err := rows.Scan(&w.ID, &w.TaskID, &w.Path, &w.Ref,
			&w.Repo.ID, &w.Repo.Name, &w.Repo.Path, &w.Repo.DefaultBranch); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// AddWorktree records a new worktree on branch, and that the task owns the
// branch. A branch that already has a row, because its pull request has been
// seen, keeps everything but its owner.
func (s *Store) AddWorktree(ctx context.Context, w *Worktree, base string, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO worktree (task_id, repo_id, path, current_ref, created_at) VALUES (?, ?, ?, ?, ?)`,
		w.TaskID, w.Repo.ID, w.Path, null(w.Ref), stamp(now))
	if err != nil {
		return err
	}
	if w.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO change (repo_id, head_ref, base_ref, task_id, role) VALUES (?, ?, ?, ?, 'author')
		ON CONFLICT (repo_id, head_ref) DO UPDATE SET task_id = excluded.task_id`,
		w.Repo.ID, w.Ref, base, w.TaskID); err != nil {
		return err
	}
	return tx.Commit()
}

// SetWorktreeRef records the branch a worktree has checked out, and claims
// that branch for the worktree's task if nothing owns it yet: a branch made
// inside a task's worktree is that task's work, whether or not it has a pull
// request.
func (s *Store) SetWorktreeRef(ctx context.Context, w Worktree, ref string) error {
	if _, err := s.DB.ExecContext(ctx,
		`UPDATE worktree SET current_ref = ? WHERE id = ?`, null(ref), w.ID); err != nil {
		return err
	}
	if ref == "" {
		return nil
	}
	// Its base is not known until GitHub says what the pull request targets.
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO change (repo_id, head_ref, base_ref, task_id, role) VALUES (?, ?, '', ?, 'author')
		ON CONFLICT (repo_id, head_ref) DO NOTHING`,
		w.Repo.ID, ref, w.TaskID); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `
		UPDATE change SET task_id = ? WHERE repo_id = ? AND head_ref = ? AND task_id IS NULL`,
		w.TaskID, w.Repo.ID, ref)
	return err
}

// LocalBranch is a branch a task owns that has no pull request.
type LocalBranch struct {
	ChangeID int64
	Clone    string
	Head     string
}

func (s *Store) LocalBranches(ctx context.Context) ([]LocalBranch, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, r.main_clone_path, c.head_ref
		FROM change c JOIN repo r ON r.id = c.repo_id
		WHERE c.pr_number IS NULL AND r.main_clone_path IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LocalBranch
	for rows.Next() {
		var b LocalBranch
		if err := rows.Scan(&b.ChangeID, &b.Clone, &b.Head); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) DeleteChange(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM change WHERE id = ?`, id)
	return err
}

func (s *Store) RemoveWorktree(ctx context.Context, id int64, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE worktree SET removed_at = ? WHERE id = ?`, stamp(now), id)
	return err
}

// LinkStacks gives every pull request in a stack to the task that owns any
// one of them. A branch stacked on a task's branch is the same piece of work,
// whether or not a worktree was ever standing on it when a cycle ran.
func (s *Store) LinkStacks(ctx context.Context) error {
	type row struct {
		id, repo   int64
		head, base string
		task       sql.NullInt64
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, repo_id, head_ref, base_ref, task_id FROM change WHERE role = 'author' AND state IN ('LOCAL', 'OPEN')`)
	if err != nil {
		return err
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.repo, &r.head, &r.base, &r.task); err != nil {
			rows.Close()
			return err
		}
		all = append(all, r)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	type key struct {
		repo int64
		ref  string
	}
	byHead := map[key]int{}
	parent := make([]int, len(all))
	for i, r := range all {
		byHead[key{r.repo, r.head}] = i
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	for i, r := range all {
		if below, ok := byHead[key{r.repo, r.base}]; ok {
			parent[find(i)] = find(below)
		}
	}
	owner := map[int]int64{}
	for i, r := range all {
		if r.task.Valid {
			if _, claimed := owner[find(i)]; !claimed {
				owner[find(i)] = r.task.Int64
			}
		}
	}
	for i, r := range all {
		task, ok := owner[find(i)]
		if r.task.Valid || !ok {
			continue
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE change SET task_id = ? WHERE id = ?`, task, r.id); err != nil {
			return err
		}
	}
	return nil
}

// CompleteTasks marks done every open task whose pull requests have all
// merged or closed, and returns their slugs. A branch with no pull request
// yet keeps its task open. Nothing is removed: that waits for a sweep.
func (s *Store) CompleteTasks(ctx context.Context, now time.Time) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, slug FROM task t
		WHERE state = 'open'
		  AND EXISTS (SELECT 1 FROM change c WHERE c.task_id = t.id)
		  AND NOT EXISTS (SELECT 1 FROM change c WHERE c.task_id = t.id AND c.state IN ('LOCAL', 'OPEN'))`)
	if err != nil {
		return nil, err
	}
	var ids []int64
	var slugs []string
	for rows.Next() {
		var id int64
		var slug string
		if err := rows.Scan(&id, &slug); err != nil {
			rows.Close()
			return nil, err
		}
		ids, slugs = append(ids, id), append(slugs, slug)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if err := s.SetTaskState(ctx, id, TaskOpen, TaskDone, now); err != nil {
			return nil, err
		}
	}
	return slugs, nil
}

func (s *Store) SetTaskState(ctx context.Context, id int64, from, to string, now time.Time) error {
	at := stamp(now)
	if _, err := s.DB.ExecContext(ctx,
		`UPDATE task SET state = ?, closed_at = coalesce(closed_at, ?) WHERE id = ?`, to, at, id); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO event (at, task_id, kind, from_value, to_value) VALUES (?, ?, 'task', ?, ?)`,
		at, id, from, to)
	return err
}

// MarkSwept records that a task's session and directory are gone, which frees
// its slug. Its pull requests stop belonging to it, so they go back to being
// a cache of what GitHub says, and a branch that never got one is forgotten.
func (s *Store) MarkSwept(ctx context.Context, id int64, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM change WHERE task_id = ? AND pr_number IS NULL`,
		`UPDATE change SET task_id = NULL WHERE task_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE task SET swept_at = ? WHERE id = ?`, stamp(now), id); err != nil {
		return err
	}
	return tx.Commit()
}
