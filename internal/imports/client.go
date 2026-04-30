package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

var (
	ErrUnauthorized    = errors.New("backend unauthorized")
	ErrBackend         = errors.New("backend error")
	ErrSessionNotFound = errors.New("import session not found")
)

// Client talks to the backend /v1/import/sessions, /v1/export, and /v1/restore endpoints.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	// Override the timeout for AI parsing — the backend's Anthropic streaming
	// uses a 5-minute timeout, so the middleend must match (or exceed) it to
	// avoid canceling the upstream request mid-stream and surfacing AI_TIMEOUT.
	const aiAnalysisTimeout = 6 * time.Minute
	if timeout < aiAnalysisTimeout {
		timeout = aiAnalysisTimeout
	}
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// StartSession uploads the file via multipart/form-data and returns the
// parsed Session. Blocks until the backend's AI completes (the BE side is
// synchronous in v1; see the design doc §1).
func (c *Client) StartSession(ctx context.Context, authorization string, fileContent []byte, mediaType, hint string) (*Session, error) {
	body, contentType, err := buildAnalyzeMultipart(fileContent, mediaType, hint)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/import/sessions", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		var s Session
		if err := json.Unmarshal(respBody, &s); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &s, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func buildAnalyzeMultipart(content []byte, mediaType, hint string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// File part with the original media type so the BE can detect format.
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="upload"`)
	if mediaType != "" {
		hdr.Set("Content-Type", mediaType)
	}
	fw, err := w.CreatePart(hdr)
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(content); err != nil {
		return nil, "", err
	}

	if hint != "" {
		if err := w.WriteField("hint", hint); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

// parseBackendError attempts to read {error:{code,message}} from a response
// body. Returns nil when the body is not in that shape.
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

// ResolveGaps PATCHes /v1/import/sessions/:id/gaps with the given resolutions
// and returns the updated session.
func (c *Client) ResolveGaps(ctx context.Context, authorization, sessionID string, resolutions []GapResolution) (*Session, error) {
	payload := struct {
		Resolutions []GapResolution `json:"resolutions"`
	}{Resolutions: resolutions}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		c.baseURL+"/v1/import/sessions/"+sessionID+"/gaps", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var s Session
		if err := json.Unmarshal(respBody, &s); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &s, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSessionNotFound
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// ConfirmSession POSTs /v1/import/sessions/:id/confirm and returns the result.
func (c *Client) ConfirmSession(ctx context.Context, authorization, sessionID string) (*ConfirmResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/import/sessions/"+sessionID+"/confirm", nil)
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
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var r ConfirmResult
		if err := json.Unmarshal(respBody, &r); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &r, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSessionNotFound
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// CancelSession DELETEs the session. 404 is treated as success (idempotent).
func (c *Client) CancelSession(ctx context.Context, authorization, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/v1/import/sessions/"+sessionID, nil)
	if err != nil {
		return err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusNotFound, http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	default:
		respBody, _ := io.ReadAll(resp.Body)
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return be
		}
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// ExportStream calls GET /v1/export and returns the live response. The caller
// is responsible for copying headers and body to the client and for closing
// resp.Body. Unlike other methods, this does not buffer the body — exports
// can be large.
func (c *Client) ExportStream(ctx context.Context, authorization string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/export", nil)
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
	switch resp.StatusCode {
	case http.StatusOK:
		return resp, nil
	case http.StatusUnauthorized:
		_ = resp.Body.Close()
		return nil, ErrUnauthorized
	default:
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// Restore uploads the CSV via multipart and returns the import/skip counts.
func (c *Client) Restore(ctx context.Context, authorization string, fileContent []byte) (*RestoreResult, error) {
	body, contentType, err := buildRestoreMultipart(fileContent)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/restore", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var r RestoreResult
		if err := json.Unmarshal(respBody, &r); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &r, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		if be := parseBackendError(resp.StatusCode, respBody); be != nil {
			return nil, be
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func buildRestoreMultipart(content []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="restore.csv"`)
	hdr.Set("Content-Type", "text/csv")
	fw, err := w.CreatePart(hdr)
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(content); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

// Compile-time guard: keep helper imports referenced even if some methods
// arrive in later tasks. (strconv / strings are used by ResolveGaps and friends.)
var _ = strconv.Itoa
var _ = strings.TrimSpace
