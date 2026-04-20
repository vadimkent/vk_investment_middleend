package format

import (
	"strconv"
	"strings"
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

// --- internals ---

func currencyPrefix(code string) string {
	if sym, ok := currencySymbols[code]; ok {
		return sym
	}
	return code + " "
}

func formatDecimal(v float64, decimals int, lang string) string {
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	intPart, frac := s, ""
	if i := strings.Index(s, "."); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
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
