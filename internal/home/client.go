package home

import (
	"fmt"
	"net/http"
)

// Client handles calls to backend services for the home screen.
type Client struct {
	backendURL string
	httpClient *http.Client
}

// NewClient creates a new home screen backend client.
func NewClient(backendURL string) *Client {
	return &Client{
		backendURL: backendURL,
		httpClient: &http.Client{},
	}
}

// FetchData fetches data from the backend for the home screen.
func (c *Client) FetchData() (map[string]any, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/api/home", c.backendURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	_ = resp
	return map[string]any{}, nil
}
