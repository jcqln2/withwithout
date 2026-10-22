// Package beacon reads genesis, the slot length, and the client version from a Beacon API.
package beacon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Identity is the consensus client behind this connection.
type Identity struct {
	Name    string
	Version string
}

// Info is the slot schedule and the client that served it.
type Info struct {
	Genesis        time.Time
	SecondsPerSlot uint64
	Client         Identity
	// ClockOffsetMS is collector time minus the HTTP Date header, in milliseconds.
	// Positive means the collector clock is ahead of that header. Date is one-second
	// resolution, so this is a log, not a correction applied to delays.
	ClockOffsetMS int64
}

// Client talks to one Beacon API base URL.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Load reads genesis, SECONDS_PER_SLOT, and the node version.
func (c *Client) Load(ctx context.Context) (Info, error) {
	genesis, offset, err := c.genesis(ctx)
	if err != nil {
		return Info{}, err
	}
	seconds, err := c.secondsPerSlot(ctx)
	if err != nil {
		return Info{}, err
	}
	identity, err := c.Version(ctx)
	if err != nil {
		return Info{}, err
	}
	return Info{
		Genesis:        genesis,
		SecondsPerSlot: seconds,
		Client:         identity,
		ClockOffsetMS:  offset,
	}, nil
}

func (c *Client) genesis(ctx context.Context) (time.Time, int64, error) {
	body, header, err := c.get(ctx, "/eth/v1/beacon/genesis")
	if err != nil {
		return time.Time{}, 0, err
	}
	var payload struct {
		Data struct {
			GenesisTime string `json:"genesis_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return time.Time{}, 0, fmt.Errorf("genesis json: %w", err)
	}
	sec, err := strconv.ParseInt(payload.Data.GenesisTime, 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("genesis_time %q: %w", payload.Data.GenesisTime, err)
	}
	return time.Unix(sec, 0).UTC(), clockOffset(header.Get("Date"), time.Now()), nil
}

func (c *Client) secondsPerSlot(ctx context.Context) (uint64, error) {
	body, _, err := c.get(ctx, "/eth/v1/config/spec")
	if err != nil {
		return 0, err
	}
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, fmt.Errorf("spec json: %w", err)
	}
	raw, ok := payload.Data["SECONDS_PER_SLOT"]
	if !ok {
		return 0, fmt.Errorf("spec missing SECONDS_PER_SLOT")
	}
	switch v := raw.(type) {
	case string:
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			return 0, fmt.Errorf("SECONDS_PER_SLOT %q", v)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("SECONDS_PER_SLOT has type %T", raw)
	}
}

// Version reads /eth/v1/node/version. "teku/v26.10.1/..." becomes name teku and the full string as version.
func (c *Client) Version(ctx context.Context) (Identity, error) {
	body, _, err := c.get(ctx, "/eth/v1/node/version")
	if err != nil {
		return Identity{}, err
	}
	var payload struct {
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Identity{}, fmt.Errorf("version json: %w", err)
	}
	return ParseIdentity(payload.Data.Version), nil
}

// ParseIdentity splits a beacon node version string on the first slash.
func ParseIdentity(version string) Identity {
	version = strings.TrimSpace(version)
	name, _, _ := strings.Cut(version, "/")
	if name == "" {
		name = "unknown"
	}
	return Identity{Name: name, Version: version}
}

// EventsURL is the SSE subscription. topics are comma-separated, which the Hoodi
// ethPandaOps endpoint accepted on 2026-10-22.
func (c *Client) EventsURL(topics []string) (string, error) {
	u, err := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/eth/v1/events")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("topics", strings.Join(topics, ","))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// OpenEvents starts an SSE stream. The caller must close the body.
// The HTTP client used here must not set a total timeout.
func (c *Client) OpenEvents(ctx context.Context, topics []string, httpClient *http.Client) (*http.Response, error) {
	endpoint, err := c.EventsURL(topics)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("events status %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return resp, nil
}

func (c *Client) get(ctx context.Context, path string) ([]byte, http.Header, error) {
	endpoint := strings.TrimRight(c.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("get %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("get %s: status %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	return body, resp.Header.Clone(), nil
}

func clockOffset(dateHeader string, now time.Time) int64 {
	if dateHeader == "" {
		return 0
	}
	server, err := http.ParseTime(dateHeader)
	if err != nil {
		return 0
	}
	return now.Sub(server).Milliseconds()
}
