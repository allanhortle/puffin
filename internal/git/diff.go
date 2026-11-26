package git

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

// rev is git for a command whose answer is one name.
func rev(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := git(ctx, dir, args...)
	return strings.TrimSpace(out), err
}

// Fetch brings the named commits into the clone at dir, asking origin only
// for the ones it does not already have. A commit a force push replaced is on
// no branch, and GitHub still serves it by name.
func Fetch(ctx context.Context, dir string, commits ...string) error {
	var missing []string
	for _, c := range commits {
		if _, err := git(ctx, dir, "cat-file", "-e", c+"^{commit}"); err != nil {
			missing = append(missing, c)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	args := append([]string{"fetch", "--quiet", "--no-tags", "origin"}, missing...)
	_, err := git(ctx, dir, args...)
	return err
}

// Since is what to diff head against to see only what its author changed
// after old, where base is the branch both were made for. With no old it is
// where head left base: the whole change.
//
// When head merely adds commits to old, that is old itself. When the branch
// was rebased in between, or base merged into it, comparing the two would
// also show everything that
// landed on base meanwhile, so old's changes are replayed onto head's
// starting point, in memory, and the answer is the tree that leaves.
//
// Where the two collide, which is where the author had to resolve a conflict
// while rebasing, the tree holds git's conflict markers and clean is false:
// those files then show how the conflict was resolved, and the rest is still
// free of base's changes.
func Since(ctx context.Context, dir, old, head, base string) (from string, clean bool, err error) {
	newBase, err := rev(ctx, dir, "merge-base", head, base)
	if err != nil {
		return "", false, err
	}
	if old == "" {
		return newBase, true, nil
	}
	// When head still contains old, nothing was rewritten and where old
	// meets base is where it started. Base may have been merged in since all
	// the same, which brings in as much as a rebase does.
	var oldBase string
	if _, err := rev(ctx, dir, "merge-base", "--is-ancestor", old, head); err == nil {
		oldBase, err = rev(ctx, dir, "merge-base", old, base)
	} else {
		oldBase, err = startOf(ctx, dir, old, base)
	}
	if err != nil {
		return "", false, err
	}
	if oldBase == newBase {
		return old, true, nil // rewritten where it stood
	}
	// The tree is the first line; after a conflict, the files that conflicted
	// follow it.
	out, err := git(ctx, dir, "merge-tree", "--write-tree", "--merge-base="+oldBase, old, newBase)
	tree, _, _ := strings.Cut(out, "\n")
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 && tree != "" {
			return tree, false, nil
		}
		return "", false, err
	}
	return tree, true, nil
}

// Who wrote a commit, when, and its subject: what a rebase carries over.
const identity = "%ae %at %s"

// startOf is the commit old was built on. That is not simply where it meets
// base: in a stack base is the branch below, which is rebased too, so old
// sits on a version of it that base no longer contains, and the two only
// meet far below at the trunk. A rebase keeps a commit's author, date and
// subject, so the start is the first commit under old that base has a copy
// of by those.
func startOf(ctx context.Context, dir, old, base string) (string, error) {
	meet, err := rev(ctx, dir, "merge-base", old, base)
	if err != nil {
		return "", err
	}
	theirs, err := git(ctx, dir, "log", "-n", "2000", "--format="+identity, meet+".."+base)
	if err != nil {
		return "", err
	}
	copies := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(theirs), "\n") {
		copies[line] = true
	}
	mine, err := git(ctx, dir, "log", "--first-parent", "--format=%H "+identity, meet+".."+old)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(mine), "\n") {
		if commit, who, ok := strings.Cut(line, " "); ok && copies[who] {
			return commit, nil
		}
	}
	return meet, nil
}
