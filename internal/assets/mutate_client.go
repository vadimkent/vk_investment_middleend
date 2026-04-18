package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// ErrAssetNotFound is returned when the backend responds 404 to a single-asset lookup.
var ErrAssetNotFound = errors.New("asset not found")

// BackendValidationError carries a 4xx validation error from the backend.
// Includes the error code (e.g. ASSET_ALREADY_EXISTS, ASSET_HAS_DATA) and
// a human-readable message.
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

// GetAsset fetches a single asset by id.
func (c *Client) GetAsset(ctx context.Context, authorization, id string) (*Asset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/assets/"+id, nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return c.doAsset(req, http.StatusOK)
}

// CreateAsset posts the given fields to /v1/assets.
func (c *Client) CreateAsset(ctx context.Context, authorization string, body map[string]any) (*Asset, error) {
	return c.doAssetWithBody(ctx, authorization, http.MethodPost, "/v1/assets", body, http.StatusCreated)
}

// UpdateAsset patches an existing asset.
func (c *Client) UpdateAsset(ctx context.Context, authorization, id string, body map[string]any) (*Asset, error) {
	return c.doAssetWithBody(ctx, authorization, http.MethodPatch, "/v1/assets/"+id, body, http.StatusOK)
}

// DeleteAsset deletes an asset, optionally with force.
func (c *Client) DeleteAsset(ctx context.Context, authorization, id string, force bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/v1/assets/"+id, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("force", strconv.FormatBool(force))
	req.URL.RawQuery = q.Encode()
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
		return ErrAssetNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return parseValidationError(rawBody)
	default:
		return fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doAsset(req *http.Request, successStatus int) (*Asset, error) {
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
		var ra rawAsset
		if err := json.Unmarshal(rawBody, &ra); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		a := Asset(ra)
		return &a, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrAssetNotFound
	case http.StatusUnprocessableEntity, http.StatusBadRequest, http.StatusConflict:
		return nil, parseValidationError(rawBody)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}

func (c *Client) doAssetWithBody(ctx context.Context, authorization, method, path string, body map[string]any, successStatus int) (*Asset, error) {
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
	return c.doAsset(req, successStatus)
}

func parseValidationError(body []byte) error {
	var b backendErrorBody
	if err := json.Unmarshal(body, &b); err != nil || b.Error.Code == "" {
		return fmt.Errorf("%w: status 4xx", ErrBackend)
	}
	return &BackendValidationError{Code: b.Error.Code, Message: b.Error.Message}
}
