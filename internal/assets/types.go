package assets

import "encoding/json"

// Asset is the middleend domain representation of a backend asset.
type Asset struct {
	ID            string
	Ticker        string
	Name          string
	AssetType     string
	Currency      string
	IsComplex     bool
	PriceProvider *string
}

// ListParams captures the query parameters accepted by both asset endpoints.
type ListParams struct {
	AssetType string // "" means no filter; otherwise one of STOCK/ETF/CRYPTO/BOND
	Offset    int    // non-negative; 0 when unset
}

// ListResult wraps the parsed backend list response.
type ListResult struct {
	Assets []Asset
	Total  int
	Size   int
	Offset int
}

type rawAsset struct {
	ID            string  `json:"id"`
	Ticker        string  `json:"ticker"`
	Name          string  `json:"name"`
	AssetType     string  `json:"asset_type"`
	Currency      string  `json:"currency"`
	IsComplex     bool    `json:"is_complex"`
	PriceProvider *string `json:"price_provider"`
}

type rawListResponse struct {
	Assets []rawAsset `json:"assets"`
	Total  int        `json:"total"`
	Size   int        `json:"size"`
	Offset int        `json:"offset"`
}

// ParseListResponse parses the backend GET /v1/assets body into a ListResult.
func ParseListResponse(body []byte) (*ListResult, error) {
	var r rawListResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := &ListResult{Total: r.Total, Size: r.Size, Offset: r.Offset}
	out.Assets = make([]Asset, 0, len(r.Assets))
	for _, ra := range r.Assets {
		out.Assets = append(out.Assets, Asset{
			ID:            ra.ID,
			Ticker:        ra.Ticker,
			Name:          ra.Name,
			AssetType:     ra.AssetType,
			Currency:      ra.Currency,
			IsComplex:     ra.IsComplex,
			PriceProvider: ra.PriceProvider,
		})
	}
	return out, nil
}
