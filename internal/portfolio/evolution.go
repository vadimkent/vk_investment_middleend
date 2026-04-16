package portfolio

import (
	"encoding/json"
	"strconv"
	"time"
)

// EvolutionPoint is one (snapshot, currency) value from the backend
// /v1/portfolio/evolution endpoint.
type EvolutionPoint struct {
	SnapshotID     string
	RecordedAt     time.Time
	IsFullSnapshot bool
	TotalValue     float64
	TotalCost      *float64 // nil when not computable for the snapshot
	Currency       string
}

type rawEvolutionPoint struct {
	SnapshotID     string  `json:"snapshot_id"`
	RecordedAt     string  `json:"recorded_at"`
	IsFullSnapshot bool    `json:"is_full_snapshot"`
	TotalValue     string  `json:"total_value"`
	TotalCost      *string `json:"total_cost"`
	Currency       string  `json:"currency"`
}

type rawEvolutionResponse struct {
	Evolution []rawEvolutionPoint `json:"evolution"`
}

// ParseEvolution parses the backend /v1/portfolio/evolution body.
func ParseEvolution(body []byte) ([]EvolutionPoint, error) {
	var r rawEvolutionResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := make([]EvolutionPoint, 0, len(r.Evolution))
	for _, rp := range r.Evolution {
		p := EvolutionPoint{
			SnapshotID:     rp.SnapshotID,
			IsFullSnapshot: rp.IsFullSnapshot,
			Currency:       rp.Currency,
		}
		if v, err := strconv.ParseFloat(rp.TotalValue, 64); err == nil {
			p.TotalValue = v
		}
		if rp.TotalCost != nil {
			if v, err := strconv.ParseFloat(*rp.TotalCost, 64); err == nil {
				p.TotalCost = &v
			}
		}
		if t, err := time.Parse(time.RFC3339, rp.RecordedAt); err == nil {
			p.RecordedAt = t
		}
		out = append(out, p)
	}
	return out, nil
}
