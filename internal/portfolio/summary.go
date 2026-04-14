package portfolio

import "sort"

// SummaryMetrics holds the aggregated values that feed the five summary cards.
// All per-currency maps are keyed by currency code (e.g. "USD").
type SummaryMetrics struct {
	TotalValue     map[string]float64  // sum of non-null current_value
	TotalPnL       map[string]float64  // sum of non-null unrealized_pnl + sum of realized_pnl
	Performance    map[string]*float64 // Σ unrealized / Σ total_cost × 100; nil when Σ cost == 0 or no data
	SnapshotChange map[string]*float64 // (last.total - prev.total) / prev.total × 100; nil when < 2 points or zero base
	OpenPositions  int
	CurrencyOrder  []string // currencies present in positions, sorted by TotalValue DESC
}

// ComputeMetrics computes the five stat cards' inputs from positions and
// (best-effort) evolution points. Currencies that appear only in evolution
// (not in positions) are ignored.
func ComputeMetrics(positions []Position, evolution []EvolutionPoint) SummaryMetrics {
	m := SummaryMetrics{
		TotalValue:     map[string]float64{},
		TotalPnL:       map[string]float64{},
		Performance:    map[string]*float64{},
		SnapshotChange: map[string]*float64{},
		OpenPositions:  len(positions),
	}

	sumUnrealized := map[string]float64{}
	sumTotalCost := map[string]float64{}
	hasUnrealized := map[string]bool{}
	hasTotalCost := map[string]bool{}
	currencySet := map[string]struct{}{}

	for _, p := range positions {
		if p.CurrentValue != nil {
			m.TotalValue[p.Currency] += *p.CurrentValue
			currencySet[p.Currency] = struct{}{}
		}
		if p.UnrealizedPnL != nil {
			m.TotalPnL[p.Currency] += *p.UnrealizedPnL
			sumUnrealized[p.Currency] += *p.UnrealizedPnL
			hasUnrealized[p.Currency] = true
		}
		m.TotalPnL[p.Currency] += p.RealizedPnL
		if p.TotalCost != nil {
			sumTotalCost[p.Currency] += *p.TotalCost
			hasTotalCost[p.Currency] = true
		}
		currencySet[p.Currency] = struct{}{}
	}

	for c := range currencySet {
		if hasUnrealized[c] && hasTotalCost[c] && sumTotalCost[c] != 0 {
			pct := sumUnrealized[c] / sumTotalCost[c] * 100
			m.Performance[c] = &pct
		} else {
			m.Performance[c] = nil
		}
		m.SnapshotChange[c] = snapshotChangeFor(c, evolution)
	}

	m.CurrencyOrder = currencyOrderByValueDesc(currencySet, m.TotalValue)
	return m
}

func snapshotChangeFor(currency string, evo []EvolutionPoint) *float64 {
	pts := make([]EvolutionPoint, 0, len(evo))
	for _, p := range evo {
		if p.Currency == currency {
			pts = append(pts, p)
		}
	}
	if len(pts) < 2 {
		return nil
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].RecordedAt.Before(pts[j].RecordedAt) })
	prev := pts[len(pts)-2]
	last := pts[len(pts)-1]
	if prev.TotalValue == 0 {
		return nil
	}
	pct := (last.TotalValue - prev.TotalValue) / prev.TotalValue * 100
	return &pct
}

func currencyOrderByValueDesc(set map[string]struct{}, totals map[string]float64) []string {
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		vi, vj := totals[out[i]], totals[out[j]]
		if vi == vj {
			return out[i] < out[j]
		}
		return vi > vj
	})
	return out
}
