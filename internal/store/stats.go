package store

import (
	"context"
	"sort"
	"time"

	"github.com/allanhortle/puffin/internal/pr"
)

// Spread is how long something took, over every time it happened.
type Spread struct {
	N       int           `json:"n"`
	Median  time.Duration `json:"median"`
	Longest time.Duration `json:"longest"`
}

func spread(all []time.Duration) Spread {
	if len(all) == 0 {
		return Spread{}
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	mid := all[len(all)/2]
	if len(all)%2 == 0 {
		mid = (all[len(all)/2-1] + mid) / 2
	}
	return Spread{N: len(all), Median: mid, Longest: all[len(all)-1]}
}

// Stats is what the event log says about a stretch of time. Everything in it
// is read off status changes as the poller saw them, so a time is good to
// within a poll, and nothing from before the log began is in it.
type Stats struct {
	Since    time.Time `json:"since"`
	Until    time.Time `json:"until"`
	LogStart time.Time `json:"log_start"`

	// My pull requests that merged, and how long each was open. Only those
	// first seen after the log began have a known length.
	Merged    int    `json:"merged"`
	MergeTime Spread `json:"merge_time"`
	// My open pull requests now, counted by the word each shows.
	Open []Count `json:"open"`

	// Times a pull request of mine went red, on how many of them, and how
	// long each stayed that way.
	Failures  int    `json:"failures"`
	FailedPRs int    `json:"failed_prs"`
	FixTime   Spread `json:"fix_time"`
	// How long mine sat with a reviewer each time.
	ReviewWait Spread `json:"review_wait"`

	// Reviews asked of me, and how long each took me to hand back.
	Asked      int    `json:"asked"`
	ReviewTime Spread `json:"review_time"`

	TasksStarted   int `json:"tasks_started"`
	TasksDone      int `json:"tasks_done"`
	TasksDiscarded int `json:"tasks_discarded"`
}

type Count struct {
	Word   string `json:"word"`
	Colour string `json:"colour"`
	N      int    `json:"n"`
}

type moment struct {
	at     time.Time
	status string
}

// Stats reads the log between since and until.
func (s *Store) Stats(ctx context.Context, since, until time.Time) (Stats, error) {
	out := Stats{Since: since, Until: until}
	within := func(t time.Time) bool { return !t.Before(since) && !t.After(until) }

	var first string
	if err := s.DB.QueryRowContext(ctx, `SELECT coalesce(min(at), '') FROM event`).Scan(&first); err != nil {
		return out, err
	}
	if first == "" {
		return out, nil
	}
	out.LogStart, _ = time.Parse(time.RFC3339, first)

	// UNKNOWN is GitHub not having worked something out yet. The pull request
	// is still whatever it was, so those are not moments of their own.
	rows, err := s.DB.QueryContext(ctx, `
		SELECT e.change_id, c.role, e.at, e.to_value
		FROM event e JOIN change c ON c.id = e.change_id
		WHERE e.kind = 'status' AND e.to_value != ? AND e.at <= ?
		ORDER BY e.change_id, e.at, e.id`, pr.Unknown, stamp(until))
	if err != nil {
		return out, err
	}
	defer rows.Close()
	history := map[int64][]moment{}
	roles := map[int64]string{}
	var order []int64
	for rows.Next() {
		var id int64
		var role, at, status string
		if err := rows.Scan(&id, &role, &at, &status); err != nil {
			return out, err
		}
		when, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return out, err
		}
		h := history[id]
		if len(h) > 0 && h[len(h)-1].status == status {
			continue
		}
		if len(h) == 0 {
			order = append(order, id)
			roles[id] = role
		}
		history[id] = append(h, moment{when, status})
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	var merge, fix, wait, review []time.Duration
	for _, id := range order {
		h := history[id]
		failed := false
		for i, m := range h {
			// What it was before this moment, and for how long.
			var was string
			var held time.Duration
			if i > 0 {
				was, held = h[i-1].status, m.at.Sub(h[i-1].at)
			}
			if !within(m.at) {
				continue
			}
			if roles[id] == RoleReviewer {
				if m.status == pr.ReviewRequested {
					out.Asked++
				}
				if m.status == pr.Reviewed && (was == pr.ReviewRequested || was == pr.ReviewUpdated) {
					review = append(review, held)
				}
				continue
			}
			switch m.status {
			case pr.Merged:
				out.Merged++
				if h[0].at.After(out.LogStart) {
					merge = append(merge, m.at.Sub(h[0].at))
				}
			case pr.Failing:
				out.Failures++
				failed = true
			}
			switch was {
			case pr.Failing:
				fix = append(fix, held)
			case pr.WaitingReview:
				wait = append(wait, held)
			}
		}
		if failed {
			out.FailedPRs++
		}
	}
	out.MergeTime, out.FixTime, out.ReviewWait, out.ReviewTime = spread(merge), spread(fix), spread(wait), spread(review)

	open, err := s.DB.QueryContext(ctx, `
		SELECT status, is_draft, count(*) FROM change
		WHERE role = 'author' AND state = 'OPEN' AND pr_number IS NOT NULL AND archived_at IS NULL
		GROUP BY status, is_draft`)
	if err != nil {
		return out, err
	}
	defer open.Close()
	byWord := map[string]*Count{}
	for open.Next() {
		var status string
		var isDraft bool
		var n int
		if err := open.Scan(&status, &isDraft, &n); err != nil {
			return out, err
		}
		label := pr.PR{Status: status, IsDraft: isDraft}.Label()
		if byWord[label.Word] == nil {
			byWord[label.Word] = &Count{Word: label.Word, Colour: label.Colour}
		}
		byWord[label.Word].N += n
	}
	if err := open.Err(); err != nil {
		return out, err
	}
	for _, c := range byWord {
		out.Open = append(out.Open, *c)
	}
	sort.Slice(out.Open, func(i, j int) bool {
		if out.Open[i].N != out.Open[j].N {
			return out.Open[i].N > out.Open[j].N
		}
		return out.Open[i].Word < out.Open[j].Word
	})

	tasks, err := s.DB.QueryContext(ctx, `
		SELECT to_value, count(*) FROM event
		WHERE kind = 'task' AND at >= ? AND at <= ? GROUP BY to_value`, stamp(since), stamp(until))
	if err != nil {
		return out, err
	}
	defer tasks.Close()
	for tasks.Next() {
		var to string
		var n int
		if err := tasks.Scan(&to, &n); err != nil {
			return out, err
		}
		switch to {
		case "open":
			out.TasksStarted = n
		case "done":
			out.TasksDone = n
		case "discarded":
			out.TasksDiscarded = n
		}
	}
	return out, tasks.Err()
}
