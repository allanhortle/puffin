package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/allanhortle/puffin/internal/api"
	"github.com/allanhortle/puffin/internal/github"
	"github.com/allanhortle/puffin/internal/pr"
	"github.com/allanhortle/puffin/internal/store"
	"github.com/allanhortle/puffin/internal/task"
	"github.com/allanhortle/puffin/internal/tmux"
)

const (
	// A 304 on the notification feed costs no rate limit, so the heartbeat
	// runs at GitHub's own advertised cadence. The aggregate query costs a
	// point, so it only runs when the heartbeat moves or a backstop falls due.
	heartbeatInterval = 60 * time.Second
	// Notifications never report a check going green, a base branch moving or
	// a fresh conflict, which is why the idle backstop is not optional.
	backstopIdle = 5 * time.Minute
	backstopBusy = 30 * time.Second
	// Below this many points the aggregate query stops until the limit resets.
	rateLimitFloor = 200
	// Around a whole cycle, for whatever the per-command and per-request
	// deadlines forgot.
	cycleTimeout = 90 * time.Second
	maxBackoff   = 5
	// A sleep that overruns by this much was suspended rather than slept: the
	// machine went away, and everything on the bar is now arbitrarily old.
	wakeSlack = 60 * time.Second
)

// Keys in the meta table.
const (
	metaAuthor      = "author"
	metaFetchedAt   = "fetched_at"
	metaHeartbeat   = "heartbeat"
	metaRateLimit   = "rate_limit"
	metaReviewCount = "review_count"
)

type refreshRequest struct {
	force bool
	reply chan refreshReply
}

type refreshReply struct {
	cycle api.Cycle
	err   error
}

type poller struct {
	store *store.Store
	gh    *github.Client
	tmux  tmux.Client
	tasks *task.Manager

	refresh chan refreshRequest

	// Touched only by the loop.
	windowKey string
	busy      bool
	pollSecs  int

	mu          sync.Mutex
	lastCycleAt *time.Time
	lastError   string
	windows     []api.Window
}

func newPoller(st *store.Store, tm tmux.Client, tasks *task.Manager) *poller {
	return &poller{
		tasks:   tasks,
		store:   st,
		gh:      github.New(),
		tmux:    tm,
		refresh: make(chan refreshRequest),
	}
}

// Refresh runs a cycle now and waits for it.
func (p *poller) Refresh(ctx context.Context, force bool) (api.Cycle, error) {
	req := refreshRequest{force: force, reply: make(chan refreshReply, 1)}
	select {
	case p.refresh <- req:
	case <-ctx.Done():
		return api.Cycle{}, ctx.Err()
	}
	select {
	case r := <-req.reply:
		return r.cycle, r.err
	case <-ctx.Done():
		return api.Cycle{}, ctx.Err()
	}
}

func (p *poller) run(ctx context.Context) {
	failures := 0
	force := false
	wait := time.Duration(0)
	for {
		// Wall clock, with the monotonic reading stripped: the monotonic
		// clock stops while the machine is asleep, and the sleep is exactly
		// what this is here to notice.
		before := time.Now().Round(0)
		timer := time.NewTimer(wait)
		var asked *refreshRequest
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case req := <-p.refresh:
			timer.Stop()
			asked = &req
		case <-timer.C:
			// Suspend and resume, not a slow timer. The notification feed
			// cannot report the hours that just went missing, so skip it.
			if time.Now().Round(0).Sub(before) > wait+wakeSlack {
				force = true
			}
		}

		cycle, err := p.tick(ctx, asked, force)
		if ctx.Err() != nil {
			return
		}
		if asked != nil {
			asked.reply <- refreshReply{cycle, err}
		}
		p.record(err)
		if err != nil {
			failures++
			slog.Error("cycle failed", "err", err)
		} else {
			failures = 0
			force = false
			if cycle.Refetched {
				slog.Info("refetched", "prs", cycle.PRs, "windows", cycle.Windows, "reviews", cycle.Reviews)
			}
		}

		idle := heartbeatInterval
		if p.pollSecs > 0 {
			idle = time.Duration(p.pollSecs) * time.Second
		}
		wait = idle
		// Checks in flight want a tighter loop than the notification
		// heartbeat, which would otherwise hold every cycle at a minute.
		if p.busy {
			wait = min(idle, backstopBusy)
		}
		wait *= time.Duration(min(1<<min(failures, 8), maxBackoff))
	}
}

