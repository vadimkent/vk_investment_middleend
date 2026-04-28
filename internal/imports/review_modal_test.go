package imports

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func sampleSessionReady() *Session {
	return &Session{
		ID: "sess-1", Status: "ready",
		AISummary:   "Looks like a Broker X export.",
		Assumptions: []string{"USD amounts"},
		Preview: Preview{
			Assets: []PreviewAsset{{Ticker: "AAPL", Name: "Apple", AssetType: "stock", Currency: "USD", Action: "create"}},
			Trades: []PreviewTrade{{Row: 2, Ticker: "AAPL", TradeType: "buy", Date: "2026-01-15", Fees: "0", Status: "ok"}},
		},
		GapCounts: GapCounts{Blocking: 0, Warnings: 0},
	}
}

func sampleSessionBlocking() *Session {
	return &Session{
		ID: "sess-1", Status: "needs_review",
		AISummary: "x",
		Gaps: []Gap{
			{ID: "g1", Severity: "blocking", Type: "missing_currency",
				Description: "currency not detected", AffectedRows: []int{2, 5},
				Suggestion: "set currency to USD"},
			{ID: "g2", Severity: "warning", Type: "ambiguous_date",
				Description: "date ambiguous", AffectedRows: []int{8}, Suggestion: "use ISO"},
		},
		GapCounts: GapCounts{Blocking: 1, Warnings: 1},
	}
}

func TestBuildReviewModal_ReadyState(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	c := BuildReviewModal("en", sampleSessionReady(), "")
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"import-review-modal"`,
		`"type":"modal"`,
		`Ready to import`,
		`Looks like a Broker X export.`,
		`USD amounts`,
		`AAPL`,
		`/actions/import/sessions/sess-1/confirm`,
		`/actions/import/sessions/sess-1/cancel`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in review modal: %s", want, js)
		}
	}

	if strings.Contains(js, `"id":"issues-section"`) {
		t.Error("expected no issues section when blocking == 0")
	}
}

func TestBuildReviewModal_BlockingState(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	c := BuildReviewModal("en", sampleSessionBlocking(), "")
	b, _ := json.Marshal(c)
	js := string(b)

	for _, want := range []string{
		`"id":"issues-section"`,
		`"name":"resolutions[g1]"`,
		`/actions/import/sessions/sess-1/resolve_gaps`,
		`This file has 1 issue`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in blocking review modal", want)
		}
	}
}

func TestBuildReviewModal_WithErrorBanner(t *testing.T) {
	loadTestLocales(t)
	loadReviewLocales(t)
	c := BuildReviewModal("en", sampleSessionReady(), "Validation failed.")
	b, _ := json.Marshal(c)
	if !strings.Contains(string(b), "Validation failed.") {
		t.Fatal("missing error banner message")
	}
}

func loadReviewLocales(t *testing.T) {
	// Extends the locales loaded by loadTestLocales with review-specific keys.
	// The test helper calls i18n.Load again with a richer snapshot.
	t.Helper()
	dir := t.TempDir()
	en := `{
		"import": {
			"review": {
				"blocking_banner": "This file has {n} issue(s) that need your input before importing.",
				"ready_banner": "Ready to import — review the preview and confirm.",
				"summary": "AI Summary",
				"assumptions": "Assumptions ({n})",
				"issues": "Issues",
				"warnings": "{n} warning(s)",
				"preview": "Preview",
				"preview.assets": "Assets ({n})",
				"preview.trades": "Trades ({n})",
				"preview.snapshots": "Snapshots ({n})",
				"confirm": "Confirm import",
				"cancel": "Cancel",
				"status": { "needs_review": "Needs review", "ready": "Ready" }
			},
			"gaps": { "affected_rows": "Affected rows: {rows}", "input_placeholder": "Enter value…", "save": "Save resolutions" }
		}
	}`
	_ = writeFile(dir+"/en.json", en)
	_ = i18n.Load(dir)
}
