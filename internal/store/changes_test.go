package store

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/allanhortle/puffin/internal/pr"
)

func mine(number int, head, state, status string) pr.PR {
	return pr.PR{Repo: "o/r", Number: number, Head: head, Base: "main", State: state, Status: status}
}

func numbers(prs []pr.PR) []int {
	var out []int
	for _, p := range prs {
		out = append(out, p.Number)
	}
	sort.Ints(out)
	return out
}

func TestApplyFetch(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "puffin.db"))
	now := time.Unix(1_700_000_000, 0)
	events := func() (n int) {
		t.Helper()
		if err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM event`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	fetch := func(authored, reviews []pr.PR) {
		t.Helper()
		if err := s.ApplyFetch(ctx, now, authored, reviews); err != nil {
			t.Fatal(err)
		}
	}
	asked := pr.PR{Repo: "o/other", Number: 7, Head: "theirs", Base: "main", State: "OPEN",
		Author: "someone", Status: pr.ReviewRequested}

	// Branch a carries a closed pull request and the open one that replaced it.
	fetch([]pr.PR{
		mine(1, "a", "CLOSED", pr.Closed),
		mine(2, "a", "OPEN", pr.Building),
		mine(3, "b", "OPEN", pr.Ready),
	}, []pr.PR{asked})

	authored, err := s.Changes(ctx, RoleAuthor)
	if err != nil {
		t.Fatal(err)
	}
	if got := numbers(authored); !reflect.DeepEqual(got, []int{2, 3}) {
		t.Fatalf("authored = %v, want the open pull request to hold branch a", got)
	}
	if events() != 3 {
		t.Errorf("%d events after the first fetch, want one per new row", events())
	}

	// The same answer again moves nothing.
	fetch([]pr.PR{mine(2, "a", "OPEN", pr.Building), mine(3, "b", "OPEN", pr.Ready)}, []pr.PR{asked})
	if events() != 3 {
		t.Errorf("%d events after an identical fetch", events())
	}

	// A status moves, and the row keeps its identity.
	fetch([]pr.PR{mine(2, "a", "OPEN", pr.Failing), mine(3, "b", "OPEN", pr.Ready)}, []pr.PR{asked})
	var from, to string
	if err := s.DB.QueryRowContext(ctx,
		`SELECT from_value, to_value FROM event ORDER BY id DESC LIMIT 1`).Scan(&from, &to); err != nil {
		t.Fatal(err)
	}
	if from != pr.Building || to != pr.Failing {
		t.Errorf("last event %s -> %s", from, to)
	}

	// Attach a task to 3 and to the review; then everything drops out of the
	// answer. Rows with a task stay in view; the bare one is archived, kept
	// as a record and out of sight.
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO task (slug, dir, flow, created_at) VALUES ('t', '/t', 'work', 'now');
		UPDATE change SET task_id = 1 WHERE pr_number IN (3, 7)`); err != nil {
		t.Fatal(err)
	}
	fetch(nil, nil)
	authored, _ = s.Changes(ctx, RoleAuthor)
	if got := numbers(authored); !reflect.DeepEqual(got, []int{3}) {
		t.Errorf("authored after dropping out = %v", got)
	}
	reviews, _ := s.Changes(ctx, RoleReviewer)
	if len(reviews) != 1 || reviews[0].Status != pr.Reviewed {
		t.Errorf("review after dropping out = %+v", reviews)
	}
	var kept, archived int
	if err := s.DB.QueryRowContext(ctx,
		`SELECT count(*), count(archived_at) FROM change WHERE pr_number = 2`).Scan(&kept, &archived); err != nil {
		t.Fatal(err)
	}
	if kept != 1 || archived != 1 {
		t.Errorf("the pull request with no task: %d rows, %d archived, want it kept and archived", kept, archived)
	}

	// Coming back into the answer brings it back into view.
	fetch([]pr.PR{mine(2, "a", "OPEN", pr.Ready)}, nil)
	authored, _ = s.Changes(ctx, RoleAuthor)
	if got := numbers(authored); !reflect.DeepEqual(got, []int{2, 3}) {
		t.Errorf("authored after one came back = %v", got)
	}
}

func TestMeta(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "puffin.db"))

	var got int
	if ok, err := s.Meta(ctx, "count", &got); ok || err != nil {
		t.Fatalf("unset key: %v, %v", ok, err)
	}
	for _, want := range []int{4, 9} {
		if err := s.SetMeta(ctx, "count", want); err != nil {
			t.Fatal(err)
		}
		if ok, err := s.Meta(ctx, "count", &got); !ok || err != nil || got != want {
			t.Errorf("got %d, %v, %v, want %d", got, ok, err, want)
		}
	}
}