func (p *poller) tick(ctx context.Context, asked *refreshRequest, force bool) (api.Cycle, error) {
	ctx, cancel := context.WithTimeout(ctx, cycleTimeout)
	defer cancel()

	if asked != nil {
		return p.cycle(ctx, asked.force || force)
	}
	// Nobody is looking, so there is nothing to keep fresh.
	attached, err := p.tmux.Attached(ctx)
	if errors.Is(err, tmux.ErrNoServer) || (err == nil && !attached) {
		return api.Cycle{}, nil
	}
	if err != nil {
		return api.Cycle{}, err
	}
	return p.cycle(ctx, force)
}

func (p *poller) record(err error) {
	now := time.Now().UTC()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastCycleAt = &now
	p.lastError = ""
	if err != nil {
		p.lastError = err.Error()
	}
}

func windowKey(windows []tmux.Window) string {
	parts := make([]string, len(windows))
	for i, w := range windows {
		parts[i] = w.ID + ":" + w.Repo + "#" + w.Branch
	}
	return strings.Join(parts, "|")
}

func (p *poller) cycle(ctx context.Context, force bool) (api.Cycle, error) {
	windows, err := p.tmux.Windows(ctx)
	server := !errors.Is(err, tmux.ErrNoServer)
	if err != nil && server {
		return api.Cycle{}, err
	}
	key := windowKey(windows)

	authored, err := p.store.Changes(ctx, store.RoleAuthor)
	if err != nil {
		return api.Cycle{}, err
	}
	p.busy = anyBusy(authored)

	var fetchedAt time.Time
	var heartbeat github.Heartbeat
	if _, err := p.store.Meta(ctx, metaFetchedAt, &fetchedAt); err != nil {
		return api.Cycle{}, err
	}
	if _, err := p.store.Meta(ctx, metaHeartbeat, &heartbeat); err != nil {
		return api.Cycle{}, err
	}
	p.pollSecs = heartbeat.PollSecs

	backstop := backstopIdle
	if p.busy {
		backstop = backstopBusy
	}
	// A new window may be on a branch whose pull request has never been seen.
	refetch := force || key != p.windowKey || time.Since(fetchedAt) >= backstop
	if !refetch {
		// The heartbeat is only saved once the fetch it calls for succeeds.
		// Saved any earlier, a failed fetch would leave the feed looking
		// quiet and the change it announced unread until the backstop.
		moved, err := p.gh.NotificationsMoved(ctx, &heartbeat)
		if err != nil {
			return api.Cycle{}, err
		}
		p.pollSecs = heartbeat.PollSecs
		refetch = moved
	}

	if refetch {
		if err := p.fetch(ctx, heartbeat); err != nil {
			return api.Cycle{}, err
		}
		if authored, err = p.store.Changes(ctx, store.RoleAuthor); err != nil {
			return api.Cycle{}, err
		}
		p.busy = anyBusy(authored)
	}

	// Not worth failing the cycle over: the letters do not depend on it, and
	// the next cycle tries again.
	if err := p.tasks.Reconcile(ctx); err != nil {
		slog.Error("reconciling tasks failed", "err", err)
	}
	attention, err := p.tasks.Attention(ctx)
	if err != nil {
		slog.Error("counting tasks failed", "err", err)
	}

	var reviews int
	if _, err := p.store.Meta(ctx, metaReviewCount, &reviews); err != nil {
		return api.Cycle{}, err
	}

	// Only my own pull requests get a letter. A colleague's branch checked
	// out for review is not mine to merge, and the letters are about whose
	// move it is on something I am shipping.
	stacks := pr.BuildStacks(authored)
	rendered := map[string]tmux.Rendered{}
	ids := make([]string, len(windows))
	view := make([]api.Window, len(windows))
	for i, w := range windows {
		ids[i] = w.ID
		view[i] = api.Window{ID: w.ID, Repo: w.Repo, Branch: w.Branch, Path: w.Path, Pinned: w.Pinned}
		held, ok := stacks.ByHead(w.Repo, w.Branch)
		if !ok {
			continue
		}
		// Everything shown describes the bottom of the stack and nothing
		// else, threads included, so the whole indicator has one subject.
		bottom := stacks.BottomOf(held)
		label := bottom.Label()
		letter := fmt.Sprintf("#[fg=%s]%s", label.Colour, label.Letter)
		if bottom.Unresolved > 0 {
			letter += fmt.Sprintf("#[fg=red]%d", bottom.Unresolved)
		}
		rendered[w.ID] = tmux.Rendered{Letter: letter, Number: bottom.Number}
		view[i].PR, view[i].Status = held.Number, held.Status
		view[i].Bottom, view[i].BottomStatus = bottom.Number, bottom.Status
		view[i].Stack = stacks.StackOf(held)
		view[i].Unresolved = bottom.Unresolved
	}

	if server {
		if err := p.tmux.Push(ctx, ids, rendered, reviews, attention); err != nil && !errors.Is(err, tmux.ErrNoServer) {
			return api.Cycle{}, err
		}
	}
	p.windowKey = key
	p.mu.Lock()
	p.windows = view
	p.mu.Unlock()

	return api.Cycle{Refetched: refetch, Windows: len(windows), PRs: len(authored), Reviews: reviews}, nil
}

