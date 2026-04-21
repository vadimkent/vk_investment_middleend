package snapshots

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrUnauthorized     = errors.New("backend unauthorized")
	ErrBackend          = errors.New("backend error")
	ErrSnapshotNotFound = errors.New("snapshot not found")
)

// Client talks to the backend /v1/snapshots endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// List calls GET /v1/snapshots, always sending size=10, sort=recorded_at,
// order=desc, offset. Forwards Authorization. Emits is_full_snapshot only when
// the filter is set.
func (c *Client) List(ctx context.Context, authorization string, p ListParams) (*ListResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/snapshots", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("size", "10")
	q.Set("sort", "recorded_at")
	q.Set("order", "desc")
	q.Set("offset", strconv.Itoa(p.Offset))
	if p.IsFullSnapshot != nil {
		q.Set("is_full_snapshot", strconv.FormatBool(*p.IsFullSnapshot))
	}
	req.URL.RawQuery = q.Encode()
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		res, err := ParseListResponse(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return res, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// GetSnapshot fetches a single snapshot by id.
func (c *Client) GetSnapshot(ctx context.Context, authorization, id string) (*Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/snapshots/"+id, nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return ParseSnapshot(body)
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSnapshotNotFound
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
