package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePositions_AllFieldsSet(t *testing.T) {
	raw := []byte(`{
      "positions":[
        {
          "asset_id":"a1","ticker":"AAPL","name":"Apple Inc","asset_type":"STOCK","currency":"USD",
          "quantity":"10","avg_cost":"153.33","total_cost":"1533.33",
          "current_price":"185.50","current_value":"1855.00",
          "unrealized_pnl":"321.67","realized_pnl":"175.00",
          "last_snapshot_at":"2024-06-01T10:00:00Z"
        }
      ]
    }`)

	positions, err := ParsePositions(raw)
	require.NoError(t, err)
	require.Len(t, positions, 1)

	p := positions[0]
	assert.Equal(t, "a1", p.AssetID)
	assert.Equal(t, "AAPL", p.Ticker)
	assert.Equal(t, "Apple Inc", p.Name)
	assert.Equal(t, "STOCK", p.AssetType)
	assert.Equal(t, "USD", p.Currency)

	require.NotNil(t, p.Quantity)
	assert.InDelta(t, 10.0, *p.Quantity, 1e-9)
	require.NotNil(t, p.AvgCost)
	assert.InDelta(t, 153.33, *p.AvgCost, 1e-9)
	require.NotNil(t, p.TotalCost)
	assert.InDelta(t, 1533.33, *p.TotalCost, 1e-9)
	require.NotNil(t, p.CurrentPrice)
	require.NotNil(t, p.CurrentValue)
	assert.InDelta(t, 1855.0, *p.CurrentValue, 1e-9)
	require.NotNil(t, p.UnrealizedPnL)
	assert.InDelta(t, 321.67, *p.UnrealizedPnL, 1e-9)
	assert.InDelta(t, 175.0, p.RealizedPnL, 1e-9)

	require.NotNil(t, p.LastSnapshotAt)
	assert.Equal(t, time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC), *p.LastSnapshotAt)
}

func TestParsePositions_NullsAndComplexAsset(t *testing.T) {
	raw := []byte(`{
      "positions":[
        {
          "asset_id":"a2","ticker":"REAL-ESTATE","name":"Apartment","asset_type":"COMPLEX","currency":"USD",
          "quantity":null,"avg_cost":null,"total_cost":null,
          "current_price":null,"current_value":"100000.00",
          "unrealized_pnl":null,"realized_pnl":"0",
          "last_snapshot_at":null
        }
      ]
    }`)

	positions, err := ParsePositions(raw)
	require.NoError(t, err)
	require.Len(t, positions, 1)

	p := positions[0]
	assert.Nil(t, p.Quantity)
	assert.Nil(t, p.AvgCost)
	assert.Nil(t, p.TotalCost)
	assert.Nil(t, p.CurrentPrice)
	require.NotNil(t, p.CurrentValue)
	assert.InDelta(t, 100000.0, *p.CurrentValue, 1e-9)
	assert.Nil(t, p.UnrealizedPnL)
	assert.Equal(t, 0.0, p.RealizedPnL)
	assert.Nil(t, p.LastSnapshotAt)
}

func TestParsePositions_EmptyArray(t *testing.T) {
	raw := []byte(`{"positions":[]}`)
	positions, err := ParsePositions(raw)
	require.NoError(t, err)
	assert.Empty(t, positions)
}

func TestParsePositions_InvalidJSON(t *testing.T) {
	_, err := ParsePositions([]byte(`not json`))
	require.Error(t, err)
}

func TestParsePortfolioResponse_LiveFields(t *testing.T) {
	raw := []byte(`{
	  "positions":[
	    {
	      "asset_id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD",
	      "quantity":"10","avg_cost":"150","total_cost":"1500",
	      "current_price":"180","current_value":"1800",
	      "unrealized_pnl":"300","realized_pnl":"0",
	      "last_snapshot_at":"2024-06-01T10:00:00Z",
	      "price_source":"live",
	      "price_as_of":"2026-04-14T12:00:00Z"
	    }
	  ],
	  "is_live": true,
	  "prices_as_of": "2026-04-14T12:00:00Z",
	  "warnings": [
	    {"asset_id":"w1","ticker":"DOGE","error":"provider timeout"}
	  ]
	}`)

	resp, err := ParsePortfolioResponse(raw)
	require.NoError(t, err)
	assert.True(t, resp.IsLive)
	require.NotNil(t, resp.PricesAsOf)
	assert.Equal(t, time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC), *resp.PricesAsOf)
	require.Len(t, resp.Warnings, 1)
	assert.Equal(t, "DOGE", resp.Warnings[0].Ticker)
	assert.Equal(t, "provider timeout", resp.Warnings[0].Error)

	require.Len(t, resp.Positions, 1)
	p := resp.Positions[0]
	require.NotNil(t, p.PriceSource)
	assert.Equal(t, "live", *p.PriceSource)
	require.NotNil(t, p.PriceAsOf)
	assert.Equal(t, time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC), *p.PriceAsOf)
}

func TestParsePortfolioResponse_StandardMode(t *testing.T) {
	raw := []byte(`{
	  "positions":[
	    {"asset_id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD",
	     "quantity":"10","avg_cost":"150","total_cost":"1500",
	     "current_price":"180","current_value":"1800",
	     "unrealized_pnl":"300","realized_pnl":"0"}
	  ]
	}`)

	resp, err := ParsePortfolioResponse(raw)
	require.NoError(t, err)
	assert.False(t, resp.IsLive)
	assert.Nil(t, resp.PricesAsOf)
	assert.Empty(t, resp.Warnings)
	assert.Nil(t, resp.Positions[0].PriceSource)
	assert.Nil(t, resp.Positions[0].PriceAsOf)
}

func TestParsePositions_StillWorks(t *testing.T) {
	raw := []byte(`{"positions":[{"asset_id":"a1","ticker":"X","name":"X","asset_type":"STOCK","currency":"USD","quantity":"1","avg_cost":"1","total_cost":"1","current_value":"1","unrealized_pnl":"0","realized_pnl":"0"}]}`)
	positions, err := ParsePositions(raw)
	require.NoError(t, err)
	require.Len(t, positions, 1)
}
