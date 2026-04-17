package assets

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
	ErrUnauthorized = errors.New("backend unauthorized")
	ErrBackend      = errors.New("backend error")
)

// Client talks to the backend /v1/assets endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// List calls GET /v1/assets with the caller's Authorization header forwarded
// verbatim. Always sends size=10, sort=ticker, order=desc. Sends asset_type
// and offset when set. Returns ErrUnauthorized on 401, ErrBackend on 5xx,
// network errors, or malformed JSON.
func (c *Client) List(ctx context.Context, authorization string, p ListParams) (*ListResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/assets", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("size", "10")
	q.Set("sort", "ticker")
	q.Set("order", "desc")
	q.Set("offset", strconv.Itoa(p.Offset))
	if p.AssetType != "" {
		q.Set("asset_type", p.AssetType)
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
