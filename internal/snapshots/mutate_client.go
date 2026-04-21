package snapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// BackendValidationError carries a 4xx/5xx-with-code validation error from the
// backend. Code is e.g. "FUTURE_DATED_SNAPSHOT"; Message is the BE's
// human-readable message.
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

// AutoWarning identifies an asset whose provider price fetch failed during an
// auto-snapshot.
type AutoWarning struct {
	AssetID string `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Error   string `json:"error"`
}

// AutoResult is the parsed response of POST /v1/snapshots/auto.
type AutoResult struct {
	Snapshot Snapshot
	Warnings []AutoWarning // nil when the BE omits the field (no warnings)
}

type rawAutoResult struct {
	Snapshot rawSnapshot   `json:"snapshot"`
	Warnings []AutoWarning `json:"warnings"`
}

// CreateSnapshot posts the given body to POST /v1/snapshots.
func (c *Client) CreateSnapshot(ctx context.Context, authorization string, body map[string]any) (*Snapshot, error) {
	return c.doSnapshotWithBody(ctx, authorization, http.MethodPost, "/v1/snapshots", body, http.StatusCreated)
}

// UpdateSnapshot patches an existing snapshot via PATCH /v1/snapshots/:id.
func (c *Client) UpdateSnapshot(ctx context.Context, authorization, id string, body map[string]any) (*Snapshot, error) {
	return c.doSnapshotWithBody(ctx, authorization, http.MethodPatch, "/v1/snapshots/"+id, body, http.StatusOK)
}

// DeleteSnapshot deletes a snapshot by id via DELETE /v1/snapshots/:id.
func (c *Client) DeleteSnapshot(ctx context.Context, authorization, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/snapshots/"+id, nil)
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
	raw, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrSnapshotNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return parseValidationError(raw)
	default:
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// AutoSnapshot triggers POST /v1/snapshots/auto. When notes is non-empty it is
// sent as {"notes": notes}; otherwise the body is {} (no key). Returns the
// created Snapshot plus any per-asset provider warnings.
//
// Terminal failures (NO_PRICE_PROVIDERS_CONFIGURED, ALL_PROVIDERS_FAILED) come
// back as *BackendValidationError when the BE body includes a structured code;
// plain 5xx without a code return ErrBackend.
func (c *Client) AutoSnapshot(ctx context.Context, authorization, notes string) (*AutoResult, error) {
	body := map[string]any{}
	if notes != "" {
		body["notes"] = notes
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/snapshots/auto", bytes.NewReader(buf))
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
	case http.StatusCreated:
		var r rawAutoResult
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &AutoResult{Snapshot: r.Snapshot.toDomain(), Warnings: r.Warnings}, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict,
		http.StatusBadGateway, http.StatusInternalServerError:
		// BE may emit a structured error code even on 5xx (e.g. ALL_PROVIDERS_FAILED).
		ve := parseValidationError(raw)
		if !errors.Is(ve, ErrBackend) {
			return nil, ve
		}
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// doSnapshotWithBody is a shared helper for POST/PATCH endpoints that return a
// Snapshot body on success.
func (c *Client) doSnapshotWithBody(ctx context.Context, authorization, method, path string, body map[string]any, successStatus int) (*Snapshot, error) {
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
		return ParseSnapshot(raw)
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrSnapshotNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return nil, parseValidationError(raw)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

// parseValidationError tries to extract a structured BackendValidationError
// from a 4xx/5xx body. Falls back to ErrBackend if the body is not structured.
func parseValidationError(body []byte) error {
	var b backendErrorBody
	if err := json.Unmarshal(body, &b); err != nil || b.Error.Code == "" {
		return fmt.Errorf("%w: status 4xx", ErrBackend)
	}
	return &BackendValidationError{Code: b.Error.Code, Message: b.Error.Message}
}
