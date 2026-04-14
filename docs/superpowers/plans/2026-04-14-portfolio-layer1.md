# Portfolio Layer 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `spec/screens/portfolio/01-positions.md` — public `GET /screens/portfolio` protected endpoint that returns an SDUI tree with a Total-Value summary and an 11-column positions table.

**Architecture:** New `internal/portfolio/` package with focused files (types, client, sort, format, builder, use case, handler). The handler calls the use case; the use case calls the BE client, sorts, formats, and hands clean data to the builder. The builder composes the SDUI tree. Login success navigate target flips from `/screens/home` to `/screens/portfolio` at the end.

**Tech Stack:** Go, Gin, testify, existing `internal/components`, `internal/i18n`, `internal/auth`.

---

## File Structure

**Create:**

| File | Responsibility |
|---|---|
| `internal/portfolio/types.go` | `Position` domain struct + parse from BE JSON |
| `internal/portfolio/client.go` | HTTP client for `GET /v1/portfolio`, forwards `Authorization` |
| `internal/portfolio/client_test.go` | client behavior (forward header, parse, error map) |
| `internal/portfolio/sort.go` | `SortPositions([]Position)` per spec order |
| `internal/portfolio/sort_test.go` | stable sort semantics |
| `internal/portfolio/format.go` | money, signed money, quantity, percent, relative time — per locale |
| `internal/portfolio/format_test.go` | formatting edge cases per locale |
| `internal/portfolio/builder.go` | `BuildScreen(positions, lang) components.Component` + `BuildEmpty(lang)` |
| `internal/portfolio/builder_test.go` | tree shape + semantic props |
| `internal/portfolio/get_usecase.go` | orchestrates client → sort → build |
| `internal/portfolio/get_usecase_test.go` | orchestration via a fake client |
| `internal/portfolio/handler.go` | Gin handler; reads headers; maps errors to HTTP |
| `internal/portfolio/handler_test.go` | handler HTTP behavior |

**Modify:**

- `locales/en.json`, `locales/es.json` — add `portfolio.*` and `time.relative.*` keys.
- `internal/server/server.go` — register protected `GET /screens/portfolio`.
- `internal/auth/login_handler.go` — flip success navigate target.
- `internal/auth/login_handler_test.go` — update the asserted `target_id`.

---

### Task 1: Domain type and i18n keys

**Files:**
- Create: `internal/portfolio/types.go`
- Create: `internal/portfolio/types_test.go`
- Modify: `locales/en.json`, `locales/es.json`

- [ ] **Step 1: Write the failing test**

Create `internal/portfolio/types_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/vadimkent/repos/vk_investment_middleend_v2 && go test ./internal/portfolio/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement types**

Create `internal/portfolio/types.go`:

```go
package portfolio

import (
	"encoding/json"
	"strconv"
	"time"
)

// Position is the middleend domain representation of a portfolio position,
// parsed from the backend response. Nullable numeric fields use pointers so
// the tree builder can distinguish "missing" from zero.
type Position struct {
	AssetID        string
	Ticker         string
	Name           string
	AssetType      string
	Currency       string
	Quantity       *float64
	AvgCost        *float64
	TotalCost      *float64
	CurrentPrice   *float64
	CurrentValue   *float64
	UnrealizedPnL  *float64
	RealizedPnL    float64
	LastSnapshotAt *time.Time
}

type rawPosition struct {
	AssetID        string  `json:"asset_id"`
	Ticker         string  `json:"ticker"`
	Name           string  `json:"name"`
	AssetType      string  `json:"asset_type"`
	Currency       string  `json:"currency"`
	Quantity       *string `json:"quantity"`
	AvgCost        *string `json:"avg_cost"`
	TotalCost      *string `json:"total_cost"`
	CurrentPrice   *string `json:"current_price"`
	CurrentValue   *string `json:"current_value"`
	UnrealizedPnL  *string `json:"unrealized_pnl"`
	RealizedPnL    *string `json:"realized_pnl"`
	LastSnapshotAt *string `json:"last_snapshot_at"`
}

type rawResponse struct {
	Positions []rawPosition `json:"positions"`
}

