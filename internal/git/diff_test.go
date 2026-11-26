package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type repo struct {
	t   *testing.T
	dir string
}

func (r repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r repo) commit(file, content string) string {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, file), []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.git("add", file)
	r.git("commit", "-q", "-m", file+": "+strings.TrimSpace(content))
	return r.git("rev-parse", "HEAD")
}

func (r repo) changed(from, to string) string {
	return r.git("diff", "--name-only", from, to)
}

func TestSince(t *testing.T) {
	ctx := context.Background()
	r := repo{t, t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.commit("base.txt", "one\n")

	r.git("checkout", "-q", "-b", "feature")
	v1 := r.commit("feature.txt", "first\n")

	// With nothing reviewed: the whole change.
	from, clean, err := Since(ctx, r.dir, "", v1, "main")
	if err != nil || !clean || r.changed(from, v1) != "feature.txt" {
		t.Fatalf("whole change: from %s clean %v err %v", from, clean, err)
	}

	// Commits added on top: the old head itself.
	v2 := r.commit("more.txt", "more\n")
	if from, _, _ := Since(ctx, r.dir, v1, v2, "main"); from != v1 {
		t.Fatalf("added on top: from %s, want %s", from, v1)
	}

	// The base moves and the branch is rebased onto it with one more change:
	// only that change shows, not what landed on the base.
	r.git("checkout", "-q", "main")
	r.commit("landed.txt", "landed\n")
	r.git("checkout", "-q", "feature")
	r.git("rebase", "-q", "main")
	v3 := r.commit("fix.txt", "fix\n")
	from, clean, err = Since(ctx, r.dir, v2, v3, "main")
	if err != nil || !clean {
		t.Fatalf("rebased: clean %v err %v", clean, err)
	}
	if got := r.changed(from, v3); got != "fix.txt" {
		t.Fatalf("rebased: changed %q, want only fix.txt", got)
	}
	if got := r.changed(v2, v3); !strings.Contains(got, "landed.txt") {
		t.Fatalf("the plain diff should have shown the base's change, got %q", got)
	}

	// The old version collides with the base: say so, and still leave out
	// what else landed there.
	r.git("checkout", "-q", "main")
	r.commit("feature.txt", "theirs\n")
	r.commit("later.txt", "later\n")
	r.git("checkout", "-q", "-B", "feature", "main")
	v4 := r.commit("feature.txt", "resolved\n")
	from, clean, err = Since(ctx, r.dir, v3, v4, "main")
	if err != nil || clean {
		t.Fatalf("conflict: clean %v err %v", clean, err)
	}
	if got := r.changed(from, v4); got != "feature.txt\nfix.txt\nmore.txt" {
		t.Fatalf("conflict: changed %q", got)
	}
}

// In a stack the branch below is rewritten too, so the old version sits on
// commits its base no longer has.
func TestSinceInAStack(t *testing.T) {
	ctx := context.Background()
	r := repo{t, t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.commit("base.txt", "one\n")
	r.git("checkout", "-q", "-b", "below")
	below := r.commit("below.txt", "first\n")
	r.git("checkout", "-q", "-b", "feature")
	v1 := r.commit("feature.txt", "first\n")

	r.git("checkout", "-q", "main")
	r.commit("landed.txt", "landed\n")
	r.git("checkout", "-q", "below")
	r.git("rebase", "-q", "main")
	if err := os.WriteFile(filepath.Join(r.dir, "below.txt"), []byte("second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.git("commit", "-q", "-a", "--amend", "--no-edit")
	r.git("rebase", "-q", "--onto", "below", below, "feature")
	v2 := r.commit("fix.txt", "fix\n")

	from, clean, err := Since(ctx, r.dir, v1, v2, "below")
	if err != nil || !clean {
		t.Fatalf("clean %v err %v", clean, err)
	}
	if got := r.changed(from, v2); got != "fix.txt" {
		t.Fatalf("changed %q, want only fix.txt", got)
	}
}

// Merging the base in, instead of rebasing onto it, brings in just as much.
func TestSinceAcrossAMerge(t *testing.T) {
	ctx := context.Background()
	r := repo{t, t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.commit("base.txt", "one\n")
	r.git("checkout", "-q", "-b", "feature")
	v1 := r.commit("feature.txt", "first\n")
	r.git("checkout", "-q", "main")
	r.commit("landed.txt", "landed\n")
	r.git("checkout", "-q", "feature")
	r.git("merge", "-q", "--no-edit", "main")
	v2 := r.commit("fix.txt", "fix\n")

	from, clean, err := Since(ctx, r.dir, v1, v2, "main")
	if err != nil || !clean {
		t.Fatalf("clean %v err %v", clean, err)
	}
	if got := r.changed(from, v2); got != "fix.txt" {
		t.Fatalf("changed %q, want only fix.txt", got)
	}
}
