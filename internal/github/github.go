// Package github asks GitHub for the pull requests that involve one person.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/allanhortle/puffin/internal/run"
)

const (
	notificationsURL = "https://api.github.com/notifications"
	graphqlURL       = "https://api.github.com/graphql"

	// Node's fetch had no timeout of its own, and neither does Go's client by
	// default. A socket that dies while the laptop sleeps must not hang a poll.
	fetchTimeout = 20 * time.Second

	// How far back to look for merged and closed pull requests. A worktree
	// usually outlives the merge, and its letter is the signal that it can go.
	closedWindow = 7 * 24 * time.Hour
)

// Checks and review threads are only asked for on the open side. Requesting
// them for both searches makes GitHub time the whole query out with a 504, and
// a merged or closed pull request renders from its state alone anyway. The
// review-requested side stays light for the same reason: a count needs nothing
// but the count, and issueCount is the true total even when it outruns `first`,
// so the tally cannot silently cap at the page size.
const prQuery = `
query($qOpen: String!, $qClosed: String!, $qReview: String!) {
  rateLimit { cost remaining resetAt }
  open: search(query: $qOpen, type: ISSUE, first: 40) { nodes { ...prFull } }
  closed: search(query: $qClosed, type: ISSUE, first: 30) { nodes { ...prLight } }
  review: search(query: $qReview, type: ISSUE, first: 40) {
    issueCount
    nodes { ...prLight }
  }
}
fragment prLight on PullRequest {
  number title url state headRefName baseRefName headRefOid isDraft
  isCrossRepository headRepositoryOwner { login }
  repository { nameWithOwner }
  author { login }
}
fragment prFull on PullRequest {
  ...prLight
  mergeable mergeStateStatus reviewDecision
  isInMergeQueue mergeQueueEntry { position }
  reviewThreads(first: 100) { nodes { isResolved } }
  commits(last: 1) { nodes { commit { statusCheckRollup {
    state
    contexts(first: 100) { nodes {
      ... on CheckRun {
        name conclusion status startedAt
        checkSuite {
          app { slug }
          workflowRun { databaseId workflow { databaseId } }
        }
      }
      ... on StatusContext { context state description createdAt }
    } }
  } } } }
}`

type Actor struct {
	Login string `json:"login"`
}

// Context is one entry in a commit's check rollup: a CheckRun or a
// StatusContext, with the fields of whichever it is not left empty.
type Context struct {
	// CheckRun
	Name       string      `json:"name"`
	Conclusion string      `json:"conclusion"`
	Status     string      `json:"status"`
	StartedAt  *time.Time  `json:"startedAt"`
	CheckSuite *CheckSuite `json:"checkSuite"`
	// StatusContext
	Context     string     `json:"context"`
	State       string     `json:"state"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"createdAt"`
}

type CheckSuite struct {
	App *struct {
		Slug string `json:"slug"`
	} `json:"app"`
	WorkflowRun *struct {
		DatabaseID *int64 `json:"databaseId"`
		Workflow   *struct {
			DatabaseID *int64 `json:"databaseId"`
		} `json:"workflow"`
	} `json:"workflowRun"`
}

type Rollup struct {
	State    string `json:"state"`
	Contexts struct {
		Nodes []Context `json:"nodes"`
	} `json:"contexts"`
}