func anyBusy(prs []pr.PR) bool {
	for _, p := range prs {
		if p.State == "OPEN" && p.Busy() {
			return true
		}
	}
	return false
}

func (p *poller) fetch(ctx context.Context, heartbeat github.Heartbeat) error {
	now := time.Now()
	var rate github.RateLimit
	held, err := p.store.Meta(ctx, metaRateLimit, &rate)
	if err != nil {
		return err
	}
	// The remaining count only refreshes on a fetch, and the floor is what
	// stops the fetch, so the hold has to expire on the clock or it never
	// lifts.
	if held && rate.Remaining < rateLimitFloor && now.Before(rate.ResetAt) {
		return fmt.Errorf("rate limit floor reached (%d left, resets %s)",
			rate.Remaining, rate.ResetAt.Local().Format(time.Kitchen))
	}

	var author string
	if ok, err := p.store.Meta(ctx, metaAuthor, &author); err != nil {
		return err
	} else if !ok {
		if author, err = p.gh.Login(ctx); err != nil {
			return err
		}
		if err := p.store.SetMeta(ctx, metaAuthor, author); err != nil {
			return err
		}
	}

	data, err := p.gh.FetchPRs(ctx, author, now)
	if err != nil {
		return err
	}
	var authored, reviews []pr.PR
	for _, n := range data.Open.Nodes {
		if n.Number != 0 {
			authored = append(authored, pr.Authored(n))
		}
	}
	for _, n := range data.Closed.Nodes {
		if n.Number != 0 {
			authored = append(authored, pr.Authored(n))
		}
	}
	requested := map[github.Ref]bool{}
	for _, n := range data.Review.Nodes {
		if n.Number != 0 {
			requested[github.Ref{Repo: n.Repository.NameWithOwner, Number: n.Number}] = true
		}
	}
	// A pull request with a review task is asked after by number, and that
	// answer replaces the search's: it also knows which commit I reviewed.
	watching, err := p.store.WatchedReviews(ctx)
	if err != nil {
		return err
	}
	refs := make([]github.Ref, len(watching))
	for i, w := range watching {
		refs[i] = github.Ref{Repo: w.Repo, Number: w.Number}
	}
	watched, err := p.gh.FetchWatched(ctx, author, refs)
	if err != nil {
		return err
	}
	answered := map[github.Ref]bool{}
	for _, n := range watched {
		ref := github.Ref{Repo: n.Repository.NameWithOwner, Number: n.Number}
		answered[ref] = true
		reviews = append(reviews, pr.Watched(n, requested[ref]))
	}
	for _, n := range data.Review.Nodes {
		if n.Number != 0 && !answered[github.Ref{Repo: n.Repository.NameWithOwner, Number: n.Number}] {
			reviews = append(reviews, pr.ToReview(n))
		}
	}
	if err := p.store.ApplyFetch(ctx, now, authored, reviews); err != nil {
		return err
	}
	for key, value := range map[string]any{
		metaFetchedAt:   now.UTC(),
		metaRateLimit:   data.RateLimit,
		metaHeartbeat:   heartbeat,
		metaReviewCount: data.Review.IssueCount,
	} {
		if err := p.store.SetMeta(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

// snapshot is the poller's half of the status reply.
func (p *poller) snapshot(ctx context.Context) (api.Poll, error) {
	p.mu.Lock()
	poll := api.Poll{LastCycleAt: p.lastCycleAt, LastError: p.lastError, Windows: p.windows}
	p.mu.Unlock()

	var fetchedAt time.Time
	if ok, err := p.store.Meta(ctx, metaFetchedAt, &fetchedAt); err != nil {
		return poll, err
	} else if ok {
		poll.LastFetchAt = &fetchedAt
	}
	var rate github.RateLimit
	if ok, err := p.store.Meta(ctx, metaRateLimit, &rate); err != nil {
		return poll, err
	} else if ok {
		poll.RateRemaining = &rate.Remaining
	}
	if _, err := p.store.Meta(ctx, metaReviewCount, &poll.Reviews); err != nil {
		return poll, err
	}
	authored, err := p.store.Changes(ctx, store.RoleAuthor)
	if err != nil {
		return poll, err
	}
	for _, c := range authored {
		if c.State == "OPEN" {
			poll.Open++
		} else {
			poll.Closed++
		}
	}
	return poll, nil
}

// errNoMatch is a question the daemon understood and has no answer to.
type errNoMatch string

func (e errNoMatch) Error() string { return string(e) }

// find resolves what a person typed, or where they are standing, to a pull
// request. A target is a number, a branch, or the name of a worktree's
// directory. Without one, repo and branch say where the caller is.
func (p *poller) find(ctx context.Context, target, repo, branch string, number int) (api.PR, error) {
	authored, err := p.store.Changes(ctx, store.RoleAuthor)
	if err != nil {
		return api.PR{}, err
	}
	reviews, err := p.store.Changes(ctx, store.RoleReviewer)
	if err != nil {
		return api.PR{}, err
	}
	if n, ok := asNumber(target); ok {
		number = n
	}

	match := func(p pr.PR) bool { return false }
	missing := ""
	switch {
	case number != 0:
		// A number alone can name a pull request in more than one repository;
		// repo, when given, settles it.
		match = func(p pr.PR) bool { return p.Number == number && (repo == "" || p.Repo == repo) }
		missing = fmt.Sprintf("pull request #%d is not one the daemon holds", number)
	case target != "":
		p.mu.Lock()
		for _, w := range p.windows {
			if w.PR != 0 && strings.HasSuffix(w.Path, "/"+target) {
				repo, number = w.Repo, w.PR
			}
		}
		p.mu.Unlock()
		match = func(p pr.PR) bool {
			return p.Head == target || (number != 0 && p.Number == number && p.Repo == repo)
		}
		missing = fmt.Sprintf("no pull request matches %q", target)
	case branch != "":
		match = func(p pr.PR) bool { return p.Repo == repo && p.Head == branch }
		missing = "no pull request for " + branch
	default:
		return api.PR{}, errNoMatch("not on a branch")
	}

	stacks := pr.BuildStacks(authored)
	var found *pr.PR
	// Mine before the ones I am reviewing, and an open one before a closed
	// one on the same branch.
	for _, list := range [][]pr.PR{authored, reviews} {
		for i := range list {
			if match(list[i]) && (found == nil || (found.State != "OPEN" && list[i].State == "OPEN")) {
				found = &list[i]
			}
		}
		if found != nil {
			break
		}
	}
	if found == nil {
		return api.PR{}, errNoMatch(missing)
	}

	out := api.PR{PR: *found, Stack: stacks.StackOf(*found)}
	var fetchedAt time.Time
	if ok, err := p.store.Meta(ctx, metaFetchedAt, &fetchedAt); err != nil {
		return out, err
	} else if ok {
		out.FetchedAt = &fetchedAt
	}
	return out, nil
}

func asNumber(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

func (p *poller) all(ctx context.Context) (api.PRs, error) {
	// Empty lists rather than null, so a reader can iterate without checking.
	out := api.PRs{PRs: []pr.PR{}, Windows: []api.Window{}}
	var err error
	if out.Reviews, err = p.reviews(ctx); err != nil {
		return out, err
	}
	authored, err := p.store.Changes(ctx, store.RoleAuthor)
	if err != nil {
		return out, err
	}
	out.PRs = append(out.PRs, authored...)
	var fetchedAt time.Time
	if ok, err := p.store.Meta(ctx, metaFetchedAt, &fetchedAt); err != nil {
		return out, err
	} else if ok {
		out.FetchedAt = &fetchedAt
	}
	p.mu.Lock()
	out.Windows = append(out.Windows, p.windows...)
	p.mu.Unlock()
	return out, nil
}

func (p *poller) reviews(ctx context.Context) (api.Reviews, error) {
	out := api.Reviews{PRs: []pr.PR{}}
	return out, p.fillReviews(ctx, &out)
}

func (p *poller) fillReviews(ctx context.Context, out *api.Reviews) error {
	if _, err := p.store.Meta(ctx, metaReviewCount, &out.Count); err != nil {
		return err
	}
	all, err := p.store.Changes(ctx, store.RoleReviewer)
	if err != nil {
		return err
	}
	for _, c := range all {
		if c.Status == pr.ReviewRequested {
			out.PRs = append(out.PRs, c)
		}
	}
	return nil
}
