package portfolio

import (
	"fmt"
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

// FormatRelativeTime renders t relative to now, localized.
func FormatRelativeTime(t *time.Time, now time.Time, lang string) string {
	if t == nil {
		return "—"
	}
	d := now.Sub(*t)
	if d < 0 {
		d = -d
	}
	switch {
	case d < 30*time.Second:
		return i18n.T(lang, "time.relative.just_now")
	case d < time.Minute:
		return interp(i18n.T(lang, "time.relative.seconds_ago"), int(d.Seconds()))
	case d < time.Hour:
		return interp(i18n.T(lang, "time.relative.minutes_ago"), int(d.Minutes()))
	case d < 24*time.Hour:
		return interp(i18n.T(lang, "time.relative.hours_ago"), int(d.Hours()))
	default:
		return interp(i18n.T(lang, "time.relative.days_ago"), int(d.Hours()/24))
	}
}

// PnLPct computes unrealized_pnl / total_cost * 100. Returns nil if either
// input is nil, or total_cost is zero.
func PnLPct(unrealized, totalCost *float64) *float64 {
	if unrealized == nil || totalCost == nil || *totalCost == 0 {
		return nil
	}
	v := (*unrealized) / (*totalCost) * 100
	return &v
}

// --- internals ---

func interp(template string, n int) string {
	return strings.Replace(template, "{n}", fmt.Sprintf("%d", n), 1)
}
