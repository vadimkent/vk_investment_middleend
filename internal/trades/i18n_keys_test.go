package trades

import (
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

func TestAllI18nKeysResolvedInRenderedScreen(t *testing.T) {
	cat := []assetscatalog.Asset{
		{ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD", IsComplex: false},
	}
	trade := Trade{
		ID: "t1", AssetID: "a1", TradeType: "BUY", Quantity: "10", PricePerUnit: "100",
		Fees: "0", Date: "2024-03-15T10:00:00Z", Source: "MANUAL", Notes: "n",
	}
	res := &ListResult{Trades: []Trade{trade}, Total: 1, Size: 10, Offset: 0}

	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			tree := BuildScreen(res, cat, ListParams{}, lang)
			assertNoRawTradesKeys(t, tree, lang, "BuildScreen")

			empty := BuildScreen(&ListResult{Trades: nil, Total: 0, Size: 10}, cat, ListParams{AssetID: "a1"}, lang)
			assertNoRawTradesKeys(t, empty, lang, "BuildScreen empty+filter")

			cm := BuildCreateModal(cat, ListParams{}, lang, "")
			assertNoRawTradesKeys(t, cm, lang, "BuildCreateModal")

			em := BuildEditModal(trade, cat, ListParams{}, lang, "")
			assertNoRawTradesKeys(t, em, lang, "BuildEditModal")

			dm := BuildDeleteModal(trade, cat, ListParams{}, lang, "")
			assertNoRawTradesKeys(t, dm, lang, "BuildDeleteModal")
		})
	}
}

func assertNoRawTradesKeys(t *testing.T, c components.Component, lang, where string) {
	t.Helper()
	for _, s := range collectStrings(c) {
		if strings.HasPrefix(s, "trades.") && !strings.Contains(s, " ") {
			t.Errorf("[%s/%s] unresolved key-like string rendered: %q", lang, where, s)
		}
	}
}

func collectStrings(c components.Component) []string {
	var out []string
	for _, v := range c.Props {
		switch val := v.(type) {
		case string:
			out = append(out, val)
		case []components.SelectOption:
			for _, opt := range val {
				out = append(out, opt.Label, opt.Value)
			}
		}
	}
	for _, child := range c.Children {
		out = append(out, collectStrings(child)...)
	}
	return out
}
