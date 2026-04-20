package trades

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrTradeNotFound is returned when the backend responds 404 to a single-trade lookup.
var ErrTradeNotFound = errors.New("trade not found")

// BackendValidationError carries a 4xx validation error from the backend.
// Includes the error code (e.g. INSUFFICIENT_QUANTITY) and a human-readable message.
type BackendValidationError struct {
	Code    string
	Message string
}

func (e *BackendValidationError) Error() string {
	return fmt.Sprintf("backend validation: %s: %s", e.Code, e.Message)
}

type backendErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// GetTrade fetches a single trade by id.
func (c *Client) GetTrade(ctx context.Context, authorization, id string) (*Trade, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/trades/"+id, nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.doTrade(req, http.StatusOK)
}

// CreateTrade posts the given fields to /v1/trades.
func (c *Client) CreateTrade(ctx context.Context, authorization string, body map[string]any) (*Trade, error) {
	return c.doTradeWithBody(ctx, authorization, http.MethodPost, "/v1/trades", body, http.StatusCreated)
}

// UpdateTrade patches an existing trade.
func (c *Client) UpdateTrade(ctx context.Context, authorization, id string, body map[string]any) (*Trade, error) {
	return c.doTradeWithBody(ctx, authorization, http.MethodPatch, "/v1/trades/"+id, body, http.StatusOK)
}

// DeleteTrade deletes a trade by id.
func (c *Client) DeleteTrade(ctx context.Context, authorization, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/trades/"+id, nil)
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
	rawBody, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrTradeNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return parseValidationError(rawBody)
	default:
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doTrade(req *http.Request, successStatus int) (*Trade, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case successStatus:
		var rt rawTrade
		if err := json.Unmarshal(rawBody, &rt); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		t := Trade(rt)
		return &t, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrTradeNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return nil, parseValidationError(rawBody)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doTradeWithBody(ctx context.Context, authorization, method, path string, body map[string]any, successStatus int) (*Trade, error) {
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
	return c.doTrade(req, successStatus)
}

func parseValidationError(body []byte) error {
	var b backendErrorBody
	if err := json.Unmarshal(body, &b); err != nil || b.Error.Code == "" {
		return fmt.Errorf("%w: status 4xx", ErrBackend)
	}
	return &BackendValidationError{Code: b.Error.Code, Message: b.Error.Message}
}
