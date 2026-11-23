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
