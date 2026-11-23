// Package client is the thin half: every command that is not the daemon
// itself asks the daemon over its socket.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"syscall"

	"github.com/allanhortle/puffin/internal/api"
	"github.com/allanhortle/puffin/internal/store"
)

// ErrNotRunning means nothing is listening on the socket.
var ErrNotRunning = errors.New("the daemon is not running")

type Client struct {
	http *http.Client
	// OnStep, when set, is told each step of a slow request as it starts.
	OnStep func(string)
}

func New(socket string) *Client {
	return &Client{http: &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socket)
			},
		},
	}}
}

func (c *Client) Status(ctx context.Context) (api.Status, error) {
	var s api.Status
	return s, c.call(ctx, http.MethodGet, api.StatusPath, &s)
}

func (c *Client) Shutdown(ctx context.Context) error {
	return c.call(ctx, http.MethodPost, api.ShutdownPath, nil)
}

// Refresh runs a poll cycle now. force refetches from GitHub even when
// nothing appears to have moved.
func (c *Client) Refresh(ctx context.Context, force bool) (api.Cycle, error) {
	path := api.RefreshPath
	if force {
		path += "?force=1"
	}
	var cycle api.Cycle
	return cycle, c.call(ctx, http.MethodPost, path, &cycle)
}

func (c *Client) Reviews(ctx context.Context) (api.Reviews, error) {
	var r api.Reviews
	return r, c.call(ctx, http.MethodGet, api.ReviewsPath, &r)
}

// PRs is everything the poller holds.
func (c *Client) PRs(ctx context.Context) (api.PRs, error) {
	var all api.PRs
	return all, c.call(ctx, http.MethodGet, api.PRsPath, &all)
}

// PR resolves a query to a pull request. The query's keys are those of the
// daemon's pr endpoint: target, or repo with branch or number.
func (c *Client) PR(ctx context.Context, query url.Values) (api.PR, error) {
	var found api.PR
	return found, c.call(ctx, http.MethodGet, api.PRPath+"?"+query.Encode(), &found)
}

func (c *Client) Repos(ctx context.Context, scan bool) ([]store.Repo, error) {
	method := http.MethodGet
	if scan {
		method = http.MethodPost
	}
	var repos []store.Repo
	return repos, c.call(ctx, method, api.ReposPath, &repos)
}

func (c *Client) Tasks(ctx context.Context) ([]store.Task, error) {
	var tasks []store.Task
	return tasks, c.call(ctx, http.MethodGet, api.TasksPath, &tasks)
}

func (c *Client) NewTask(ctx context.Context, req api.NewTask) (store.Task, error) {
	var t store.Task
	return t, c.send(ctx, api.TasksPath, req, &t)
}

func (c *Client) Attach(ctx context.Context, req api.Attach) (store.Task, error) {
	var t store.Task
	return t, c.send(ctx, api.AttachPath, req, &t)
}

// Ensure brings a task's session back if it has gone.
func (c *Client) Ensure(ctx context.Context, task string) (store.Task, error) {
	var t store.Task
	return t, c.send(ctx, api.EnsurePath+"?"+url.Values{"task": {task}}.Encode(), nil, &t)
}

func (c *Client) Close(ctx context.Context, task string, force bool) error {
	q := url.Values{"task": {task}}
	if force {
		q.Set("force", "1")
	}
	return c.send(ctx, api.ClosePath+"?"+q.Encode(), nil, nil)
}

// Review makes, or finds, the task for reviewing a pull request.
func (c *Client) Review(ctx context.Context, repo string, number int) (store.Task, error) {
	var t store.Task
	q := url.Values{"repo": {repo}, "number": {strconv.Itoa(number)}}
	return t, c.send(ctx, api.ReviewPath+"?"+q.Encode(), nil, &t)
}

// Recount asks the daemon to recount what is waiting on me and show it.
func (c *Client) Recount(ctx context.Context) error {
	return c.call(ctx, http.MethodPost, api.AttentionPath, nil)
}

func (c *Client) Here(ctx context.Context, req api.Here) (store.Task, error) {
	var t store.Task
	return t, c.send(ctx, api.HerePath, req, &t)
}

func (c *Client) Adopt(ctx context.Context, req api.Adopt) (store.Task, error) {
	var t store.Task
	return t, c.send(ctx, api.AdoptPath, req, &t)
}

func (c *Client) Rename(ctx context.Context, task, to string) (store.Task, error) {
	var t store.Task
	q := url.Values{"task": {task}, "to": {to}}
	return t, c.send(ctx, api.RenamePath+"?"+q.Encode(), nil, &t)
}

func (c *Client) Sweep(ctx context.Context) (api.Swept, error) {
	var swept api.Swept
	return swept, c.send(ctx, api.SweepPath, nil, &swept)
}

// send posts a request that changes a task, with in as its body, and reads
// the stream that comes back: steps as they happen, then the result into out.
func (c *Client) send(ctx context.Context, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		encoded, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://puffin"+path, body)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
			return ErrNotRunning
		}
		return err
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if res.StatusCode >= 400 {
		var e api.Error
		if dec.Decode(&e) == nil && e.Error != "" {
			return errors.New(e.Error)
		}
		return fmt.Errorf("POST %s: %s", path, res.Status)
	}
	for {
		var p api.Progress
		if err := dec.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				return errors.New("the daemon stopped answering before it had finished")
			}
			return err
		}
		switch {
		case !p.Done:
			if c.OnStep != nil {
				c.OnStep(p.Step)
			}
		case p.Error != "":
			return errors.New(p.Error)
		case out == nil:
			return nil
		default:
			return json.Unmarshal(p.Result, out)
		}
	}
}

func (c *Client) call(ctx context.Context, method, path string, out any) error {
	return c.do(ctx, method, path, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	// The host is ignored: the transport always dials the socket.
	req, err := http.NewRequestWithContext(ctx, method, "http://puffin"+path, body)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		// No socket file, or one left by a daemon that was killed.
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
			return ErrNotRunning
		}
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		var e api.Error
		if json.NewDecoder(res.Body).Decode(&e) == nil && e.Error != "" {
			return errors.New(e.Error)
		}
		return fmt.Errorf("%s %s: %s", method, path, res.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
