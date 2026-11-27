package github

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Comment is one thing one person said.
type Comment struct {
	Author string
	Bot    bool // said by an app, not a person
	Body   string
	At     time.Time
	// Verdict is set on a review's own message: APPROVED, CHANGES_REQUESTED,
	// COMMENTED or DISMISSED.
	Verdict string
}

// Thread is a conversation about a place in the diff.
type Thread struct {
	Path     string
	Line     int    // 0 for a thread about the whole file, or a line since gone
	Hunk     string // the diff the first comment was made on
	Resolved bool
	Outdated bool
	Comments []Comment
}

// Discussion is everything said on a pull request.
type Discussion struct {
	Repo   string
	Number int
	Title  string
	// Conversation is the comments under the pull request and what each
	// review said for itself, oldest first.
	Conversation []Comment
	Threads      []Thread
}

const discussionQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      title
      comments(first: 100) { nodes { author { __typename login } body createdAt } }
      reviews(first: 100) { nodes { author { __typename login } body state submittedAt } }
      reviewThreads(first: 100) { nodes {
        isResolved isOutdated path line originalLine
        comments(first: 50) { nodes { author { __typename login } body createdAt diffHunk } }
      } }
    }
  }
}`

// Discuss fetches what has been said on a pull request.
func (c *Client) Discuss(ctx context.Context, repo string, number int) (Discussion, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return Discussion{}, fmt.Errorf("%q is not owner/name", repo)
	}
	type said struct {
		Author *struct {
			Typename string `json:"__typename"`
			Login    string
		}
		Body        string
		CreatedAt   time.Time
		SubmittedAt *time.Time
		State       string
		DiffHunk    string
	}
	var reply struct {
		Repository struct {
			PullRequest *struct {
				Title         string
				Comments      struct{ Nodes []said }
				Reviews       struct{ Nodes []said }
				ReviewThreads struct {
					Nodes []struct {
						IsResolved, IsOutdated bool
						Path                   string
						Line, OriginalLine     *int
						Comments               struct{ Nodes []said }
					}
				}
			}
		}
	}
	err := c.graphqlAny(ctx, discussionQuery,
		map[string]any{"owner": owner, "name": name, "number": number}, &reply)
	if err != nil {
		return Discussion{}, err
	}
	p := reply.Repository.PullRequest
	if p == nil {
		return Discussion{}, fmt.Errorf("%s has no pull request %d", repo, number)
	}
	comment := func(s said) Comment {
		out := Comment{Body: s.Body, At: s.CreatedAt}
		if s.Author != nil {
			out.Author, out.Bot = s.Author.Login, s.Author.Typename == "Bot"
		}
		return out
	}

	d := Discussion{Repo: repo, Number: number, Title: p.Title}
	for _, s := range p.Comments.Nodes {
		d.Conversation = append(d.Conversation, comment(s))
	}
	for _, s := range p.Reviews.Nodes {
		// A review that only carried comments on lines says nothing of its
		// own; its comments are in the threads. A pending one is unsent.
		if s.SubmittedAt == nil || (s.State == "COMMENTED" && strings.TrimSpace(s.Body) == "") {
			continue
		}
		c := comment(s)
		c.At, c.Verdict = *s.SubmittedAt, s.State
		d.Conversation = append(d.Conversation, c)
	}
	sort.SliceStable(d.Conversation, func(i, j int) bool { return d.Conversation[i].At.Before(d.Conversation[j].At) })

	for _, t := range p.ReviewThreads.Nodes {
		thread := Thread{Path: t.Path, Resolved: t.IsResolved, Outdated: t.IsOutdated}
		if t.Line != nil {
			thread.Line = *t.Line
		} else if t.OriginalLine != nil {
			thread.Line = *t.OriginalLine
		}
		for i, s := range t.Comments.Nodes {
			if i == 0 {
				thread.Hunk = s.DiffHunk
			}
			thread.Comments = append(thread.Comments, comment(s))
		}
		if len(thread.Comments) > 0 {
			d.Threads = append(d.Threads, thread)
		}
	}
	return d, nil
}
