package snapshots

import (
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

func TestAllI18nKeysResolvedInRenderedTree(t *testing.T) {
	cat := []assetscatalog.Asset{
		{ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD", IsComplex: false},
		{ID: "a2", Ticker: "BTC-F", Name: "Bitcoin Fund", AssetType: "FUND", Currency: "USD", IsComplex: true},
	}

	snap := Snapshot{
		ID:             "s1",
		RecordedAt:     "2024-03-15T10:00:00Z",
		IsFullSnapshot: true,
		Notes:          "test note",
		Entries: []Entry{
			{AssetID: "a1", Quantity: "10", CurrentPrice: "150.00", Source: "MANUAL"},
			{AssetID: "a2", Quantity: "1", CurrentValueOverride: "9500.00", Source: "COINGECKO"},
		},
	}

	res := &ListResult{
		Snapshots: []Snapshot{snap},
		Total:     1,
		Size:      10,
		Offset:    0,
	}

	for _, lang := range []string{"en", "es"} {
		t.Run(lang, func(t *testing.T) {
			// BuildScreen — populated list
			tree := BuildScreen(res, cat, ListParams{}, lang)
			assertNoRawSnapshotsKeys(t, tree, lang, "BuildScreen")

			// BuildScreen — empty state (no filter)
			empty := BuildScreen(&ListResult{Snapshots: nil, Total: 0, Size: 10}, cat, ListParams{}, lang)
			assertNoRawSnapshotsKeys(t, empty, lang, "BuildScreen empty")

			// BuildScreen — empty state with filter active
			isFull := true
			emptyFiltered := BuildScreen(&ListResult{Snapshots: nil, Total: 0, Size: 10}, cat, ListParams{IsFullSnapshot: &isFull}, lang)
			assertNoRawSnapshotsKeys(t, emptyFiltered, lang, "BuildScreen empty+filter")

			// BuildCreateWizard
			cw := BuildCreateWizard(cat, ListParams{}, lang, "", "")
			assertNoRawSnapshotsKeys(t, cw, lang, "BuildCreateWizard")

			// BuildEditWizard
			ew := BuildEditWizard(&snap, cat, ListParams{}, lang, "", "", nil)
			assertNoRawSnapshotsKeys(t, ew, lang, "BuildEditWizard")

			// BuildDeleteModal
			dm := BuildDeleteModal(&snap, ListParams{}, lang)
			assertNoRawSnapshotsKeys(t, dm, lang, "BuildDeleteModal")
		})
	}
}

func assertNoRawSnapshotsKeys(t *testing.T, c components.Component, lang, where string) {
	t.Helper()
	for _, s := range collectSnapshotStrings(c) {
		if strings.HasPrefix(s, "snapshots.") && !strings.Contains(s, " ") {
			t.Errorf("[%s/%s] unresolved key-like string rendered: %q", lang, where, s)
		}
	}
}

// collectSnapshotStrings recursively collects all string values from the
// component tree, including wizard steps' children and their props.
func collectSnapshotStrings(c components.Component) []string {
	var out []string

	for _, v := range c.Props {
		out = append(out, extractStrings(v)...)
	}

	for _, child := range c.Children {
		out = append(out, collectSnapshotStrings(child)...)
	}

	return out
}

// extractStrings handles the known prop value types and recursively descends
// into WizardStep slices so wizard content is fully covered.
func extractStrings(v any) []string {
	var out []string
	switch val := v.(type) {
	case string:
		out = append(out, val)
	case []components.SelectOption:
		for _, opt := range val {
			out = append(out, opt.Label, opt.Value)
		}
	case []components.WizardStep:
		for _, step := range val {
			out = append(out, step.Label)
			for _, child := range step.Children {
				out = append(out, collectSnapshotStrings(child)...)
			}
		}
	case *components.WizardBanner:
		if val != nil {
			out = append(out, val.Message, val.Title)
		}
	}
	return out
}
