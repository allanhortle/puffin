package git

import (
	"context"
	"strings"
	"time"

	"github.com/allanhortle/puffin/internal/run"
)

// Checking out a large repository, or fetching one, is slow for good reason.
const slow = 10 * time.Minute

func git(ctx context.Context, clone string, args ...string) (string, error) {
	return run.In(ctx, clone, slow, "git", args...)
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

// AddWorktree checks branch out at path. A branch that exists, here or on
// origin, is used as it is; otherwise it is created from base as origin has
// it now.
func AddWorktree(ctx context.Context, clone, path, branch, base string) error {
	// Best effort: being offline should not stop work from starting.
	git(ctx, clone, "fetch", "origin", base)

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

// RemoveWorktree refuses a worktree with uncommitted work unless forced.
func RemoveWorktree(ctx context.Context, clone, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	_, err := git(ctx, clone, append(args, path)...)
	return err
}

// DeleteBranch deletes a branch only if git considers it merged, so it never
// loses commits that exist nowhere else.
func DeleteBranch(ctx context.Context, clone, branch string) error {
	_, err := git(ctx, clone, "branch", "-d", branch)
	return err
}
