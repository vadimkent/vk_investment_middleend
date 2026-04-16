package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEvolution_Basic(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"},
	    {"snapshot_id":"s2","recorded_at":"2026-04-13T10:00:00Z","is_full_snapshot":true,"total_value":"1200.00","currency":"USD"}
	  ],
	  "total": 2
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 2)
	assert.Equal(t, "s1", points[0].SnapshotID)
	assert.Equal(t, time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC), points[0].RecordedAt)
	assert.True(t, points[0].IsFullSnapshot)
	assert.InDelta(t, 1000.0, points[0].TotalValue, 1e-9)
	assert.Equal(t, "USD", points[0].Currency)
	assert.InDelta(t, 1200.0, points[1].TotalValue, 1e-9)
}

func TestParseEvolution_Empty(t *testing.T) {
	raw := []byte(`{"evolution":[],"total":0}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	assert.Empty(t, points)
}

func TestParseEvolution_MultiCurrency(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"1000.00","currency":"USD"},
	    {"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"800.00","currency":"EUR"}
	  ]
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 2)
	assert.Equal(t, "USD", points[0].Currency)
	assert.Equal(t, "EUR", points[1].Currency)
}

func TestParseEvolution_InvalidJSON(t *testing.T) {
	_, err := ParseEvolution([]byte(`not json`))
	require.Error(t, err)
}

func TestParseEvolution_ParsesAssets(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {
	      "snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,
	      "total_value":"15420.50","total_cost":"12000.00","currency":"USD",
	      "assets":[
	        {"asset_id":"u1","ticker":"AAPL","value":"5000.00"},
	        {"asset_id":"u2","ticker":"GOOG","value":"10420.50"}
	      ]
	    }
	  ]
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Len(t, points[0].Assets, 2)
	assert.Equal(t, "u1", points[0].Assets[0].AssetID)
	assert.Equal(t, "AAPL", points[0].Assets[0].Ticker)
	assert.InDelta(t, 5000.0, points[0].Assets[0].Value, 1e-9)
	assert.Equal(t, "GOOG", points[0].Assets[1].Ticker)
	assert.InDelta(t, 10420.50, points[0].Assets[1].Value, 1e-9)
}

func TestParseEvolution_AssetsAbsentIsEmpty(t *testing.T) {
	raw := []byte(`{"evolution":[{"snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,"total_value":"100","currency":"USD"}]}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 1)
	assert.Empty(t, points[0].Assets)
}

func TestParseEvolution_AssetWithMalformedValueSkipped(t *testing.T) {
	raw := []byte(`{
	  "evolution":[
	    {
	      "snapshot_id":"s1","recorded_at":"2026-04-10T10:00:00Z","is_full_snapshot":true,
	      "total_value":"100","currency":"USD",
	      "assets":[
	        {"asset_id":"u1","ticker":"AAPL","value":"abc"},
	        {"asset_id":"u2","ticker":"GOOG","value":"200"}
	      ]
	    }
	  ]
	}`)
	points, err := ParseEvolution(raw)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Len(t, points[0].Assets, 1)
	assert.Equal(t, "GOOG", points[0].Assets[0].Ticker)
}