// ParsePositions parses the backend /v1/portfolio body into []Position.
func ParsePositions(body []byte) ([]Position, error) {
	var r rawResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	out := make([]Position, 0, len(r.Positions))
	for _, rp := range r.Positions {
		p := Position{
			AssetID:   rp.AssetID,
			Ticker:    rp.Ticker,
			Name:      rp.Name,
			AssetType: rp.AssetType,
			Currency:  rp.Currency,
		}
		p.Quantity = parseFloatPtr(rp.Quantity)
		p.AvgCost = parseFloatPtr(rp.AvgCost)
		p.TotalCost = parseFloatPtr(rp.TotalCost)
		p.CurrentPrice = parseFloatPtr(rp.CurrentPrice)
		p.CurrentValue = parseFloatPtr(rp.CurrentValue)
		p.UnrealizedPnL = parseFloatPtr(rp.UnrealizedPnL)
		if v := parseFloatPtr(rp.RealizedPnL); v != nil {
			p.RealizedPnL = *v
		}
		if rp.LastSnapshotAt != nil {
			if t, err := time.Parse(time.RFC3339, *rp.LastSnapshotAt); err == nil {
				p.LastSnapshotAt = &t
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func parseFloatPtr(s *string) *float64 {
	if s == nil {
		return nil
	}
	v, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		return nil
	}
	return &v
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Update `locales/en.json`**

Replace its contents with:

```json
{
  "app": {
    "name": "VK Investment Tracker"
  },
  "nav": {
    "portfolio": "Portfolio",
    "assets": "Assets",
    "trades": "Trades",
    "snapshots": "Snapshots",
    "import": "Import",
    "analysis": "Analysis",
    "logout": "Log out"
  },
  "auth": {
    "login_title": "Log in",
    "email_label": "Email",
    "email_placeholder": "you@example.com",
    "password_label": "Password",
    "password_placeholder": "Your password",
    "submit": "Log in",
    "no_account_prompt": "Don't have an account?",
    "register_link": "Sign up"
  },
  "portfolio": {
    "title": "Portfolio",
    "total_value": "Total Value",
    "empty_title": "No positions yet",
    "empty_subtitle": "Register your first trade or snapshot.",
    "col": {
      "ticker": "Ticker",
      "name": "Name",
      "type": "Type",
      "quantity": "Quantity",
      "avg_cost": "Avg Cost",
      "total_cost": "Total Cost",
      "market_value": "Market Value",
      "unrealized_pnl": "Unrealized P&L",
      "pnl_pct": "% P&L",
      "realized_pnl": "Realized P&L",
      "last_snapshot": "Last Snapshot"
    }
  },
  "time": {
    "relative": {
      "just_now": "just now",
      "seconds_ago": "{n} seconds ago",
      "minutes_ago": "{n} minutes ago",
      "hours_ago": "{n} hours ago",
      "days_ago": "{n} days ago"
    }
  },
  "home": {
    "welcome_title": "Welcome to VK Investment Tracker",
    "subtitle": "This is a scaffolded middleend serving SDUI components."
  }
}
```

- [ ] **Step 6: Update `locales/es.json`**

Replace its contents with:

```json
{
  "app": {
    "name": "VK Investment Tracker"
  },
  "nav": {
    "portfolio": "Portafolio",
    "assets": "Activos",
    "trades": "Operaciones",
    "snapshots": "Snapshots",
    "import": "Importar",
    "analysis": "Análisis",
    "logout": "Cerrar sesión"
  },
  "auth": {
    "login_title": "Iniciar sesión",
    "email_label": "Correo",
    "email_placeholder": "vos@ejemplo.com",
    "password_label": "Contraseña",
    "password_placeholder": "Tu contraseña",
    "submit": "Ingresar",
    "no_account_prompt": "¿No tenés cuenta?",
    "register_link": "Registrarme"
  },
  "portfolio": {
    "title": "Portafolio",
    "total_value": "Valor total",
    "empty_title": "Aún no hay posiciones",
    "empty_subtitle": "Registrá tu primer trade o snapshot.",
    "col": {
      "ticker": "Ticker",
      "name": "Nombre",
      "type": "Tipo",
      "quantity": "Cantidad",
      "avg_cost": "Costo prom.",
      "total_cost": "Costo total",
      "market_value": "Valor de mercado",
      "unrealized_pnl": "G/P no realizada",
      "pnl_pct": "% G/P",
      "realized_pnl": "G/P realizada",
      "last_snapshot": "Último snapshot"
    }
  },
  "time": {
    "relative": {
      "just_now": "hace instantes",
      "seconds_ago": "hace {n} segundos",
      "minutes_ago": "hace {n} minutos",
      "hours_ago": "hace {n} horas",
      "days_ago": "hace {n} días"
    }
  },
  "home": {
    "welcome_title": "Bienvenido a VK Investment Tracker",
    "subtitle": "Este es un middleend scaffoldeado que sirve componentes SDUI."
  }
}
```

- [ ] **Step 7: Run full suite to confirm nothing broke**

Run: `go test ./... -count=1`
Expected: all existing tests pass; portfolio types tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/portfolio/types.go internal/portfolio/types_test.go locales/en.json locales/es.json
git commit -m "feat(portfolio): Position domain type + i18n keys"
```

---

### Task 2: Formatters

**Files:**
- Create: `internal/portfolio/format.go`
- Create: `internal/portfolio/format_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/format_test.go`:

```go
package portfolio

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
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

func ptr(v float64) *float64   { return &v }
func ptrTime(t time.Time) *time.Time { return &t }
```

Note: tests reference `require` — add import.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -v`
Expected: FAIL — format helpers not defined.

- [ ] **Step 3: Implement formatters**

Create `internal/portfolio/format.go`:

```go
package portfolio

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

var currencySymbols = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"JPY": "¥",
	"ARS": "$",
	"BRL": "R$",
}

// FormatMoney formats amount with the currency's symbol (or code + space for
// unknown currencies) using locale-specific thousand/decimal separators.
func FormatMoney(amount *float64, currency, lang string) string {
	if amount == nil {
		return "—"
	}
	return currencyPrefix(currency) + formatDecimal(*amount, 2, lang)
}

// FormatSignedMoney is like FormatMoney but prefixes a "+" for positive
// (non-zero) values. Zero has no sign.
func FormatSignedMoney(amount *float64, currency, lang string) string {
	if amount == nil {
		return "—"
	}
	prefix := currencyPrefix(currency)
	v := *amount
	if v > 0 {
		return "+" + prefix + formatDecimal(v, 2, lang)
	}
	if v < 0 {
		return "-" + prefix + formatDecimal(-v, 2, lang)
	}
	return prefix + formatDecimal(0, 2, lang)
}

// FormatQuantity formats a quantity stripping trailing zeros; at most 8 decimals.
func FormatQuantity(q *float64, lang string) string {
	if q == nil {
		return "—"
	}
	s := strconv.FormatFloat(*q, 'f', -1, 64)
	// Convert '.' to locale's decimal separator.
	if lang == "es" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

// FormatSignedPercent formats a percentage value (already in percent units,
// e.g. 12.34 → "+12.34%").
func FormatSignedPercent(pct *float64, lang string) string {
	if pct == nil {
		return "—"
	}
	v := *pct
	body := formatDecimal(absFloat(v), 2, lang) + "%"
	switch {
	case v > 0:
		return "+" + body
	case v < 0:
		return "-" + body
	default:
		return formatDecimal(0, 2, lang) + "%"
	}
}

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

func currencyPrefix(code string) string {
	if sym, ok := currencySymbols[code]; ok {
		return sym
	}
	return code + " "
}

func formatDecimal(v float64, decimals int, lang string) string {
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	// Split integer and fractional parts.
	intPart, frac := s, ""
	if i := strings.Index(s, "."); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	// Thousand separators.
	intPart = withThousands(intPart, lang)
	if frac == "" {
		return intPart
	}
	decSep := "."
	if lang == "es" {
		decSep = ","
	}
	return intPart + decSep + frac
}

func withThousands(intPart, lang string) string {
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}
	sep := ","
	if lang == "es" {
		sep = "."
	}
	n := len(intPart)
	if n <= 3 {
		if negative {
			return "-" + intPart
		}
		return intPart
	}
	var b strings.Builder
	rem := n % 3
	if rem > 0 {
		b.WriteString(intPart[:rem])
	}
	for i := rem; i < n; i += 3 {
		if b.Len() > 0 {
			b.WriteString(sep)
		}
		b.WriteString(intPart[i : i+3])
	}
	if negative {
		return "-" + b.String()
	}
	return b.String()
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func interp(template string, n int) string {
	return strings.Replace(template, "{n}", fmt.Sprintf("%d", n), 1)
}
```

- [ ] **Step 4: Add the missing `require` import to the test file**

Open `internal/portfolio/format_test.go`. At the top, replace the imports block with:

```go
import (
	"path/filepath"
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (all format tests + prior types tests).

- [ ] **Step 6: Commit**

```bash
git add internal/portfolio/format.go internal/portfolio/format_test.go
git commit -m "feat(portfolio): locale-aware money/percent/quantity/time formatters"
```

---

### Task 3: Sort

**Files:**
- Create: `internal/portfolio/sort.go`
- Create: `internal/portfolio/sort_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/sort_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestSort -v`
Expected: FAIL — `SortPositions` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/sort.go`:

```go
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
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -run TestSort -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/sort.go internal/portfolio/sort_test.go
git commit -m "feat(portfolio): SortPositions by value desc then ticker asc"
```

---

### Task 4: Backend client

**Files:**
- Create: `internal/portfolio/client.go`
- Create: `internal/portfolio/client_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/client_test.go`:

```go
package portfolio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetPositions_ForwardsAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/portfolio", r.URL.Path)
		assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"positions":[{"asset_id":"a1","ticker":"AAPL","name":"Apple","asset_type":"STOCK","currency":"USD","quantity":"1","avg_cost":"100","total_cost":"100","current_price":"110","current_value":"110","unrealized_pnl":"10","realized_pnl":"0","last_snapshot_at":"2024-06-01T10:00:00Z"}]}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	positions, err := c.GetPositions(context.Background(), "Bearer abc")
	require.NoError(t, err)
	require.Len(t, positions, 1)
	assert.Equal(t, "AAPL", positions[0].Ticker)
}

func TestClient_GetPositions_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer bad")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestClient_GetPositions_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer x")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}

func TestClient_GetPositions_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.GetPositions(context.Background(), "Bearer x")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestClient -v`
Expected: FAIL — `NewClient` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/client.go`:

```go
package portfolio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrUnauthorized = errors.New("backend unauthorized")
	ErrBackend      = errors.New("backend error")
)

// Client talks to the backend /v1/portfolio endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: timeout}}
}

// GetPositions calls GET /v1/portfolio with the caller's Authorization header
// forwarded verbatim. Returns ErrUnauthorized on 401, ErrBackend on 5xx or
// malformed response.
func (c *Client) GetPositions(ctx context.Context, authorization string) ([]Position, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/portfolio", nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBackend, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrBackend, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		positions, err := ParsePositions(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %v", ErrBackend, err)
		}
		return positions, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrBackend, resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/client.go internal/portfolio/client_test.go
git commit -m "feat(portfolio): HTTP client for GET /v1/portfolio"
```

---

### Task 5: Builder (tree + empty state)

**Files:**
- Create: `internal/portfolio/builder.go`
- Create: `internal/portfolio/builder_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/builder_test.go`:

```go
package portfolio

import (
	"testing"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEmpty_HasEmptyBlock(t *testing.T) {
	s := BuildEmpty("en")
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "portfolio", s.ID)

	empty := findDescendantByID(s, "portfolio-empty")
	require.NotNil(t, empty)
	title := findDescendantByID(*empty, "empty-title")
	require.NotNil(t, title)
	assert.Equal(t, "No positions yet", title.Props["content"])

	subtitle := findDescendantByID(*empty, "empty-subtitle")
	require.NotNil(t, subtitle)
	assert.Equal(t, "muted", subtitle.Props["color"])
}

func TestBuildEmpty_NoTable(t *testing.T) {
	s := BuildEmpty("en")
	assert.Nil(t, findDescendantByID(s, "positions-table"))
	assert.Nil(t, findDescendantByID(s, "positions-header"))
	assert.Nil(t, findDescendantByID(s, "positions-body"))
}

func TestBuildScreen_RootShape(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	ps := samplePositions()

	s := BuildScreen(ps, "en", now)
	assert.Equal(t, "screen", s.Type)
	assert.Equal(t, "portfolio", s.ID)
	assert.Equal(t, "Portfolio", s.Props["title"])

	summary := findDescendantByID(s, "portfolio-summary")
	require.NotNil(t, summary)
	table := findDescendantByID(s, "positions-table")
	require.NotNil(t, table)
}

func TestBuildScreen_TotalValueSingleCurrency(t *testing.T) {
	now := time.Now()
	v1, v2 := 1000.0, 500.0
	ps := []Position{
		{Ticker: "A", Currency: "USD", CurrentValue: &v1},
		{Ticker: "B", Currency: "USD", CurrentValue: &v2},
	}
	s := BuildScreen(ps, "en", now)

	totals := findDescendantByID(s, "total-values")
	require.NotNil(t, totals)
	require.Len(t, totals.Children, 1)
	assert.Equal(t, "$1,500.00", totals.Children[0].Props["content"])
}

func TestBuildScreen_TotalValueMultiCurrency(t *testing.T) {
	now := time.Now()
	u, e := 1000.0, 800.0
	ps := []Position{
		{Ticker: "A", Currency: "USD", CurrentValue: &u},
		{Ticker: "B", Currency: "EUR", CurrentValue: &e},
	}
	s := BuildScreen(ps, "en", now)

	totals := findDescendantByID(s, "total-values")
	require.NotNil(t, totals)
	require.Len(t, totals.Children, 2)
	assert.Equal(t, "$1,000.00", totals.Children[0].Props["content"])
	assert.Equal(t, "€800.00", totals.Children[1].Props["content"])
}

func TestBuildScreen_TotalValueAllNull(t *testing.T) {
	now := time.Now()
	ps := []Position{{Ticker: "A", Currency: "USD"}}
	s := BuildScreen(ps, "en", now)

	totals := findDescendantByID(s, "total-values")
	require.NotNil(t, totals)
	require.Len(t, totals.Children, 1)
	assert.Equal(t, "—", totals.Children[0].Props["content"])
}

func TestBuildScreen_HeaderHas11Columns(t *testing.T) {
	s := BuildScreen(samplePositions(), "en", time.Now())
	header := findDescendantByID(s, "positions-header")
	require.NotNil(t, header)
	widths, ok := header.Props["widths"].([]string)
	require.True(t, ok)
	assert.Len(t, widths, 11)
	assert.Len(t, header.Children, 11)
	labels := []string{"Ticker", "Name", "Type", "Quantity", "Avg Cost", "Total Cost", "Market Value", "Unrealized P&L", "% P&L", "Realized P&L", "Last Snapshot"}
	for i, want := range labels {
		assert.Equal(t, want, header.Children[i].Props["content"], "col %d", i)
	}
}

func TestBuildScreen_BodyUsesListWithOneItemPerPosition(t *testing.T) {
	ps := samplePositions()
	s := BuildScreen(ps, "en", time.Now())
	body := findDescendantByID(s, "positions-body")
	require.NotNil(t, body)
	assert.Equal(t, "list", body.Type)
	assert.Len(t, body.Children, len(ps))
	for _, child := range body.Children {
		assert.Equal(t, "list_item", child.Type)
	}
}

func TestBuildScreen_PositionRowValuesInOrder(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	qty, avg, total, cur, pnl, realized := 10.0, 153.33, 1533.33, 1855.0, 321.67, 175.0
	snap := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	ps := []Position{{
		AssetID: "a1", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", Currency: "USD",
		Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur,
		UnrealizedPnL: &pnl, RealizedPnL: realized, LastSnapshotAt: &snap,
	}}

	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-a1")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)
	require.Len(t, row.Children, 11)

	want := []string{"AAPL", "Apple Inc", "STOCK", "10", "$153.33", "$1,533.33", "$1,855.00", "+$321.67", "+20.98%", "+$175.00", "2 days ago"}
	for i, w := range want {
		assert.Equal(t, w, row.Children[i].Props["content"], "col %d", i)
	}
}

func TestBuildScreen_PositivePnLHasPositiveColor(t *testing.T) {
	now := time.Now()
	tc, cur, pnl := 1000.0, 1200.0, 200.0
	ps := []Position{{
		AssetID: "x1", Ticker: "X", Currency: "USD",
		TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 50.0,
	}}
	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-x1")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)

	assert.Equal(t, "positive", row.Children[7].Props["color"])  // unrealized
	assert.Equal(t, "positive", row.Children[8].Props["color"])  // %
	assert.Equal(t, "positive", row.Children[9].Props["color"])  // realized
}

func TestBuildScreen_NegativePnLHasNegativeColor(t *testing.T) {
	now := time.Now()
	tc, cur, pnl := 1000.0, 900.0, -100.0
	ps := []Position{{
		AssetID: "x2", Ticker: "X", Currency: "USD",
		TotalCost: &tc, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: -25.0,
	}}
	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-x2")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)

	assert.Equal(t, "negative", row.Children[7].Props["color"])
	assert.Equal(t, "negative", row.Children[8].Props["color"])
	assert.Equal(t, "negative", row.Children[9].Props["color"])
}

func TestBuildScreen_ZeroOrNullPnLHasNoColor(t *testing.T) {
	now := time.Now()
	zero := 0.0
	ps := []Position{{
		AssetID: "x3", Ticker: "X", Currency: "USD",
		UnrealizedPnL: &zero, RealizedPnL: 0.0,
	}}
	s := BuildScreen(ps, "en", now)
	item := findDescendantByID(s, "position-x3")
	require.NotNil(t, item)
	row := findDescendantByType(*item, "row")
	require.NotNil(t, row)

	_, hasColor := row.Children[7].Props["color"]
	assert.False(t, hasColor)
	_, hasColor = row.Children[9].Props["color"]
	assert.False(t, hasColor)
}

// helpers

func samplePositions() []Position {
	qty, avg, total, cur, pnl := 10.0, 100.0, 1000.0, 1200.0, 200.0
	return []Position{
		{AssetID: "s1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD",
			Quantity: &qty, AvgCost: &avg, TotalCost: &total, CurrentValue: &cur, UnrealizedPnL: &pnl, RealizedPnL: 0},
	}
}

func findDescendantByType(c components.Component, typ string) *components.Component {
	if c.Type == typ {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByType(c.Children[i], typ); found != nil {
			return found
		}
	}
	return nil
}

func findDescendantByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByID(c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestBuild -v`
Expected: FAIL — `BuildScreen` / `BuildEmpty` undefined.

- [ ] **Step 3: Implement the builder**

Create `internal/portfolio/builder.go`:

```go
package portfolio

import (
	"sort"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

var columnWidths = []string{
	"80px",  // ticker
	"1fr",   // name
	"80px",  // type
	"80px",  // quantity
	"110px", // avg cost
	"110px", // total cost
	"120px", // market value
	"130px", // unrealized pnl
	"80px",  // % pnl
	"120px", // realized pnl
	"120px", // last snapshot
}

var columnKeys = []string{
	"portfolio.col.ticker",
	"portfolio.col.name",
	"portfolio.col.type",
	"portfolio.col.quantity",
	"portfolio.col.avg_cost",
	"portfolio.col.total_cost",
	"portfolio.col.market_value",
	"portfolio.col.unrealized_pnl",
	"portfolio.col.pnl_pct",
	"portfolio.col.realized_pnl",
	"portfolio.col.last_snapshot",
}

// BuildScreen builds the portfolio tree for the given positions.
// now is used to format relative times.
func BuildScreen(positions []Position, lang string, now time.Time) components.Component {
	if len(positions) == 0 {
		return BuildEmpty(lang)
	}

	summary := buildSummary(positions, lang)
	table := buildTable(positions, lang, now)

	root := components.ColumnWithGap("portfolio-root", "lg", summary, table)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

// BuildEmpty builds the screen for an empty portfolio.
func BuildEmpty(lang string) components.Component {
	title := components.Text("empty-title", i18n.T(lang, "portfolio.empty_title"), "lg", "bold")
	subtitle := components.TextStyled("empty-subtitle", i18n.T(lang, "portfolio.empty_subtitle"), "md", "normal", "", "muted", "", "")
	empty := components.ColumnWithGap("portfolio-empty", "sm", title, subtitle)
	root := components.ColumnWithGap("portfolio-root", "lg", empty)
	return components.Screen("portfolio", i18n.T(lang, "portfolio.title"), root)
}

func buildSummary(ps []Position, lang string) components.Component {
	label := components.TextStyled("summary-label", i18n.T(lang, "portfolio.total_value"), "sm", "normal", "", "muted", "", "")

	totals := components.Column("total-values")
	byCurrency := totalsByCurrency(ps)
	if len(byCurrency) == 0 {
		totals.Children = append(totals.Children, components.Text("total-value-empty", "—", "xl", "bold"))
	} else {
		codes := make([]string, 0, len(byCurrency))
		for c := range byCurrency {
			codes = append(codes, c)
		}
		sort.Slice(codes, func(i, j int) bool {
			return byCurrency[codes[i]] > byCurrency[codes[j]]
		})
		for _, c := range codes {
			v := byCurrency[c]
			totals.Children = append(totals.Children,
				components.Text("total-value-"+c, FormatMoney(&v, c, lang), "xl", "bold"))
		}
	}

	return components.ColumnWithGap("portfolio-summary", "sm", label, totals)
}

func totalsByCurrency(ps []Position) map[string]float64 {
	out := map[string]float64{}
	for _, p := range ps {
		if p.CurrentValue == nil {
			continue
		}
		out[p.Currency] += *p.CurrentValue
	}
	return out
}

func buildTable(ps []Position, lang string, now time.Time) components.Component {
	headerCells := make([]components.Component, 0, 11)
	for i, key := range columnKeys {
		cell := components.Text("col-"+columnShortID(i), i18n.T(lang, key), "sm", "bold")
		headerCells = append(headerCells, cell)
	}
	header := components.Row("positions-header", columnWidths, headerCells...)

	listChildren := make([]components.Component, 0, len(ps))
	for _, p := range ps {
		listChildren = append(listChildren, buildPositionItem(p, lang, now))
	}
	body := components.List("positions-body", listChildren...)

	return components.ColumnWithGap("positions-table", "sm", header, body)
}

func columnShortID(i int) string {
	names := []string{"ticker", "name", "type", "quantity", "avg-cost", "total-cost", "market-value", "unrealized-pnl", "pnl-pct", "realized-pnl", "last-snapshot"}
	return names[i]
}

func buildPositionItem(p Position, lang string, now time.Time) components.Component {
	realized := p.RealizedPnL
	pct := PnLPct(p.UnrealizedPnL, p.TotalCost)

	cells := []components.Component{
		components.Text("cell-ticker", p.Ticker, "sm", "bold"),
		components.Text("cell-name", p.Name, "sm", "normal"),
		components.Text("cell-type", p.AssetType, "sm", "normal"),
		components.Text("cell-quantity", FormatQuantity(p.Quantity, lang), "sm", "normal"),
		components.Text("cell-avg-cost", FormatMoney(p.AvgCost, p.Currency, lang), "sm", "normal"),
		components.Text("cell-total-cost", FormatMoney(p.TotalCost, p.Currency, lang), "sm", "normal"),
		components.Text("cell-market-value", FormatMoney(p.CurrentValue, p.Currency, lang), "sm", "normal"),
		colored("cell-unrealized-pnl", FormatSignedMoney(p.UnrealizedPnL, p.Currency, lang), pnlColor(p.UnrealizedPnL)),
		colored("cell-pnl-pct", FormatSignedPercent(pct, lang), pnlColor(pct)),
		colored("cell-realized-pnl", FormatSignedMoney(&realized, p.Currency, lang), pnlColor(&realized)),
		components.Text("cell-last-snapshot", FormatRelativeTime(p.LastSnapshotAt, now, lang), "sm", "normal"),
	}
	row := components.Row("position-"+p.AssetID+"-row", columnWidths, cells...)
	return components.ListItem("position-"+p.AssetID, row)
}

// pnlColor returns "positive", "negative" or "" (no color) based on v.
func pnlColor(v *float64) string {
	if v == nil || *v == 0 {
		return ""
	}
	if *v > 0 {
		return "positive"
	}
	return "negative"
}

func colored(id, content, color string) components.Component {
	if color == "" {
		return components.Text(id, content, "sm", "normal")
	}
	return components.TextStyled(id, content, "sm", "normal", "", color, "", "")
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS (all builder tests + types + format + sort + client).

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/builder.go internal/portfolio/builder_test.go
git commit -m "feat(portfolio): SDUI tree builder + empty state"
```

---

### Task 6: Use case

**Files:**
- Create: `internal/portfolio/get_usecase.go`
- Create: `internal/portfolio/get_usecase_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/portfolio/get_usecase_test.go`:

```go
package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	positions []Position
	err       error
	gotAuth   string
}

func (f *fakeClient) GetPositions(ctx context.Context, auth string) ([]Position, error) {
	f.gotAuth = auth
	return f.positions, f.err
}

func TestGetUseCase_ReturnsBuiltScreen(t *testing.T) {
	v := 100.0
	client := &fakeClient{positions: []Position{
		{AssetID: "a1", Ticker: "AAPL", Name: "Apple", Currency: "USD", CurrentValue: &v},
	}}
	uc := NewGetUseCase(client)
	now := time.Now()
	screen, err := uc.Execute(context.Background(), "Bearer tok", "en", now)
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok", client.gotAuth)
	assert.Equal(t, "screen", screen.Type)
	assert.Equal(t, "portfolio", screen.ID)
}

func TestGetUseCase_SortsBeforeBuilding(t *testing.T) {
	v1, v2 := 100.0, 500.0
	client := &fakeClient{positions: []Position{
		{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v1},
		{AssetID: "b1", Ticker: "B", Currency: "USD", CurrentValue: &v2},
	}}
	uc := NewGetUseCase(client)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	body := findDescendantByID(screen, "positions-body")
	require.NotNil(t, body)
	require.Len(t, body.Children, 2)
	assert.Equal(t, "position-b1", body.Children[0].ID)
	assert.Equal(t, "position-a1", body.Children[1].ID)
}

func TestGetUseCase_EmptyPositions(t *testing.T) {
	client := &fakeClient{positions: []Position{}}
	uc := NewGetUseCase(client)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.NotNil(t, findDescendantByID(screen, "portfolio-empty"))
}

func TestGetUseCase_PropagatesErrors(t *testing.T) {
	client := &fakeClient{err: ErrUnauthorized}
	uc := NewGetUseCase(client)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestGetUseCase -v`
Expected: FAIL — `NewGetUseCase` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/get_usecase.go`:

```go
package portfolio

import (
	"context"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
)

// positionsFetcher is the interface the use case depends on; *Client satisfies it.
type positionsFetcher interface {
	GetPositions(ctx context.Context, authorization string) ([]Position, error)
}

type GetUseCase struct {
	client positionsFetcher
}

func NewGetUseCase(client positionsFetcher) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches positions from the backend, sorts them, and builds the
// portfolio SDUI tree. `now` is used for relative-time formatting.
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string, now time.Time) (components.Component, error) {
	positions, err := uc.client.GetPositions(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	SortPositions(positions)
	return BuildScreen(positions, lang, now), nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/get_usecase.go internal/portfolio/get_usecase_test.go
git commit -m "feat(portfolio): use case orchestrating fetch + sort + build"
```

---

### Task 7: HTTP handler

**Files:**
- Create: `internal/portfolio/handler.go`
- Create: `internal/portfolio/handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/portfolio/handler_test.go`:

```go
package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct {
	positions []Position
	err       error
	gotAuth   string
}

func (s *stubFetcher) GetPositions(ctx context.Context, auth string) ([]Position, error) {
	s.gotAuth = auth
	return s.positions, s.err
}

func setupHandlerRouter(f positionsFetcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(NewGetUseCase(f))
	r.GET("/screens/portfolio", h.Get)
	return r
}

func TestHandler_ForwardsAuthorizationAndReturnsScreen(t *testing.T) {
	v := 100.0
	f := &stubFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Name: "Apple", Currency: "USD", CurrentValue: &v}}}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Bearer abc", f.gotAuth)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "screen", body["type"])
	assert.Equal(t, "portfolio", body["id"])
}

func TestHandler_BackendUnauthorizedReturns401WithRedirect(t *testing.T) {
	f := &stubFetcher{err: ErrUnauthorized}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"unauthorized"`)
	assert.Contains(t, w.Body.String(), `"redirect":"/screens/login"`)
}

func TestHandler_BackendErrorReturns502(t *testing.T) {
	f := &stubFetcher{err: ErrBackend}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "BACKEND_ERROR")
}

func TestHandler_UsesAcceptLanguage(t *testing.T) {
	v := 100.0
	f := &stubFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Name: "Apple", Currency: "USD", CurrentValue: &v}}}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	req.Header.Set("Accept-Language", "es")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Portafolio")
}

