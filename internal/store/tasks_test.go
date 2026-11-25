package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/allanhortle/puffin/internal/pr"
)

func TestTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "puffin.db"))
	now := time.Unix(1_700_000_000, 0)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	owner := func(head string) (task int64) {
		t.Helper()
		must(s.DB.QueryRowContext(ctx,
			`SELECT coalesce(task_id, 0) FROM change WHERE head_ref = ?`, head).Scan(&task))
		return task
	}
	state := func(slug string) string {
		t.Helper()
		task, err := s.Task(ctx, slug)
		must(err)
		return task.State
	}
	fetched := func(number int, head, base, st string) pr.PR {
		return pr.PR{Repo: "o/r", Number: number, Head: head, Base: base, State: st, Status: pr.Ready}
	}

	must(s.SaveClone(ctx, "o/r", "/clones/r", "main"))
	repos, err := s.Repos(ctx)
	must(err)
	repo := repos[0]

	task := Task{Slug: "work", Dir: "/tasks/work", Flow: "work", Reads: []Repo{repo}}
	must(s.CreateTask(ctx, &task, now))
	other := Task{Slug: "other", Dir: "/tasks/other", Flow: "work"}
	must(s.CreateTask(ctx, &other, now))

	w := Worktree{TaskID: task.ID, Repo: repo, Path: "/tasks/work/r", Ref: "a"}
	must(s.AddWorktree(ctx, &w, "main", now))
	if owner("a") != task.ID {
		t.Fatal("attaching did not give the task its branch")
	}

	// A branch with no pull request holds the task open, and so does an
	// exploration with nothing attached.
	done, err := s.CompleteTasks(ctx, now)
	must(err)
	if len(done) != 0 {
		t.Fatalf("completed %v with nothing shipped", done)
	}

	// Pull requests appear: a for the task's branch, b stacked on a, c
	// stacked on b, and one that is no relation.
	must(s.ApplyFetch(ctx, now, []pr.PR{
		fetched(1, "a", "main", "OPEN"),
		fetched(2, "b", "a", "OPEN"),
		fetched(3, "c", "b", "OPEN"),
		fetched(4, "elsewhere", "main", "OPEN"),
	}, nil))
	if owner("a") != task.ID {
		t.Fatal("the fetch took the branch away from its task")
	}
	must(s.LinkStacks(ctx))
	if owner("b") != task.ID || owner("c") != task.ID {
		t.Errorf("stack not linked: b=%d c=%d", owner("b"), owner("c"))
	}
	if owner("elsewhere") != 0 {
		t.Error("an unrelated pull request was given to the task")
	}

	// A branch made inside the worktree is the task's from the moment the
	// worktree is seen standing on it.
	must(s.SetWorktreeRef(ctx, w, "fresh"))
	if owner("fresh") != task.ID {
		t.Error("a new branch in the worktree was not claimed")
	}
	local, err := s.LocalBranches(ctx)
	must(err)
	if len(local) != 1 || local[0].Head != "fresh" || local[0].Clone != "/clones/r" {
		t.Fatalf("local branches = %+v", local)
	}
	must(s.DeleteChange(ctx, local[0].ChangeID))

	// Two of three merged is not done.
	merged := []pr.PR{fetched(1, "a", "main", "MERGED"), fetched(2, "b", "a", "MERGED")}
	must(s.ApplyFetch(ctx, now, append(merged, fetched(3, "c", "b", "OPEN")), nil))
	done, err = s.CompleteTasks(ctx, now)
	must(err)
	if len(done) != 0 {
		t.Fatalf("completed %v with a pull request still open", done)
	}
	must(s.ApplyFetch(ctx, now, append(merged, fetched(3, "c", "b", "CLOSED")), nil))
	done, err = s.CompleteTasks(ctx, now)
	must(err)
	if len(done) != 1 || done[0] != "work" || state("work") != TaskDone || state("other") != TaskOpen {
		t.Fatalf("completed %v; work is %s, other is %s", done, state("work"), state("other"))
	}

	// Swept: the slug is free, and the pull requests are a cache again.
	must(s.RemoveWorktree(ctx, w.ID, now))
	must(s.MarkSwept(ctx, task.ID, now))
	if _, err := s.Task(ctx, "work"); err != ErrNoTask {
		t.Errorf("a swept task is still listed: %v", err)
	}
	if taken, _ := s.SlugTaken(ctx, "work"); taken {
		t.Error("a swept task still holds its slug")
	}
	if owner("a") != 0 {
		t.Error("a swept task still owns its pull requests")
	}
	live, err := s.LiveWorktrees(ctx)
	must(err)
	if len(live) != 0 {
		t.Errorf("worktrees after sweep: %+v", live)
	}
}

// A finished task reopened, swept or not, stays open with nothing live until
// a pull request of its own opens, and finishes as usual after that.
func TestReopen(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "puffin.db"))
	now := time.Unix(1_700_000_000, 0)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	ship := func(number int, state string) {
		t.Helper()
		must(s.ApplyFetch(ctx, now, []pr.PR{{Repo: "o/r", Number: number, Head: "a", Base: "main",
			State: state, Status: map[string]string{"OPEN": pr.Ready, "MERGED": pr.Merged}[state]}}, nil))
		must(s.ReleaseKeptOpen(ctx))
		_, err := s.CompleteTasks(ctx, now)
		must(err)
	}
	must(s.SaveClone(ctx, "o/r", "/clones/r", "main"))
	repos, err := s.Repos(ctx)
	must(err)
	task := Task{Slug: "work", Dir: "/tasks/work", Flow: "work"}
	must(s.CreateTask(ctx, &task, now))
	w := Worktree{TaskID: task.ID, Repo: repos[0], Path: "/tasks/work/r", Ref: "a"}
	must(s.AddWorktree(ctx, &w, "main", now))
	ship(1, "MERGED")

	got, err := s.Task(ctx, "work")
	must(err)
	if got.State != TaskDone {
		t.Fatalf("state after merging = %s", got.State)
	}
	must(s.RemoveWorktree(ctx, w.ID, now))
	must(s.MarkSwept(ctx, task.ID, now))
	if _, err := s.Task(ctx, "work"); err != ErrNoTask {
		t.Fatalf("swept task still live: %v", err)
	}

	swept, err := s.SweptTask(ctx, "work")
	must(err)
	must(s.Reopen(ctx, swept, now))
	gone, err := s.RemovedWorktrees(ctx, task.ID)
	must(err)
	if len(gone) != 1 || gone[0].Ref != "a" || gone[0].Path != "/tasks/work/r" {
		t.Fatalf("removed worktrees = %+v", gone)
	}
	w2 := Worktree{TaskID: task.ID, Repo: repos[0], Path: "/tasks/work/r", Ref: "a"}
	must(s.AddWorktree(ctx, &w2, "main", now))

	// Nothing live, the old pull request merged: it stays open regardless.
	ship(1, "MERGED")
	if got, _ := s.Task(ctx, "work"); got.State != TaskOpen {
		t.Fatalf("reopened task finished again on its old pull request: %s", got.State)
	}
	// A new pull request on the branch opens, then merges: done as usual.
	ship(2, "OPEN")
	ship(2, "MERGED")
	if got, _ := s.Task(ctx, "work"); got.State != TaskDone {
		t.Errorf("reopened task did not finish after its new pull request merged: %s", got.State)
	}
}
