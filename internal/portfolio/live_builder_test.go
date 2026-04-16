package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleLiveResp() *PortfolioResponse {
	src := "live"
	priceAsOf := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	qty, avg, total, cur, pnl := 10.0, 100.0, 1000.0, 1200.0, 200.0
	return &PortfolioResponse{
		IsLive:     true,
		PricesAsOf: &priceAsOf,
		Positions: []Position{
			{
				AssetID: "s1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD",
				Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur, UnrealizedPnL: &pnl,
				PriceSource: &src, PriceAsOf: &priceAsOf,
			},
		},
	}
}

func sampleStandardResp() *PortfolioResponse {
	qty, avg, total, cur, pnl := 10.0, 100.0, 1000.0, 1200.0, 200.0
	return &PortfolioResponse{
		IsLive: false,
		Positions: []Position{
			{
				AssetID: "s1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD",
				Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur, UnrealizedPnL: &pnl,
			},
		},
	}
}

func TestBuildLiveDataSection_StandardMode_HasHeaderSummaryFormTable(t *testing.T) {
	resp := sampleStandardResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: false}, []string{"USD"}, "en", time.Now())

	assert.Equal(t, "live-data-section", s.ID)
	assert.NotNil(t, findDescendantByID(s, "live-header-row"))
	assert.NotNil(t, findDescendantByID(s, "portfolio-summary-row"))
	assert.NotNil(t, findDescendantByID(s, "include-closed-form"))
	assert.NotNil(t, findDescendantByID(s, "positions-table-card"))

	// No banner in standard mode
	assert.Nil(t, findDescendantByID(s, "live-banner"))
	// No warnings in standard mode
	assert.Nil(t, findDescendantByID(s, "live-warnings"))
}

func TestBuildLiveDataSection_StandardMode_ToggleIsInactive(t *testing.T) {
	resp := sampleStandardResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: false}, []string{"USD"}, "en", time.Now())

	toggle := findDescendantByID(s, "live-toggle")
	require.NotNil(t, toggle)
	assert.Equal(t, "icon_toggle", toggle.Type)
	assert.Equal(t, false, toggle.Props["active"])

	require.Len(t, toggle.Actions, 2)
	assert.Contains(t, toggle.Actions[0].Endpoint, "live=true")
	assert.Contains(t, toggle.Actions[1].Endpoint, "live=false")
}

func TestBuildLiveDataSection_LiveMode_HasBannerAndDots(t *testing.T) {
	resp := sampleLiveResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: true}, []string{"USD"}, "en", time.Now())

	assert.NotNil(t, findDescendantByID(s, "live-banner"))
	assert.NotNil(t, findDescendantByID(s, "live-status"))
	assert.NotNil(t, findDescendantByID(s, "live-refresh"))
}

func TestBuildLiveDataSection_LiveMode_ToggleIsActive(t *testing.T) {
	resp := sampleLiveResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: true}, []string{"USD"}, "en", time.Now())

	toggle := findDescendantByID(s, "live-toggle")
	require.NotNil(t, toggle)
	assert.Equal(t, "icon_toggle", toggle.Type)
	assert.Equal(t, true, toggle.Props["active"])
}

func TestBuildLiveDataSection_LiveMode_RefreshButtonURL(t *testing.T) {
	resp := sampleLiveResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: true, Refresh: true}, []string{"USD"}, "en", time.Now())

	refresh := findDescendantByID(s, "live-refresh")
	require.NotNil(t, refresh)
	require.Len(t, refresh.Actions, 1)
	assert.Equal(t, "/actions/portfolio/live_data?live=true&refresh=true", refresh.Actions[0].Endpoint)
	assert.Equal(t, "live-data-section", refresh.Actions[0].TargetID)
}

func TestBuildLiveDataSection_LiveMode_WarningsPresent(t *testing.T) {
	resp := sampleLiveResp()
	resp.Warnings = []LiveWarning{
		{AssetID: "w1", Ticker: "DOGE", Error: "provider timeout"},
		{AssetID: "w2", Ticker: "SHIB", Error: "rate limit"},
	}
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: true}, []string{"USD"}, "en", time.Now())

	warnings := findDescendantByID(s, "live-warnings")
	require.NotNil(t, warnings)
	content, _ := warnings.Props["content"].(string)
	assert.Contains(t, content, "DOGE")
	assert.Contains(t, content, "SHIB")
}

func TestBuildLiveDataSection_LiveMode_WarningsAbsentWhenEmpty(t *testing.T) {
	resp := sampleLiveResp()
	resp.Warnings = nil
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: true}, []string{"USD"}, "en", time.Now())

	assert.Nil(t, findDescendantByID(s, "live-warnings"))
}

func TestBuildLiveDataSection_LiveMode_PriceSourceDots(t *testing.T) {
	cases := []struct {
		source      string
		wantColor   string
		wantDotText string
	}{
		{"live", "positive", "●"},
		{"snapshot", "muted", "●"},
		{"none", "negative", "●"},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			src := tc.source
			priceAsOf := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
			cur := 1200.0
			resp := &PortfolioResponse{
				IsLive:     true,
				PricesAsOf: &priceAsOf,
				Positions: []Position{
					{
						AssetID: "p1", Ticker: "TST", Name: "Test", AssetType: "STOCK", Currency: "USD",
						CurrentValue: &cur,
						PriceSource:  &src,
					},
				},
			}
			metrics := ComputeMetrics(resp.Positions, nil)
			s := BuildLiveDataSection(resp, metrics, LiveState{Live: true}, []string{"USD"}, "en", time.Now())

			cell := findDescendantByID(s, "cell-market-value")
			require.NotNil(t, cell)

			content, _ := cell.Props["content"].(string)
			assert.Contains(t, content, "● ", "market value should be prefixed with dot")
			assert.Equal(t, tc.wantColor, cell.Props["color"], "dot color should match price source")
		})
	}
}

func TestBuildLiveDataSection_StandardMode_NoDots(t *testing.T) {
	resp := sampleStandardResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: false}, []string{"USD"}, "en", time.Now())

	cell := findDescendantByID(s, "cell-market-value")
	require.NotNil(t, cell)

	content, _ := cell.Props["content"].(string)
	assert.NotContains(t, content, "● ", "standard mode should not have dot prefix")
	_, hasColor := cell.Props["color"]
	assert.False(t, hasColor, "standard mode market-value cell should have no color")
}

func TestBuildLiveDataSection_ToggleActionsTargetLiveDataSection(t *testing.T) {
	resp := sampleStandardResp()
	metrics := ComputeMetrics(resp.Positions, nil)
	s := BuildLiveDataSection(resp, metrics, LiveState{Live: false}, []string{"USD"}, "en", time.Now())

	toggle := findDescendantByID(s, "live-toggle")
	require.NotNil(t, toggle)
	require.Len(t, toggle.Actions, 2)
	for _, a := range toggle.Actions {
		assert.Equal(t, "live-data-section", a.TargetID)
		assert.Equal(t, "reload", a.Type)
	}
}
