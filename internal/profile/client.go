package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// GetMe → GET /v1/user/me. 401 maps to ErrUnauthorized (session-expiry).
func (c *Client) GetMe(ctx context.Context, authorization string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/user/me", nil)
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
		var me User
		if err := json.Unmarshal(body, &me); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &me, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// UpdateProfile → PATCH /v1/user/me.
func (c *Client) UpdateProfile(ctx context.Context, authorization string, body map[string]any) (*User, error) {
	raw, err := c.doMutation(ctx, http.MethodPatch, "/v1/user/me", authorization, body, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var me User
	if err := json.Unmarshal(raw, &me); err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
	}
	return &me, nil
}

// UpdateEmail → PATCH /v1/user/me/email.
func (c *Client) UpdateEmail(ctx context.Context, authorization, newEmail, currentPassword string) error {
	body := map[string]any{"new_email": newEmail, "current_password": currentPassword}
	_, err := c.doMutation(ctx, http.MethodPatch, "/v1/user/me/email", authorization, body, http.StatusOK)
	return err
}

// ChangePassword → POST /v1/user/me/password.
func (c *Client) ChangePassword(ctx context.Context, authorization, currentPassword, newPassword string) error {
	body := map[string]any{"current_password": currentPassword, "new_password": newPassword}
	_, err := c.doMutation(ctx, http.MethodPost, "/v1/user/me/password", authorization, body, http.StatusNoContent)
	return err
}

// DeleteAccount → DELETE /v1/user/me.
func (c *Client) DeleteAccount(ctx context.Context, authorization, password string) error {
	body := map[string]any{"password": password}
	_, err := c.doMutation(ctx, http.MethodDelete, "/v1/user/me", authorization, body, http.StatusNoContent)
	return err
}

// doMutation handles the request/response envelope for every JSON mutation.
// Per spec: 401 from these mutation endpoints is INVALID_CREDENTIALS, not
// session-expiry — translate it the same way as 4xx into BackendValidationError.
func (c *Client) doMutation(ctx context.Context, method, path, authorization string, body map[string]any, successStatus int) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(buf))
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case successStatus:
		return raw, nil
	case http.StatusUnauthorized, http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusConflict:
		return nil, parseValidationError(raw)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
