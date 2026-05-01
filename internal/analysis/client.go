package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnauthorized    = errors.New("backend unauthorized")
	ErrBackend         = errors.New("backend error")
	ErrSessionNotFound = errors.New("analysis session not found")
)

// Client streams SSE from the backend's analysis endpoints. Tuned for SSE:
// ResponseHeaderTimeout caps the wait for the upstream to start streaming;
// Client.Timeout is left zero so the body can stream for as long as the
// backend wants. Cancellation is governed by the request context.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a Client. The headerTimeout argument is a safety net for
// cases where the backend never responds; once headers arrive the body can
// stream indefinitely.
func NewClient(baseURL string, headerTimeout time.Duration) *Client {
	if headerTimeout <= 0 {
		headerTimeout = 30 * time.Second
	}
	transport := &http.Transport{
		ResponseHeaderTimeout: headerTimeout,
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Transport: transport},
	}
}

// do issues req and inspects the response status. On 200 it returns the live
// response (caller must close). On error statuses it consumes the body, maps
// to one of the package errors, and returns nil response.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp, nil
	case http.StatusUnauthorized:
		_ = resp.Body.Close()
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		_ = resp.Body.Close()
		return nil, ErrSessionNotFound
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if be := parseBackendError(resp.StatusCode, body); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// parseBackendError reads {error:{code,message}} from a response body.
// Returns nil when the body is empty or not in that shape.
func parseBackendError(httpStatus int, body []byte) *BackendError {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}
	var wrapper struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil
	}
	if wrapper.Error.Code == "" && wrapper.Error.Message == "" {
		return nil
	}
	return &BackendError{HTTPStatus: httpStatus, Code: wrapper.Error.Code, Message: wrapper.Error.Message}
}

// Compile-time guard: keep helper imports referenced even if a method
// arrives in a later task.
var _ = strings.NewReader
var _ = url.PathEscape
var _ = json.Marshal
var _ = context.Background
