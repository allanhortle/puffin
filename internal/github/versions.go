package github

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Version is one state a pull request's branch was pushed in.
type Version struct {
	SHA string
	At  time.Time
}

// Commit is one commit of a pull request as it stands now.
type Commit struct {
	SHA      string
	Headline string
	At       time.Time
}

// History is what a pull request has been: everything needed to diff it
// from some earlier point instead of from its base.
type History struct {
	Repo    string
	Number  int
	ID      string // GitHub's own name for it, which its mutations take
	Title   string
	Head    string // the branch's commit now
	BaseRef string
	Base    string // the base branch's commit now
	// Versions is oldest first, and the last is Head. GitHub only records a
	// push that rewrote the branch, so a version is what a force push
	// replaced: commits added on top in between belong to the version after.
	Versions []Version
	Commits  []Commit
	// Reviewed is the commit my last submitted review was made on, or empty.
	Reviewed   string
	ReviewedAt time.Time
}

const historyQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      id title headRefOid baseRefName baseRefOid
      timelineItems(itemTypes: [HEAD_REF_FORCE_PUSHED_EVENT], first: 100) { nodes {
        ... on HeadRefForcePushedEvent { beforeCommit { oid committedDate } }
      } }
      commits(last: 100) { nodes { commit { oid messageHeadline committedDate } } }
      reviews(last: 50) { nodes { viewerDidAuthor state submittedAt commit { oid } } }
    }
  }
}`

type stamped struct {
	Oid           string
	CommittedDate time.Time
}

// History fetches a pull request's versions, its commits, and where the
// person asking last reviewed it.
func (c *Client) History(ctx context.Context, repo string, number int) (History, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return History{}, fmt.Errorf("%q is not owner/name", repo)
	}
	var reply struct {
		Repository struct {
			PullRequest *struct {
				ID, Title, HeadRefOid, BaseRefName, BaseRefOid string
				TimelineItems                                  struct {
					Nodes []struct{ BeforeCommit *stamped }
				}
				Commits struct {
					Nodes []struct {
						Commit struct {
							stamped
							MessageHeadline string
						}
					}
				}
				Reviews struct {
					Nodes []struct {
						ViewerDidAuthor bool
						State           string
						SubmittedAt     *time.Time
						Commit          *stamped
					}
				}
			}
		}
	}
	err := c.graphqlAny(ctx, historyQuery,
		map[string]any{"owner": owner, "name": name, "number": number}, &reply)
	if err != nil {
		return History{}, err
	}
	p := reply.Repository.PullRequest
	if p == nil {
		return History{}, fmt.Errorf("%s has no pull request %d", repo, number)
	}
	h := History{
		Repo: repo, Number: number, ID: p.ID, Title: p.Title,
		Head: p.HeadRefOid, BaseRef: p.BaseRefName, Base: p.BaseRefOid,
	}
	var headAt time.Time
	for _, n := range p.Commits.Nodes {
		h.Commits = append(h.Commits, Commit{n.Commit.Oid, n.Commit.MessageHeadline, n.Commit.CommittedDate})
		headAt = n.Commit.CommittedDate
	}
	var replaced []Version
	for _, n := range p.TimelineItems.Nodes {
		// GitHub forgets a commit nothing points at, eventually.
		if n.BeforeCommit != nil {
			replaced = append(replaced, Version{n.BeforeCommit.Oid, n.BeforeCommit.CommittedDate})
		}
	}
	h.Versions = versions(replaced, Version{h.Head, headAt})
	for _, r := range p.Reviews.Nodes {
		// A pending review is a draft: nothing has been said yet.
		if r.ViewerDidAuthor && r.State != "PENDING" && r.Commit != nil {
			h.Reviewed = r.Commit.Oid
			if r.SubmittedAt != nil {
				h.ReviewedAt = *r.SubmittedAt
			}
		}
	}
	return h, nil
}

// versions is each commit a force push replaced and then the head, without
// the repeats a push of the same commit twice leaves behind.
func versions(replaced []Version, head Version) []Version {
	var out []Version
	for _, v := range append(replaced, head) {
		if len(out) > 0 && out[len(out)-1].SHA == v.SHA {
			continue
		}
		out = append(out, v)
	}
	return out
}
