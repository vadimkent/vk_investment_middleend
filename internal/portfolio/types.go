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
}

type rawResponse struct {
	Positions []rawPosition `json:"positions"`
}

// ParsePositions parses the backend /v1/portfolio body into []Position.
func ParsePositions(body []byte) ([]Position, error) {
	var r rawResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := make([]Position, 0, len(r.Positions))
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
		out = append(out, p)
	}
	return out, nil
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
