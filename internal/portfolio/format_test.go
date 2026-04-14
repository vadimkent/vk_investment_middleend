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

func TestFormatMoney_EnUSD(t *testing.T) {
	v := 1234.56
	assert.Equal(t, "$1,234.56", FormatMoney(&v, "USD", "en"))
}

func TestFormatMoney_EsUSD(t *testing.T) {
	v := 1234.56
	assert.Equal(t, "$1.234,56", FormatMoney(&v, "USD", "es"))
}

func TestFormatMoney_EUR(t *testing.T) {
	v := 1234.56
	assert.Equal(t, "€1,234.56", FormatMoney(&v, "EUR", "en"))
}

func TestFormatMoney_UnknownCurrencyUsesCode(t *testing.T) {
	v := 1234.56
	assert.Equal(t, "XYZ 1,234.56", FormatMoney(&v, "XYZ", "en"))
}

func TestFormatMoney_Nil(t *testing.T) {
	assert.Equal(t, "—", FormatMoney(nil, "USD", "en"))
}

func TestFormatSignedMoney(t *testing.T) {
	plus := 321.67
	minus := -85.0
	zero := 0.0
	assert.Equal(t, "+$321.67", FormatSignedMoney(&plus, "USD", "en"))
	assert.Equal(t, "-$85.00", FormatSignedMoney(&minus, "USD", "en"))
	assert.Equal(t, "$0.00", FormatSignedMoney(&zero, "USD", "en"))
	assert.Equal(t, "—", FormatSignedMoney(nil, "USD", "en"))
}

func TestFormatQuantity(t *testing.T) {
	tests := []struct {
		v    *float64
		lang string
		want string
	}{
		{ptr(10.0), "en", "10"},
		{ptr(10.5), "en", "10.5"},
		{ptr(10.500), "en", "10.5"},
		{ptr(0.125), "en", "0.125"},
		{ptr(10.5), "es", "10,5"},
		{nil, "en", "—"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, FormatQuantity(tc.v, tc.lang), "v=%v lang=%s", tc.v, tc.lang)
	}
}

func TestFormatSignedPercent(t *testing.T) {
	plus := 12.34
	minus := -5.678
	zero := 0.0
	assert.Equal(t, "+12.34%", FormatSignedPercent(&plus, "en"))
	assert.Equal(t, "-5.68%", FormatSignedPercent(&minus, "en"))
	assert.Equal(t, "0.00%", FormatSignedPercent(&zero, "en"))
	assert.Equal(t, "+12,34%", FormatSignedPercent(&plus, "es"))
	assert.Equal(t, "—", FormatSignedPercent(nil, "en"))
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

func ptr(v float64) *float64        { return &v }
func ptrTime(t time.Time) *time.Time { return &t }