func TestHandler_NowIsSet(t *testing.T) {
	// Uses LastSnapshotAt one day before "now" — formatted "1 day ago" in en.
	snap := time.Now().Add(-24 * time.Hour)
	v := 100.0
	f := &stubFetcher{positions: []Position{{AssetID: "a1", Ticker: "AAPL", Currency: "USD", CurrentValue: &v, LastSnapshotAt: &snap}}}
	r := setupHandlerRouter(f)

	req := httptest.NewRequest("GET", "/screens/portfolio", nil)
	req.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "1 days ago")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/portfolio/... -run TestHandler -v`
Expected: FAIL — `NewHandler` undefined.

- [ ] **Step 3: Implement**

Create `internal/portfolio/handler.go`:

```go
package portfolio

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

type Handler struct {
	uc *GetUseCase
}

func NewHandler(uc *GetUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Get(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	screen, err := h.uc.Execute(c.Request.Context(), auth, lang, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthorized):
			shared.RespondUnauthorized(c, "/screens/login")
		default:
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		}
		return
	}
	c.JSON(http.StatusOK, screen)
}

func parseLang(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return "en"
	}
	parts := strings.SplitN(header, ",", 2)
	lang := strings.SplitN(parts[0], "-", 2)[0]
	lang = strings.SplitN(lang, ";", 2)[0]
	return strings.TrimSpace(lang)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/portfolio/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/portfolio/handler.go internal/portfolio/handler_test.go
