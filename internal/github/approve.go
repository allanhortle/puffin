package github

import (
	"context"
	"fmt"
	"strings"
)

const pendingQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      id title state headRefOid
      reviews(states: PENDING, first: 5) { nodes { id viewerDidAuthor comments { totalCount } } }
    }
  }
}`

const approveMutation = `mutation($input: AddPullRequestReviewInput!) {
  addPullRequestReview(input: $input) { pullRequestReview { id } }
}`

const submitMutation = `mutation($input: SubmitPullRequestReviewInput!) {
  submitPullRequestReview(input: $input) { pullRequestReview { id } }
}`

// Approval is what an approval was given to.
type Approval struct {
	Title string
	Head  string // the commit approved
	Notes int    // comments of a pending review that went out with it
}

// Approve approves a pull request as it stands, with body as the review's
// message when there is one. A pending review of mine is what gets
// submitted, so the notes waiting in it go out with the approval; GitHub
// allows one pending review a person, so there is no approving beside it.
func (c *Client) Approve(ctx context.Context, repo string, number int, body string) (Approval, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return Approval{}, fmt.Errorf("%q is not owner/name", repo)
	}
	var reply struct {
		Repository struct {
			PullRequest *struct {
				ID, Title, State, HeadRefOid string
				Reviews                      struct {
					Nodes []struct {
						ID              string
						ViewerDidAuthor bool
						Comments        struct{ TotalCount int }
					}
				}
			}
		}
	}
	err := c.graphqlAny(ctx, pendingQuery,
		map[string]any{"owner": owner, "name": name, "number": number}, &reply)
	if err != nil {
		return Approval{}, err
	}
	p := reply.Repository.PullRequest
	if p == nil {
		return Approval{}, fmt.Errorf("%s has no pull request %d", repo, number)
	}
	if p.State != "OPEN" {
		return Approval{}, fmt.Errorf("%s#%d is %s", repo, number, strings.ToLower(p.State))
	}
	done := Approval{Title: p.Title, Head: p.HeadRefOid}

	var out any
	for _, r := range p.Reviews.Nodes {
		if !r.ViewerDidAuthor {
			continue
		}
		done.Notes = r.Comments.TotalCount
		input := map[string]any{"pullRequestReviewId": r.ID, "event": "APPROVE"}
		if body != "" {
			input["body"] = body
		}
		return done, c.graphqlAny(ctx, submitMutation, map[string]any{"input": input}, &out)
	}
	// Naming the commit means a push that lands while this is on its way is
	// not approved unseen.
	input := map[string]any{"pullRequestId": p.ID, "event": "APPROVE", "commitOID": p.HeadRefOid}
	if body != "" {
		input["body"] = body
	}
	return done, c.graphqlAny(ctx, approveMutation, map[string]any{"input": input}, &out)
}
