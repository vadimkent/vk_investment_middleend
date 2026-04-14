package portfolio

import "sort"

// SortPositions sorts in place: non-null CurrentValue first (DESC by value),
// then null CurrentValue. Ties broken by Ticker ASC.
func SortPositions(ps []Position) {
	sort.SliceStable(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		switch {
		case a.CurrentValue != nil && b.CurrentValue == nil:
			return true
		case a.CurrentValue == nil && b.CurrentValue != nil:
			return false
		case a.CurrentValue != nil && b.CurrentValue != nil:
			if *a.CurrentValue != *b.CurrentValue {
				return *a.CurrentValue > *b.CurrentValue
			}
		}
		return a.Ticker < b.Ticker
	})
}
