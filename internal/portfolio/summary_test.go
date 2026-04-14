package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeMetrics_Empty(t *testing.T) {
	m := ComputeMetrics(nil, nil)
	assert.Equal(t, 0, m.OpenPositions)
	assert.Empty(t, m.TotalValue)
	assert.Empty(t, m.TotalPnL)
	assert.Empty(t, m.Performance)
	assert.Empty(t, m.SnapshotChange)
	assert.Empty(t, m.CurrencyOrder)
}

func TestComputeMetrics_SingleCurrency(t *testing.T) {
	tc := 1000.0
	cur := 1200.0
	pnl := 200.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 50.0},
	}
	m := ComputeMetrics(positions, nil)

	assert.Equal(t, 1, m.OpenPositions)
	assert.InDelta(t, 1200.0, m.TotalValue["USD"], 1e-9)
	assert.InDelta(t, 250.0, m.TotalPnL["USD"], 1e-9)
	require.NotNil(t, m.Performance["USD"])
	assert.InDelta(t, 20.0, *m.Performance["USD"], 1e-9)
	assert.Nil(t, m.SnapshotChange["USD"])
	assert.Equal(t, []string{"USD"}, m.CurrencyOrder)
}

func TestComputeMetrics_MultiCurrencyOrderByTotalValueDesc(t *testing.T) {
	u := 1000.0
	e := 1500.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", CurrentValue: &u},
		{Ticker: "B", Currency: "EUR", CurrentValue: &e},
	}
	m := ComputeMetrics(positions, nil)
	assert.Equal(t, []string{"EUR", "USD"}, m.CurrencyOrder)
}

func TestComputeMetrics_TotalPnLIncludesRealized(t *testing.T) {
	pnl := 100.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", UnrealizedPnL: &pnl, RealizedPnL: 25.0},
		{Ticker: "B", Currency: "USD", RealizedPnL: -10.0}, // only realized
	}
	m := ComputeMetrics(positions, nil)
	assert.InDelta(t, 115.0, m.TotalPnL["USD"], 1e-9)
}

func TestComputeMetrics_PerformanceNilWhenZeroCost(t *testing.T) {
	pnl := 100.0
	zero := 0.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", UnrealizedPnL: &pnl, TotalCost: &zero, RealizedPnL: 0},
	}
	m := ComputeMetrics(positions, nil)
	assert.Nil(t, m.Performance["USD"])
}

func TestComputeMetrics_PerformanceSkipsNullEntries(t *testing.T) {
	tc := 1000.0
	pnl := 200.0
	positions := []Position{
		{Ticker: "A", Currency: "USD", UnrealizedPnL: &pnl, TotalCost: &tc},
		{Ticker: "B", Currency: "USD"}, // null cost and pnl — skipped
	}
	m := ComputeMetrics(positions, nil)
	require.NotNil(t, m.Performance["USD"])
	assert.InDelta(t, 20.0, *m.Performance["USD"], 1e-9)
}

func TestComputeMetrics_SnapshotChangeBasic(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
	}
	m := ComputeMetrics(positions, evo)
	require.NotNil(t, m.SnapshotChange["USD"])
	assert.InDelta(t, 10.0, *m.SnapshotChange["USD"], 1e-9)
}

func TestComputeMetrics_SnapshotChangeTakesLastTwoByDate(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 1100},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 1000},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC), TotalValue: 1050},
	}
	m := ComputeMetrics(positions, evo)
	require.NotNil(t, m.SnapshotChange["USD"])
	// last two by date: 1050 -> 1100 = ~4.76%
	assert.InDelta(t, 4.7619, *m.SnapshotChange["USD"], 0.01)
}

func TestComputeMetrics_SnapshotChangeNilWhenLessThanTwoPoints(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Now(), TotalValue: 1000},
	}
	m := ComputeMetrics(positions, evo)
	assert.Nil(t, m.SnapshotChange["USD"])
}

func TestComputeMetrics_SnapshotChangeNilWhenZeroBase(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 0},
		{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 100},
	}
	m := ComputeMetrics(positions, evo)
	assert.Nil(t, m.SnapshotChange["USD"])
}

func TestComputeMetrics_SnapshotChangeOnlyForActiveCurrencies(t *testing.T) {
	u := 1000.0
	positions := []Position{{Ticker: "A", Currency: "USD", CurrentValue: &u}}
	evo := []EvolutionPoint{
		{Currency: "EUR", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 800},
		{Currency: "EUR", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 900},
	}
	m := ComputeMetrics(positions, evo)
	_, ok := m.SnapshotChange["EUR"]
	assert.False(t, ok)
}
