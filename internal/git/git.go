// Package git answers the two questions puffin asks of a directory: which
// branch is checked out there, and which repository it belongs to.
package git

import (
	"context"
	"regexp"
	"strings"

	"puffin/internal/run"
)

type Info struct {
	Root      string
	Branch    string // empty while HEAD is detached, which happens mid-restack
	CommonDir string // shared by every worktree of one clone
}

// Inspect returns nil when dir is not inside a repository, or no longer
// exists.
func Inspect(ctx context.Context, dir string) *Info {
	if dir == "" {
		return nil
	}
	out, err := run.Output(ctx, "git", "-C", dir, "rev-parse", "--path-format=absolute",
		"--show-toplevel", "--abbrev-ref", "HEAD", "--git-common-dir")
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		return nil
	}
	info := &Info{Root: lines[0], Branch: lines[1], CommonDir: lines[2]}
	if info.Branch == "HEAD" {
		info.Branch = ""
	}
	return info
}

var remote = regexp.MustCompile(`[:/]([^/:]+/[^/]+?)(?:\.git)?$`)

// RepoName is the owner/name behind a clone's origin, or empty. One lookup
// serves every worktree of the clone, since they share a common directory and
// therefore a remote.
func RepoName(ctx context.Context, commonDir string) string {
	out, err := run.Output(ctx, "git", "--git-dir="+commonDir, "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}
	if m := remote.FindStringSubmatch(strings.TrimSpace(out)); m != nil {
		return m[1]
	}
	return ""
}
