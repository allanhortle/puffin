package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Description is what a reader of a pull request wants before its diff: what
// it says it is, who is on it, and how big it is.
type Description struct {
	Repo           string
	Number         int
	Title          string
	Body           string
	URL            string
	State          string
	IsDraft        bool
	Author         string
	Base           string
	Head           string
	CreatedAt      time.Time
	Additions      int
	Deletions      int
	ChangedFiles   int
	ReviewDecision string
	Labels         []string
	// Asked is everyone a review is still requested of, people and teams.
	Asked []string
	// Reviews is each reviewer's latest verdict.
	Reviews []Verdict
}

type Verdict struct {
	Login string
	State string // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
}

const describeQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      title body url state isDraft createdAt
      author { login }
      baseRefName headRefName
      additions deletions changedFiles
      reviewDecision
      labels(first: 20) { nodes { name } }
      reviewRequests(first: 20) {
        nodes { requestedReviewer {
          ... on User { login }
          ... on Team { name }
        } }
      }
      latestOpinionatedReviews(first: 20) { nodes { state author { login } } }
    }
  }
}`

// Describe fetches one pull request's description. It is asked for when
// someone sits down to read it, never polled.
func (c *Client) Describe(ctx context.Context, repo string, number int) (Description, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return Description{}, fmt.Errorf("%q is not owner/name", repo)
	}
	var reply struct {
		Repository struct {
			PullRequest *struct {
				Title, Body, URL, State string
				IsDraft                 bool
				CreatedAt               time.Time
				Author                  *Actor
				BaseRefName             string
				HeadRefName             string
				Additions, Deletions    int
				ChangedFiles            int
				ReviewDecision          string
				Labels                  struct{ Nodes []struct{ Name string } }
				ReviewRequests          struct {
					Nodes []struct {
						RequestedReviewer *struct{ Login, Name string }
					}
				}
				LatestOpinionatedReviews struct {
					Nodes []struct {
						State  string
						Author *Actor
					}
				}
			}
		}
	}
	err := c.graphqlAny(ctx, describeQuery,
		map[string]any{"owner": owner, "name": name, "number": number}, &reply)
	if err != nil {
		return Description{}, err
	}
	p := reply.Repository.PullRequest
	if p == nil {
		return Description{}, fmt.Errorf("%s has no pull request %d", repo, number)
	}
	d := Description{
		Repo: repo, Number: number, Title: p.Title, Body: p.Body, URL: p.URL,
		State: p.State, IsDraft: p.IsDraft, CreatedAt: p.CreatedAt,
		Base: p.BaseRefName, Head: p.HeadRefName,
		Additions: p.Additions, Deletions: p.Deletions, ChangedFiles: p.ChangedFiles,
		ReviewDecision: p.ReviewDecision,
	}
	if p.Author != nil {
		d.Author = p.Author.Login
	}
	for _, l := range p.Labels.Nodes {
		d.Labels = append(d.Labels, l.Name)
	}
	for _, r := range p.ReviewRequests.Nodes {
		if who := r.RequestedReviewer; who != nil {
			if who.Login != "" {
				d.Asked = append(d.Asked, who.Login)
			} else if who.Name != "" {
				d.Asked = append(d.Asked, who.Name)
			}
		}
	}
	for _, r := range p.LatestOpinionatedReviews.Nodes {
		if r.Author != nil {
			d.Reviews = append(d.Reviews, Verdict{Login: r.Author.Login, State: r.State})
		}
	}
	return d, nil
}

// Download fetches a file a pull request's description links to. GitHub's
// own attachments on a private repository are only served with a token; the
// token goes nowhere else.
func (c *Client) Download(ctx context.Context, link string, limit int64) ([]byte, error) {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("not a web address: %q", link)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	var res *http.Response
	if ownedByGitHub(u.Host) {
		res, err = c.do(ctx, req)
	} else {
		res, err = c.http.Do(req)
	}
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", link, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s is over %s bytes", link, strconv.FormatInt(limit, 10))
	}
	return body, nil
}

// Go drops the Authorization header itself when a redirect leaves the host,
// so the signed storage link an attachment redirects to never sees it.
func ownedByGitHub(host string) bool {
	return host == "github.com" || strings.HasSuffix(host, ".githubusercontent.com")
}