// Node is a pull request as the query returns it. The fields below
// IsCrossRepository's group are only present on the open side.
type Node struct {
	Number              int    `json:"number"`
	Title               string `json:"title"`
	URL                 string `json:"url"`
	State               string `json:"state"`
	HeadRefName         string `json:"headRefName"`
	BaseRefName         string `json:"baseRefName"`
	HeadRefOid          string `json:"headRefOid"`
	IsDraft             bool   `json:"isDraft"`
	IsCrossRepository   bool   `json:"isCrossRepository"`
	HeadRepositoryOwner *Actor `json:"headRepositoryOwner"`
	Repository          struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Author *Actor `json:"author"`

	Mergeable        string `json:"mergeable"`
	MergeStateStatus string `json:"mergeStateStatus"`
	ReviewDecision   string `json:"reviewDecision"`
	IsInMergeQueue   bool   `json:"isInMergeQueue"`
	MergeQueueEntry  *struct {
		Position int `json:"position"`
	} `json:"mergeQueueEntry"`
	ReviewThreads *struct {
		Nodes []struct {
			IsResolved bool `json:"isResolved"`
		} `json:"nodes"`
	} `json:"reviewThreads"`
	Commits *struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *Rollup `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

type RateLimit struct {
	Cost      int       `json:"cost"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"resetAt"`
}

type Data struct {
	RateLimit RateLimit `json:"rateLimit"`
	Open      struct {
		Nodes []Node `json:"nodes"`
	} `json:"open"`
	Closed struct {
		Nodes []Node `json:"nodes"`
	} `json:"closed"`
	Review struct {
		IssueCount int    `json:"issueCount"`
		Nodes      []Node `json:"nodes"`
	} `json:"review"`
}

// Heartbeat is what the notification feed said last time, so the next ask can
// be conditional.
//
// The condition is the ETag and not the Last-Modified date. GitHub documents
// If-Modified-Since for this endpoint, but answers it with a 200 and charges
// a rate limit point every time; only If-None-Match gets the free 304.
type Heartbeat struct {
	ETag     string `json:"etag,omitempty"`
	PollSecs int    `json:"poll_secs,omitempty"`
}

type Client struct {
	http *http.Client

	mu    sync.Mutex
	token string
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: fetchTimeout}}
}

// The token is gh's, so puffin has no credentials of its own to manage.
func (c *Client) bearer(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" {
		out, err := run.Output(ctx, "gh", "auth", "token")
		if err != nil {
			return "", fmt.Errorf("gh auth token failed, run `gh auth login`: %w", err)
		}
		c.token = strings.TrimSpace(out)
	}
	return "Bearer " + c.token, nil
}

// A rejected token is dropped so the next call asks gh again, which is how a
// rotated token gets picked up without restarting the daemon.
func (c *Client) forget() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

// Login is the authenticated user's name.
func (c *Client) Login(ctx context.Context) (string, error) {
	out, err := run.Output(ctx, "gh", "api", "user", "-q", ".login")
	if err != nil {
		return "", fmt.Errorf("cannot resolve the current github login: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	auth, err := c.bearer(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized {
		c.forget()
	}
	return res, nil
}

// NotificationsMoved reports whether anything in the notification feed changed
// since hb was last updated, and updates it. A 304 costs no rate limit, so
// this is the cheap half of the poll.
//
// Notifications never report a check going green, a base branch moving, or a
// fresh conflict, so a caller still needs a timed backstop.
func (c *Client) NotificationsMoved(ctx context.Context, hb *Heartbeat) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, notificationsURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if hb.ETag != "" {
		req.Header.Set("If-None-Match", hb.ETag)
	}
	res, err := c.do(ctx, req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body) // so the connection can be reused

	if secs, err := strconv.Atoi(res.Header.Get("X-Poll-Interval")); err == nil && secs > 0 {
		hb.PollSecs = secs
	}
	if res.StatusCode == http.StatusNotModified {
		return false, nil
	}
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("notifications: HTTP %d", res.StatusCode)
	}
	if etag := res.Header.Get("ETag"); etag != "" {
		hb.ETag = etag
	}
	return true, nil
}

// FetchPRs asks for author's open pull requests, their recently closed ones,
// and the open ones waiting on their review. It costs one rate limit point.
func (c *Client) FetchPRs(ctx context.Context, author string, now time.Time) (*Data, error) {
	since := now.Add(-closedWindow).UTC().Format("2006-01-02")
	payload, err := json.Marshal(map[string]any{
		"query": prQuery,
		"variables": map[string]string{
			"qOpen":   "is:pr is:open author:" + author,
			"qClosed": "is:pr is:closed author:" + author + " sort:updated-desc updated:>=" + since,
			// user-review-requested: and not review-requested:, so this only
			// ever finds pull requests that named this person.
			// review-requested: also matches anything aimed at a team they
			// are on, which is the team's queue rather than theirs. Drafts are
			// excluded: a draft is not asking yet. Submitting a review drops
			// the pull request out of the search on its own.
			"qReview": "is:pr is:open draft:false archived:false user-review-requested:" + author +
				" -author:" + author + " sort:updated-desc",
		},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		io.Copy(io.Discard, res.Body)
		return nil, fmt.Errorf("graphql: HTTP %d", res.StatusCode)
	}
	var body struct {
		Data   *Data `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("graphql: %w", err)
	}
	if len(body.Errors) > 0 {
		return nil, fmt.Errorf("graphql: %s", body.Errors[0].Message)
	}
	if body.Data == nil {
		return nil, errors.New("graphql: empty reply")
	}
	return body.Data, nil
}
