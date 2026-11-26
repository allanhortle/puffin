package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allanhortle/puffin/internal/run"
)

// Checking out a large repository, or fetching one, is slow for good reason.
const slow = 10 * time.Minute

func git(ctx context.Context, clone string, args ...string) (string, error) {
	return run.In(ctx, clone, slow, "git", args...)
}

// CheckedOut is every branch some worktree of the clone has checked out.
func CheckedOut(ctx context.Context, clone string) (map[string]bool, error) {
	out, err := run.Output(ctx, "git", "-C", clone, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	branches := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if ref, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
			branches[ref] = true
		}
	}
	return branches, nil
}

// HasOwnCommits reports whether branch has any commit origin's base does
// not. It errs towards yes: anything it cannot tell counts as work.
func HasOwnCommits(ctx context.Context, clone, base, branch string) bool {
	out, err := run.Output(ctx, "git", "-C", clone, "rev-list", "--count", "origin/"+base+".."+branch)
	return err != nil || strings.TrimSpace(out) != "0"
}

// DefaultBranch is the branch new work starts from.
func DefaultBranch(ctx context.Context, clone string) string {
	out, err := run.Output(ctx, "git", "-C", clone, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil {
		return strings.TrimPrefix(strings.TrimSpace(out), "origin/")
	}
	if info := Inspect(ctx, clone); info != nil && info.Branch != "" {
		return info.Branch
	}
	return "main"
}

// WorktreeCount is how many worktrees a clone has, itself included.
func WorktreeCount(ctx context.Context, clone string) int {
	out, err := run.Output(ctx, "git", "-C", clone, "worktree", "list")
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(out), "\n"))
}

// BranchExists reports whether a clone has a local branch of that name.
func BranchExists(ctx context.Context, clone, branch string) bool {
	return refExists(ctx, clone, "refs/heads/"+branch)
}

func refExists(ctx context.Context, clone, ref string) bool {
	_, err := run.Output(ctx, "git", "-C", clone, "show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// FetchBase brings the branch new work starts from up to date with origin.
// Best effort: being offline should not stop work from starting.
func FetchBase(ctx context.Context, clone, base string) {
	git(ctx, clone, "fetch", "origin", base)
}

// FetchBranch brings in a branch that already exists on origin: someone
// else's is not here until it is asked for. It is its own connection, because
// asking for a branch that is not there fails the whole fetch.
func FetchBranch(ctx context.Context, clone, branch string) {
	git(ctx, clone, "fetch", "origin", "+refs/heads/"+branch+":refs/remotes/origin/"+branch)
}

// AddWorktree checks branch out at path. A branch that exists, here or on
// origin, is used as it is; otherwise it is created from base as origin had
// it when last fetched.
func AddWorktree(ctx context.Context, clone, path, branch, base string) error {
	var err error
	switch {
	case refExists(ctx, clone, "refs/heads/"+branch):
		_, err = git(ctx, clone, "worktree", "add", path, branch)
	case refExists(ctx, clone, "refs/remotes/origin/"+branch):
		_, err = git(ctx, clone, "worktree", "add", "--track", "-b", branch, path, "origin/"+branch)
	default:
		start := base
		if refExists(ctx, clone, "refs/remotes/origin/"+base) {
			start = "origin/" + base
		}
		_, err = git(ctx, clone, "worktree", "add", "--no-track", "-b", branch, path, start)
	}
	return err
}

// MoveWorktree relocates a worktree, with everything in it that git does not
// track, and tells the clone where it went.
func MoveWorktree(ctx context.Context, clone, from, to string) error {
	_, err := git(ctx, clone, "worktree", "move", from, to)
	return err
}

// RemoveWorktree refuses a worktree with uncommitted work unless forced.
//
// Deleting a worktree is mostly deleting what its build left in it, which
// for a large repository takes many seconds. So the directory is moved into
// trash, which must be on the same disk for that to be instant, git is told
// it has gone, and the deleting goes on after this returns. When it cannot
// be moved, git removes it in its own time.
func RemoveWorktree(ctx context.Context, clone, path, trash string, force bool) error {
	if !force {
		// What `git worktree remove` checks for itself, and how it says so.
		out, err := git(ctx, path, "status", "--porcelain")
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("%s contains modified or untracked files", path)
		}
	}
	gone := filepath.Join(trash, fmt.Sprintf("%s-%d", filepath.Base(path), time.Now().UnixNano()))
	if os.MkdirAll(trash, 0o755) != nil || os.Rename(path, gone) != nil {
		args := []string{"worktree", "remove"}
		if force {
			args = append(args, "--force")
		}
		_, err := git(ctx, clone, append(args, path)...)
		return err
	}
	// All of the trash, so that what a stopped daemon left half deleted
	// goes too.
	go os.RemoveAll(trash)
	_, err := git(ctx, clone, "worktree", "prune")
	return err
}

// DeleteBranch deletes a branch only if git considers it merged, so it never
// loses commits that exist nowhere else.
func DeleteBranch(ctx context.Context, clone, branch string) error {
	_, err := git(ctx, clone, "branch", "-d", branch)
	return err
}
