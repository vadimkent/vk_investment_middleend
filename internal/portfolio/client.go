package portfolio

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

// Client talks to the backend /v1/portfolio endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// GetPositions calls GET /v1/portfolio with the caller's Authorization header
// forwarded verbatim and an include_closed query param. Returns
// ErrUnauthorized on 401, ErrBackend on 5xx or malformed response.
func (c *Client) GetPositions(ctx context.Context, authorization string, includeClosed bool) ([]Position, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/portfolio", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("include_closed", strconv.FormatBool(includeClosed))
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
		positions, err := ParsePositions(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return positions, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// GetEvolutionLast calls GET /v1/portfolio/evolution?last=N with the caller's
// Authorization header forwarded verbatim. Same error semantics as GetPositions.
func (c *Client) GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/portfolio/evolution", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("last", fmt.Sprintf("%d", n))
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
		points, err := ParseEvolution(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return points, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
