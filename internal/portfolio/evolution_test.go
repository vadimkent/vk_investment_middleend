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