git commit -m "feat(portfolio): GET /screens/portfolio handler"
```

---

### Task 8: Wire route and flip login navigate

**Files:**
- Modify: `internal/server/server.go`
- Modify: `internal/auth/login_handler.go`
- Modify: `internal/auth/login_handler_test.go`

- [ ] **Step 1: Update the login success target in the existing handler test**

In `internal/auth/login_handler_test.go`, find the line:

```go
	assert.Equal(t, "/screens/home", resp["target_id"])
```

Replace it with:

```go
	assert.Equal(t, "/screens/portfolio", resp["target_id"])
```

- [ ] **Step 2: Run the handler tests to confirm the expected failure**

Run: `go test ./internal/auth/... -run TestLoginHandler_Success -v`
Expected: FAIL — handler still returns `/screens/home`.

- [ ] **Step 3: Flip the login handler target**

In `internal/auth/login_handler.go`, find the line:

```go
	c.JSON(http.StatusOK, components.NavigateResponse("/screens/home", &fb).WithAuth(res.Token, res.ExpiresAt))
```

Replace it with:

```go
	c.JSON(http.StatusOK, components.NavigateResponse("/screens/portfolio", &fb).WithAuth(res.Token, res.ExpiresAt))
```

- [ ] **Step 4: Wire the protected route in the server**

In `internal/server/server.go`, add the `portfolio` import alphabetically in the project-imports block:

```go
	"github.com/project/vk-investment-middleend/internal/auth"
	"github.com/project/vk-investment-middleend/internal/config"
	"github.com/project/vk-investment-middleend/internal/home"
	"github.com/project/vk-investment-middleend/internal/login"
	"github.com/project/vk-investment-middleend/internal/portfolio"
	"github.com/project/vk-investment-middleend/internal/shell"
