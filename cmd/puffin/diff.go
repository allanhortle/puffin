package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/allanhortle/puffin/internal/git"
	"github.com/allanhortle/puffin/internal/github"
)

// Fetching a replaced commit takes as long as the network takes.
const diffTimeout = 5 * time.Minute

const (
	sinceBase     = "base"
	sinceReviewed = "reviewed"
)

// origin is somewhere a pull request can be diffed from.
type origin struct {
	Name string // what --since takes: base, reviewed, v2, or a short commit
	SHA  string // empty for the base
	Note string
	At   time.Time
}

// diff shows a pull request's changes in hunk. A pull request I have
// reviewed opens on what its author changed since, because reading the whole
// of a large one again to find that is the cost of reviewing it twice.
func diff(args []string) error {
	flags, rest, err := parse(args, "--since")
	if err != nil {
		return err
	}
	target := ""
	if len(rest) > 0 {
		target = rest[0]
	}
	ctx, cancel := timeout(diffTimeout)
	defer cancel()

	repo, number, _, err := readTarget(ctx, target)
	if err != nil {
		return err
	}
	h, err := github.New().History(ctx, repo, number)
	if err != nil {
		return err
	}
	origins := originsOf(h)
	if _, list := flags["--versions"]; list {
		for _, o := range origins {
			fmt.Println(originLine(o))
		}
		return nil
	}

	since := last(flags["--since"])
	if _, choose := flags["--pick"]; choose {
		if since, err = pickOrigin(h, origins); err != nil || since == "" {
			return err
		}
	}
	from, err := resolveOrigin(h, origins, since)
	if err != nil {
		return err
	}
	if since == "" && h.Reviewed == h.Head {
		fmt.Fprintln(os.Stderr, "puffin: nothing pushed since your review, so this is the whole change")
	}

	dir, err := cloneOf(ctx, repo)
	if err != nil {
		return err
	}
	want := []string{h.Head, h.Base}
	if fullSHA.MatchString(from.SHA) {
		want = append(want, from.SHA)
	}
	if err := git.Fetch(ctx, dir, want...); err != nil {
		return err
	}
	left, clean, err := git.Since(ctx, dir, from.SHA, h.Head, h.Base)
	if err != nil {
		return err
	}
	if !clean {
		fmt.Fprintf(os.Stderr, "puffin: %s collides with what landed on %s since: where it does, the old side has conflict markers and the diff is how they were resolved\n",
			from.Name, h.BaseRef)
	}
	if exec.CommandContext(ctx, "git", "-C", dir, "diff", "--quiet", left, h.Head).Run() == nil {
		fmt.Printf("%s#%d: nothing changed since %s\n", repo, number, from.Name)
		return nil
	}
	cancel()

	if _, err := exec.LookPath("hunk"); err != nil {
		show := exec.Command("git", "diff", left, h.Head)
		show.Dir = dir
		show.Stdin, show.Stdout, show.Stderr = os.Stdin, os.Stdout, os.Stderr
		return show.Run()
	}
	notes, err := hunk(dir, left, h.Head)
	if err != nil || len(notes) == 0 {
		return err
	}
	// A line of the old side only means something to GitHub when the old
	// side is the pull request's own base.
	return postNotes(h, notes, from.SHA == "")
}

var (
	fullSHA    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	versionArg = regexp.MustCompile(`^v(\d+)$`)
)

// originsOf is everywhere h can be diffed from, the likeliest first: my last
// review, the base, then its versions and commits from the newest back. The
// head itself is left out, since nothing has changed since it.
func originsOf(h github.History) []origin {
	var out []origin
	if h.Reviewed != "" && h.Reviewed != h.Head {
		out = append(out, origin{sinceReviewed, h.Reviewed, "your last review", h.ReviewedAt})
	}
	out = append(out, origin{Name: sinceBase, Note: "the whole change, from " + h.BaseRef})
	for i := len(h.Versions) - 2; i >= 0; i-- {
		v := h.Versions[i]
		out = append(out, origin{"v" + strconv.Itoa(i+1), v.SHA, "", v.At})
	}
	for i := len(h.Commits) - 2; i >= 0; i-- {
		c := h.Commits[i]
		out = append(out, origin{short(c.SHA), c.SHA, c.Headline, c.At})
	}
	return out
}

// resolveOrigin turns what --since was given into somewhere to diff from.
// Nothing means my last review when there is one with pushes after it, and
// the base otherwise.
func resolveOrigin(h github.History, origins []origin, since string) (origin, error) {
	if since == "" {
		return origins[0], nil // the review when there is one, or the base
	}
	for _, o := range origins {
		if o.Name == since || (o.SHA != "" && len(since) >= 7 && strings.HasPrefix(o.SHA, since)) {
			return o, nil
		}
	}
	switch {
	case since == sinceReviewed && h.Reviewed == h.Head:
		return origin{}, errors.New("nothing has been pushed since your review")
	case since == sinceReviewed:
		return origin{}, errors.New("you have not reviewed this pull request")
	case versionArg.MatchString(since):
		return origin{}, fmt.Errorf("no version %s to diff from: there are %d, and v%d is what it is now",
			since, len(h.Versions), len(h.Versions))
	}
	// Anything else is git's to make sense of: a commit from before the pull
	// request, a tag, a branch.
	return origin{Name: since, SHA: since}, nil
}

func short(sha string) string { return sha[:min(7, len(sha))] }

func originLine(o origin) string {
	parts := []string{fmt.Sprintf("%-8s", o.Name)}
	if o.SHA != "" && o.Name != short(o.SHA) {
		parts = append(parts, paint("colour242", short(o.SHA)))
	}
	if o.Note != "" {
		parts = append(parts, o.Note)
	}
	if !o.At.IsZero() {
		parts = append(parts, paint("colour242", o.At.Local().Format("Jan 2 15:04")))
	}
	return strings.Join(parts, "  ")
}

// pickOrigin asks where to diff from. Empty means the list was dismissed.
func pickOrigin(h github.History, origins []origin) (string, error) {
	colourful = os.Getenv("NO_COLOR") == "" // fzf reads the colours, whatever stdout is
	lines := make([]string, len(origins))
	for i, o := range origins {
		lines[i] = o.Name + "\t" + originLine(o)
	}
	cmd := exec.Command("fzf", "--ansi", "--no-sort", "--reverse", "--delimiter", "\t", "--with-nth", "2..",
		"--header", fmt.Sprintf("#%d %s: diff from", h.Number, h.Title))
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 130 || exit.ExitCode() == 1) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("fzf: %w", err)
	}
	name, _, _ := strings.Cut(string(out), "\t")
	return strings.TrimSpace(name), nil
}

// cloneOf is a clone of repo to diff in: the one I am standing in when it is
// one, which in a review task is the worktree, and otherwise the main clone.
func cloneOf(ctx context.Context, repo string) (string, error) {
	if dir, err := os.Getwd(); err == nil {
		if info := git.Inspect(ctx, dir); info != nil && strings.EqualFold(git.RepoName(ctx, info.CommonDir), repo) {
			return info.Root, nil
		}
	}
	known, err := daemonClient().Repos(ctx, false)
	if err != nil {
		return "", err
	}
	for _, r := range known {
		if strings.EqualFold(r.Name, repo) && r.Path != "" {
			return r.Path, nil
		}
	}
	return "", fmt.Errorf("no clone of %s to diff in", repo)
}
