package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrRegistrationDisabled = errors.New("registration disabled")
	ErrEmailAlreadyExists   = errors.New("email already exists")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type LoginResult struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var out LoginResult
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusUnauthorized:
		return nil, ErrInvalidCredentials
	default:
		return nil, fmt.Errorf("backend login failed: status %d", resp.StatusCode)
	}
}

func (c *Client) Register(ctx context.Context, email, password string) error {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return nil
	case http.StatusForbidden:
		return ErrRegistrationDisabled
	case http.StatusConflict:
		return ErrEmailAlreadyExists
	default:
		return fmt.Errorf("backend register failed: status %d", resp.StatusCode)
	}
}
