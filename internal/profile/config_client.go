package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ConfigClient calls GET /v1/config. Local to the profile package today; lift to
// internal/shared/configcatalog if a second screen needs it.
type ConfigClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewConfigClient(baseURL string, timeout time.Duration) *ConfigClient {
	return &ConfigClient{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

func (c *ConfigClient) GetConfig(ctx context.Context, authorization string) (*AppConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/config", nil)
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
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var cfg AppConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return &cfg, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
