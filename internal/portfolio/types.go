package portfolio

import (
	"encoding/json"
	"strconv"
	"time"
)

// Position is the middleend domain representation of a portfolio position,
// parsed from the backend response. Nullable numeric fields use pointers so
// the tree builder can distinguish "missing" from zero.
type Position struct {
	AssetID        string
	Ticker         string
	Name           string
	AssetType      string
	Currency       string
	Quantity       *float64
	AvgCost        *float64
	TotalCost      *float64
	CurrentPrice   *float64
	CurrentValue   *float64
	UnrealizedPnL  *float64
	RealizedPnL    float64
	LastSnapshotAt *time.Time
	PriceSource    *string    // "live", "snapshot", "none"; nil in standard mode
	PriceAsOf      *time.Time // nil in standard mode
}

// LiveWarning represents a warning for an asset whose live price could not be fetched.
type LiveWarning struct {
	AssetID string
	Ticker  string
	Error   string
}

// PortfolioResponse wraps the full backend response, including live metadata.
type PortfolioResponse struct {
	Positions  []Position
	IsLive     bool
	PricesAsOf *time.Time
	Warnings   []LiveWarning
}

type rawPosition struct {
	AssetID        string  `json:"asset_id"`
	Ticker         string  `json:"ticker"`
	Name           string  `json:"name"`
	AssetType      string  `json:"asset_type"`
	Currency       string  `json:"currency"`
	Quantity       *string `json:"quantity"`
	AvgCost        *string `json:"avg_cost"`
	TotalCost      *string `json:"total_cost"`
	CurrentPrice   *string `json:"current_price"`
	CurrentValue   *string `json:"current_value"`
	UnrealizedPnL  *string `json:"unrealized_pnl"`
	RealizedPnL    *string `json:"realized_pnl"`
	LastSnapshotAt *string `json:"last_snapshot_at"`
	PriceSource    *string `json:"price_source"`
	PriceAsOfRaw   *string `json:"price_as_of"`
}

type rawResponse struct {
	Positions  []rawPosition    `json:"positions"`
	IsLive     bool             `json:"is_live"`
	PricesAsOf *string          `json:"prices_as_of"`
	Warnings   []rawLiveWarning `json:"warnings"`
}

type rawLiveWarning struct {
	AssetID string `json:"asset_id"`
	Ticker  string `json:"ticker"`
	Error   string `json:"error"`
}

// ParsePortfolioResponse parses the full backend /v1/portfolio body including
// live metadata.
func ParsePortfolioResponse(body []byte) (*PortfolioResponse, error) {
	var r rawResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	resp := &PortfolioResponse{
		IsLive: r.IsLive,
	}

	if r.PricesAsOf != nil {
		if t, err := time.Parse(time.RFC3339, *r.PricesAsOf); err == nil {
			resp.PricesAsOf = &t
		}
	}

	for _, rw := range r.Warnings {
		resp.Warnings = append(resp.Warnings, LiveWarning{
			AssetID: rw.AssetID,
			Ticker:  rw.Ticker,
			Error:   rw.Error,
		})
	}

	for _, rp := range r.Positions {
		p := Position{
			AssetID:   rp.AssetID,
			Ticker:    rp.Ticker,
			Name:      rp.Name,
			AssetType: rp.AssetType,
			Currency:  rp.Currency,
		}
		p.Quantity = parseFloatPtr(rp.Quantity)
		p.AvgCost = parseFloatPtr(rp.AvgCost)
		p.TotalCost = parseFloatPtr(rp.TotalCost)
		p.CurrentPrice = parseFloatPtr(rp.CurrentPrice)
		p.CurrentValue = parseFloatPtr(rp.CurrentValue)
		p.UnrealizedPnL = parseFloatPtr(rp.UnrealizedPnL)
		if v := parseFloatPtr(rp.RealizedPnL); v != nil {
			p.RealizedPnL = *v
		}
		if rp.LastSnapshotAt != nil {
			if t, err := time.Parse(time.RFC3339, *rp.LastSnapshotAt); err == nil {
				p.LastSnapshotAt = &t
			}
		}
		p.PriceSource = rp.PriceSource
		if rp.PriceAsOfRaw != nil {
			if t, err := time.Parse(time.RFC3339, *rp.PriceAsOfRaw); err == nil {
				p.PriceAsOf = &t
			}
		}
		resp.Positions = append(resp.Positions, p)
	}

	if resp.Positions == nil {
		resp.Positions = []Position{}
	}

	return resp, nil
}

// ParsePositions is a convenience wrapper over ParsePortfolioResponse that
// returns only the positions slice. Existing callers that don't need live
// metadata can continue using this.
func ParsePositions(body []byte) ([]Position, error) {
	resp, err := ParsePortfolioResponse(body)
	if err != nil {
		return nil, err
	}
	return resp.Positions, nil
}

func parseFloatPtr(s *string) *float64 {
	if s == nil {
		return nil
	}
	v, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		return nil
	}
	return &v
}
