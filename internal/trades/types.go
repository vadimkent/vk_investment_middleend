// Package trades implements the Trades SDUI screen (list, create, edit, delete).
package trades

import "encoding/json"

// Trade is the middleend domain representation of a backend trade.
// Money fields (Quantity, PricePerUnit, Fees) stay as strings to preserve
// decimal precision; formatting happens in the builder via internal/shared/format.
type Trade struct {
	ID           string
	AssetID      string
	TradeType    string
	Quantity     string
	PricePerUnit string
	Fees         string
	Date         string
	Source       string
	Notes        string
	CreatedAt    string
}

// ListParams captures the query parameters accepted by the trades list endpoint.
type ListParams struct {
	AssetID   string // "" means no filter; otherwise a UUID
	TradeType string // "" means no filter; otherwise "BUY" or "SELL"
	Offset    int
}

// ListResult wraps the parsed backend list response.
type ListResult struct {
	Trades []Trade
	Total  int
	Size   int
	Offset int
}

type rawTrade struct {
	ID           string `json:"id"`
	AssetID      string `json:"asset_id"`
	TradeType    string `json:"trade_type"`
	Quantity     string `json:"quantity"`
	PricePerUnit string `json:"price_per_unit"`
	Fees         string `json:"fees"`
	Date         string `json:"date"`
	Source       string `json:"source"`
	Notes        string `json:"notes"`
	CreatedAt    string `json:"created_at"`
}

type rawListResponse struct {
	Trades []rawTrade `json:"trades"`
	Total  int        `json:"total"`
	Size   int        `json:"size"`
	Offset int        `json:"offset"`
}

// ParseListResponse parses the backend GET /v1/trades body into a ListResult.
func ParseListResponse(body []byte) (*ListResult, error) {
	var r rawListResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := &ListResult{Total: r.Total, Size: r.Size, Offset: r.Offset}
	out.Trades = make([]Trade, 0, len(r.Trades))
	for _, rt := range r.Trades {
		out.Trades = append(out.Trades, Trade(rt))
	}
	return out, nil
}
