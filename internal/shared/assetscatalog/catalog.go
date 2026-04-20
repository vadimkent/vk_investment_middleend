package assetscatalog

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

const pageSize = 100

// Catalog fetches the full asset list by walking backend pages.
type Catalog struct {
	baseURL    string
	httpClient *http.Client
}

func NewCatalog(baseURL string, timeout time.Duration) *Catalog {
	return &Catalog{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// List fetches every asset across all backend pages. Pages are fetched
// sequentially until offset+pageSize >= total. Partial results are discarded
// if a later page fails. See spec/shared/assets-catalog.md.
func (c *Catalog) List(ctx context.Context, authorization string) ([]Asset, error) {
	offset := 0
	var all []Asset
	for {
		page, err := c.fetchPage(ctx, authorization, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Assets...)
		if offset+pageSize >= page.Total {
			return all, nil
		}
		offset += pageSize
	}
}

func (c *Catalog) fetchPage(ctx context.Context, authorization string, offset int) (*ListPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/assets", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("size", strconv.Itoa(pageSize))
	q.Set("sort", "ticker")
	q.Set("order", "desc")
	q.Set("offset", strconv.Itoa(offset))
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
		page, err := ParseListResponse(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return page, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
