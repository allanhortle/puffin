// Package client is the thin half: every command that is not the daemon
// itself asks the daemon over its socket.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"syscall"

	"puffin/internal/api"
)

// ErrNotRunning means nothing is listening on the socket.
var ErrNotRunning = errors.New("the daemon is not running")

type Client struct {
	http *http.Client
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

// PR resolves a query to a pull request. The query's keys are those of the
// daemon's pr endpoint: target, or repo with branch or number.
func (c *Client) PR(ctx context.Context, query url.Values) (api.PR, error) {
	var found api.PR
	return found, c.call(ctx, http.MethodGet, api.PRPath+"?"+query.Encode(), &found)
}

func (c *Client) call(ctx context.Context, method, path string, out any) error {
	// The host is ignored: the transport always dials the socket.
	req, err := http.NewRequestWithContext(ctx, method, "http://puffin"+path, nil)
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
