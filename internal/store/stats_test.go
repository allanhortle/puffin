package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/allanhortle/puffin/internal/pr"
)

func TestStats(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "puffin.db"))
	start := time.Unix(1_700_000_000, 0).UTC()
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
	asked := func(status string) pr.PR {
		return pr.PR{Repo: "o/r", Number: 9, Head: "theirs", Base: "main", State: "OPEN", Author: "sam", Status: status}
	}
	fetch := func(minutes int, authored, reviews []pr.PR) {
		t.Helper()
		if err := s.ApplyFetch(ctx, at(minutes), authored, reviews); err != nil {
			t.Fatal(err)
		}
	}

	// 1 was already there when the log began; 2 is seen from its first minute.
	fetch(0, []pr.PR{mine(1, "old", "OPEN", pr.Ready)}, nil)
	fetch(10, []pr.PR{mine(1, "old", "OPEN", pr.Ready), mine(2, "new", "OPEN", pr.Building)}, []pr.PR{asked(pr.ReviewRequested)})
	fetch(20, []pr.PR{mine(1, "old", "MERGED", pr.Merged), mine(2, "new", "OPEN", pr.Failing)}, []pr.PR{asked(pr.ReviewRequested)})
	// GitHub losing track for a moment is not the failure ending.
	fetch(30, []pr.PR{mine(2, "new", "OPEN", pr.Unknown)}, []pr.PR{asked(pr.ReviewRequested)})
	fetch(40, []pr.PR{mine(2, "new", "OPEN", pr.Failing)}, nil)
	fetch(50, []pr.PR{mine(2, "new", "OPEN", pr.WaitingReview)}, nil)
	fetch(110, []pr.PR{mine(2, "new", "MERGED", pr.Merged)}, nil)
	fetch(120, []pr.PR{mine(3, "third", "OPEN", pr.Ready)}, nil)

	got, err := s.Stats(ctx, at(0), at(200))
	if err != nil {
		t.Fatal(err)
	}
	minutes := func(n int) time.Duration { return time.Duration(n) * time.Minute }
	if got.Merged != 2 || got.MergeTime.N != 1 || got.MergeTime.Median != minutes(100) {
		t.Errorf("merged %d, timed %+v; want 2 merged and only the one seen from its start timed, at 100m", got.Merged, got.MergeTime)
	}
	if got.Failures != 1 || got.FailedPRs != 1 || got.FixTime.Median != minutes(30) {
		t.Errorf("failures %d on %d, fixed in %+v; want one failure of 30m", got.Failures, got.FailedPRs, got.FixTime)
	}
	if got.ReviewWait.N != 1 || got.ReviewWait.Median != minutes(60) {
		t.Errorf("review wait %+v, want one of 60m", got.ReviewWait)
	}
	if got.Asked != 1 || got.ReviewTime.N != 1 || got.ReviewTime.Median != minutes(30) {
		t.Errorf("asked %d, reviewed %+v; want one review handed back in 30m", got.Asked, got.ReviewTime)
	}
	if len(got.Open) != 1 || got.Open[0].Word != "ready" || got.Open[0].N != 1 {
		t.Errorf("open = %+v, want the one ready pull request", got.Open)
	}

	// A window that ends before the merge does not count it.
	early, err := s.Stats(ctx, at(0), at(60))
	if err != nil {
		t.Fatal(err)
	}
	if early.Merged != 1 {
		t.Errorf("merged by minute 60 = %d, want 1", early.Merged)
	}
}
