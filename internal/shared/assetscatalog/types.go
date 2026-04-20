// Package assetscatalog provides a full-asset-list helper for SDUI screens
// that need an asset selector (trades, snapshots, import, analysis).
// Unlike internal/assets (which exposes a single paginated page for display),
// this package pages through every backend page and returns a flat slice.
package assetscatalog

import "encoding/json"

// Asset is the minimum surface downstream screens need from the catalog.
// Additional fields returned by the backend pass through untouched via rawAsset.
type Asset struct {
	ID        string
	Ticker    string
	Name      string
	AssetType string
	Currency  string
	IsComplex bool
}

type rawAsset struct {
	ID        string `json:"id"`
	Ticker    string `json:"ticker"`
	Name      string `json:"name"`
	AssetType string `json:"asset_type"`
	Currency  string `json:"currency"`
	IsComplex bool   `json:"is_complex"`
}

type rawListResponse struct {
	Assets []rawAsset `json:"assets"`
	Total  int        `json:"total"`
	Size   int        `json:"size"`
	Offset int        `json:"offset"`
}

// ListPage is one backend page result.
type ListPage struct {
	Assets []Asset
	Total  int
	Size   int
	Offset int
}

// ParseListResponse parses a single /v1/assets page body.
func ParseListResponse(body []byte) (*ListPage, error) {
	var r rawListResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := &ListPage{Total: r.Total, Size: r.Size, Offset: r.Offset}
	out.Assets = make([]Asset, 0, len(r.Assets))
	for _, ra := range r.Assets {
		out.Assets = append(out.Assets, Asset(ra))
	}
	return out, nil
}
