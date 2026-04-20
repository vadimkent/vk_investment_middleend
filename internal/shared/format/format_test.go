package format

import "testing"

func f64(v float64) *float64 { return &v }

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		name     string
		amount   *float64
		currency string
		lang     string
		want     string
	}{
		{"nil returns dash", nil, "USD", "en", "—"},
		{"usd en", f64(1234.56), "USD", "en", "$1,234.56"},
		{"usd es", f64(1234.56), "USD", "es", "$1.234,56"},
		{"eur en", f64(1234.56), "EUR", "en", "€1,234.56"},
		{"eur es", f64(1234.5), "EUR", "es", "€1.234,50"},
		{"ars en", f64(1000000), "ARS", "en", "$1,000,000.00"},
		{"unknown currency prefixes code en", f64(1234.56), "XYZ", "en", "XYZ 1,234.56"},
		{"unknown currency prefixes code es", f64(10), "XYZ", "es", "XYZ 10,00"},
		{"zero", f64(0), "USD", "en", "$0.00"},
		{"negative en", f64(-1234.56), "USD", "en", "$-1,234.56"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatMoney(tc.amount, tc.currency, tc.lang)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatSignedMoney(t *testing.T) {
	cases := []struct {
		name     string
		amount   *float64
		currency string
		lang     string
		want     string
	}{
		{"nil returns dash", nil, "USD", "en", "—"},
		{"positive usd en", f64(321.67), "USD", "en", "+$321.67"},
		{"negative usd en", f64(-85), "USD", "en", "-$85.00"},
		{"zero usd en (no sign)", f64(0), "USD", "en", "$0.00"},
		{"positive eur es", f64(1234.5), "EUR", "es", "+€1.234,50"},
		{"negative eur es", f64(-1234.5), "EUR", "es", "-€1.234,50"},
		{"zero es", f64(0), "USD", "es", "$0,00"},
		{"positive unknown currency", f64(10), "XYZ", "en", "+XYZ 10.00"},
		{"negative unknown currency", f64(-10), "XYZ", "en", "-XYZ 10.00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSignedMoney(tc.amount, tc.currency, tc.lang)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatQuantity(t *testing.T) {
	cases := []struct {
		name string
		v    *float64
		lang string
		want string
	}{
		{"nil returns dash", nil, "en", "—"},
		{"integer en", f64(10), "en", "10"},
		{"one decimal en", f64(10.5), "en", "10.5"},
		{"trailing zeros stripped en", f64(10.500), "en", "10.5"},
		{"three decimals en", f64(0.125), "en", "0.125"},
		{"decimal es", f64(10.5), "es", "10,5"},
		{"negative en", f64(-3.25), "en", "-3.25"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatQuantity(tc.v, tc.lang)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatSignedPercent(t *testing.T) {
	cases := []struct {
		name string
		v    *float64
		lang string
		want string
	}{
		{"nil returns dash", nil, "en", "—"},
		{"positive en", f64(12.34), "en", "+12.34%"},
		{"negative en (rounds)", f64(-5.678), "en", "-5.68%"},
		{"zero en (no sign)", f64(0), "en", "0.00%"},
		{"positive es", f64(12.34), "es", "+12,34%"},
		{"negative es", f64(-5.5), "es", "-5,50%"},
		{"zero es", f64(0), "es", "0,00%"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSignedPercent(tc.v, tc.lang)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
