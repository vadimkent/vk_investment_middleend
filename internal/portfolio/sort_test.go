package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortPositions_NonNullByValueDesc(t *testing.T) {
	v1, v2, v3 := 100.0, 500.0, 250.0
	ps := []Position{
		{Ticker: "A", CurrentValue: &v1},
		{Ticker: "B", CurrentValue: &v2},
		{Ticker: "C", CurrentValue: &v3},
	}
	SortPositions(ps)
	assert.Equal(t, []string{"B", "C", "A"}, tickers(ps))
}

func TestSortPositions_TiesByTickerAsc(t *testing.T) {
	v := 100.0
	ps := []Position{
		{Ticker: "BBB", CurrentValue: &v},
		{Ticker: "AAA", CurrentValue: &v},
		{Ticker: "CCC", CurrentValue: &v},
	}
	SortPositions(ps)
	assert.Equal(t, []string{"AAA", "BBB", "CCC"}, tickers(ps))
}

func TestSortPositions_NullsLastByTickerAsc(t *testing.T) {
	v1, v2 := 100.0, 500.0
	ps := []Position{
		{Ticker: "Z"},
		{Ticker: "A", CurrentValue: &v1},
		{Ticker: "M"},
		{Ticker: "B", CurrentValue: &v2},
	}
	SortPositions(ps)
	assert.Equal(t, []string{"B", "A", "M", "Z"}, tickers(ps))
}

func TestSortPositions_EmptyAndSingle(t *testing.T) {
	var empty []Position
	SortPositions(empty)
	assert.Empty(t, empty)

	v := 1.0
	single := []Position{{Ticker: "X", CurrentValue: &v}}
	SortPositions(single)
	assert.Equal(t, "X", single[0].Ticker)
}

func tickers(ps []Position) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Ticker
	}
	return out
}
