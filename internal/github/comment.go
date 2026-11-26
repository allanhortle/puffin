package github

import (
	"context"
	"errors"
	"fmt"
)

// Note is something to say about a line, or a run of lines, of a file in a
// pull request.
type Note struct {
	Path      string
	StartLine int // equal to Line for a single line
	Line      int
	Old       bool // the lines are of the file as it was, not as it is
	Body      string
}

const threadMutation = `mutation($input: AddPullRequestReviewThreadInput!) {
  addPullRequestReviewThread(input: $input) { thread { id } }
}`

// Comment adds a note to my pending review of a pull request, starting the
// review if there is none. A pending review is a draft only I can see, so
// nothing here reaches anyone until it is submitted.
//
// GitHub only takes a comment on a line the pull request's own diff shows.
// One anywhere else, or any at all with onLine false, is attached to the
// file instead, with the line named in its text; onLine reports which
// happened.
func (c *Client) Comment(ctx context.Context, pullRequestID string, n Note, onLine bool) (bool, error) {
	add := func(input map[string]any) error {
		input["pullRequestId"] = pullRequestID
		input["path"] = n.Path
		var reply struct {
			AddPullRequestReviewThread *struct {
				Thread *struct{ ID string }
			}
		}
		if err := c.graphqlAny(ctx, threadMutation, map[string]any{"input": input}, &reply); err != nil {
			return err
		}
		if reply.AddPullRequestReviewThread == nil || reply.AddPullRequestReviewThread.Thread == nil {
			return errors.New("github made no thread")
		}
		return nil
	}

	if onLine {
		side := "RIGHT"
		if n.Old {
			side = "LEFT"
		}
		input := map[string]any{"body": n.Body, "line": n.Line, "side": side, "subjectType": "LINE"}
		if n.StartLine != 0 && n.StartLine != n.Line {
			input["startLine"], input["startSide"] = n.StartLine, side
		}
		if add(input) == nil {
			return true, nil
		}
	}
	where := fmt.Sprintf("Line %d", n.Line)
	if n.StartLine != 0 && n.StartLine != n.Line {
		where = fmt.Sprintf("Lines %d to %d", n.StartLine, n.Line)
	}
	if n.Old {
		where += ", before this change"
	}
	return false, add(map[string]any{"body": where + ": " + n.Body, "subjectType": "FILE"})
}
