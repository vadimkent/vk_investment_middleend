package portfolio

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = i18n.Load(filepath.Join("..", "..", "locales"))
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		delta time.Duration
		lang  string
		want  string
	}{
		{5 * time.Second, "en", "just now"},
		{45 * time.Second, "en", "45 seconds ago"},
		{2 * time.Minute, "en", "2 minutes ago"},
		{3 * time.Hour, "en", "3 hours ago"},
		{2 * 24 * time.Hour, "en", "2 days ago"},
		{2 * 24 * time.Hour, "es", "hace 2 días"},
	}
	for _, tc := range tests {
		got := FormatRelativeTime(ptrTime(now.Add(-tc.delta)), now, tc.lang)
		assert.Equal(t, tc.want, got, "delta=%s lang=%s", tc.delta, tc.lang)
	}
	assert.Equal(t, "—", FormatRelativeTime(nil, now, "en"))
}

func TestPnLPct(t *testing.T) {
	unrealized := 321.67
	totalCost := 1533.33
	got := PnLPct(&unrealized, &totalCost)
	require.NotNil(t, got)
	assert.InDelta(t, 20.978, *got, 0.01)

	assert.Nil(t, PnLPct(nil, &totalCost))
	assert.Nil(t, PnLPct(&unrealized, nil))
	zero := 0.0
	assert.Nil(t, PnLPct(&unrealized, &zero))
}

// helpers

func ptrTime(t time.Time) *time.Time { return &t }