```

Then inside `setupRoutes()`, **after** the `protected.GET("/screens/home", homeHandler.Get)` line, append:

```go
	portfolioClient := portfolio.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	portfolioHandler := portfolio.NewHandler(portfolio.NewGetUseCase(portfolioClient))
	protected.GET("/screens/portfolio", portfolioHandler.Get)
```

- [ ] **Step 5: Run the full test suite**

Run: `go test ./... -count=1`
Expected: all tests pass.

- [ ] **Step 6: Build and lint**

Run: `./cli build 2>&1 | tail -1 && ./cli lint 2>&1 | tail -1`
Expected: both `"status":"success"`.

- [ ] **Step 7: Smoke-test end-to-end**

Run:

```bash
lsof -ti:8082 | xargs kill -9 2>/dev/null; sleep 1
./cli run >/tmp/srv.log 2>&1 &
sleep 2

# no token → 401 with redirect
echo "--- /screens/portfolio no token ---"
curl -s -w "\n%{http_code}\n" http://localhost:8082/screens/portfolio

# real login against the local BE
RESP=$(curl -s -X POST http://localhost:8082/actions/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"demo"}')
TOKEN=$(echo "$RESP" | python3 -c "import json,sys;print(json.load(sys.stdin)['auth']['token'])")

echo "--- /screens/portfolio with token ---"
curl -s -o /dev/null -w "%{http_code}\n" -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio

echo "--- first 300 chars of portfolio body ---"
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8082/screens/portfolio | head -c 300

echo
echo "--- login success navigate target ---"
echo "$RESP" | python3 -c "import json,sys;d=json.load(sys.stdin);print(d['target_id'])"

lsof -ti:8082 | xargs kill -9 2>/dev/null; true
```

Expected:
- `/screens/portfolio` without token → `{"error":"unauthorized","redirect":"/screens/login"}` and HTTP 401.
- `/screens/portfolio` with token → 200.
- Body begins with `{"type":"screen","id":"portfolio",...`.
- Login success target is `/screens/portfolio`.

- [ ] **Step 8: Commit**

```bash
git add internal/server/server.go internal/auth/login_handler.go internal/auth/login_handler_test.go
git commit -m "feat(server): wire protected GET /screens/portfolio; login navigates here"
```

---

## Self-Review Results

**Spec coverage check:**

| Spec requirement | Task |
|---|---|
| `GET /screens/portfolio` protected, 401+redirect on missing/invalid token | Task 8 wiring; Task 7 test `TestHandler_BackendUnauthorizedReturns401WithRedirect` covers BE 401 relay |
| Forward `Authorization` verbatim to BE | Task 4 `TestClient_GetPositions_ForwardsAuthorization`; Task 6 `TestGetUseCase_*` passes header through |
| `type: screen`, `id: portfolio`, title via i18n | Task 5 `TestBuildScreen_RootShape`, `TestBuildEmpty_HasEmptyBlock` |
| `portfolio-summary` with per-currency Total Value, fallback to "—" | Task 5 `TestBuildScreen_TotalValue*` |
| 11-column header with exact labels per locale | Task 5 `TestBuildScreen_HeaderHas11Columns` |
| `list` + `list_item` for body with 11 cells per row | Task 5 `TestBuildScreen_BodyUsesListWithOneItemPerPosition`, `TestBuildScreen_PositionRowValuesInOrder` |
| Sort by `current_value DESC` nulls last, ticker ASC | Task 3 all tests; Task 6 `TestGetUseCase_SortsBeforeBuilding` |
| Formatting per locale (money, signed money, quantity, pct, relative time) | Task 2 all tests |
| Color `positive`/`negative`/none per value sign | Task 5 `TestBuildScreen_{Positive,Negative,ZeroOrNull}PnL*` |
| Empty state: title + subtitle, no CTA | Task 5 `TestBuildEmpty_*` |
| BE 5xx → 502 `BACKEND_ERROR` | Task 7 `TestHandler_BackendErrorReturns502` |
| BE 401 → 401 + redirect | Task 7 `TestHandler_BackendUnauthorizedReturns401WithRedirect` |
| i18n keys added to both locales | Task 1 Step 5 + Step 6 |
| Login success navigates to `/screens/portfolio` | Task 8 Steps 1–3 |

**Placeholder scan:** none.

**Type consistency:**
- `Position` struct fields used identically across types / sort / format / builder / use case / handler.
- `GetPositions(ctx, authorization string) ([]Position, error)` signature matches on `*Client` (Task 4) and the `positionsFetcher` interface in the use case (Task 6) and the `stubFetcher` in the handler test (Task 7).
- `BuildScreen(positions, lang, now)` signature consistent in Task 5 builder, Task 5 tests, Task 6 use case.
- Color values `"positive"` / `"negative"` match the spec update committed in `de803b9`.
- Column widths and keys defined once in `builder.go` and referenced by header and each row.
