package imports

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

// loadTestLocales loads a minimal in-memory locale set so i18n.T returns
// real strings during tests. Tests run in any order, so this is idempotent.
func loadTestLocales(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	en := `{
		"import": {
			"title": "Import & Export",
			"ai": { "title": "Import historical data", "description": "Upload a file…" },
			"upload": { "label": "File", "placeholder": "Drop a file here or click to browse",
				"hint_ai": "CSV, TSV, XLS, XLSX, TXT — max 5 MB",
				"hint_restore": "CSV — max 10 MB",
				"error_size": "File exceeds the {limit} limit.",
				"error_format": "Unsupported file format.",
				"reattach_hint": "Re-select the file to retry" },
			"hint": { "label": "Hint (optional)", "placeholder": "e.g. broker x export" },
			"analyze": "Analyze file",
			"loading": { "analyze": { "1": "Detecting columns…", "2": "Mapping tickers…",
				"3": "Resolving currencies…", "4": "Building preview…", "5": "Validating consistency…" } },
			"export": { "title": "Export data", "description": "Download all data.", "submit": "Export all data" },
			"restore": { "title": "Restore from backup", "description": "Upload a CSV backup.",
				"submit": "Restore", "success_title": "Restored successfully",
				"col": { "imported": "Imported", "skipped": "Skipped" },
				"row": { "assets": "Assets", "trades": "Trades", "snapshots": "Snapshots", "snapshot_entries": "Snapshot entries" },
				"try_again": "Restore another file", "error_generic": "Restore failed." }
		}
	}`
	if err := writeFile(dir+"/en.json", en); err != nil {
		t.Fatal(err)
	}
	if err := i18n.Load(dir); err != nil {
		t.Fatal(err)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestBuildRoot_Shape(t *testing.T) {
	loadTestLocales(t)
	root := BuildRoot("en")
	b, _ := json.MarshalIndent(root, "", "  ")
	js := string(b)

	for _, want := range []string{
		`"id": "import-root"`,
		`"id": "import-section"`,
		`"id": "ai-import-card"`,
		`"id": "export-restore-stack"`,
		`"id": "export-card"`,
		`"id": "restore-card"`,
		`"id": "import-modal-slot"`,
		`Import`,
		`Export`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in root tree", want)
		}
	}
}

func TestBuildAIImportCardIdle_DefaultState(t *testing.T) {
	loadTestLocales(t)
	c := BuildAIImportCardIdle("en", "", "", "")
	b, _ := json.Marshal(c)
	js := string(b)
	if !strings.Contains(js, `"id":"ai-import-card"`) {
		t.Fatal("missing ai-import-card id")
	}
	if !strings.Contains(js, `"type":"file_upload"`) {
		t.Fatal("missing file_upload child")
	}
	if !strings.Contains(js, `"max_size_bytes":5242880`) {
		t.Fatal("missing 5MB cap on upload")
	}
	if strings.Contains(js, `"prefill_filename"`) {
		t.Fatal("expected no prefill_filename in default state")
	}
}

func TestBuildAIImportCardIdle_WithErrorAndPrefill(t *testing.T) {
	loadTestLocales(t)
	c := BuildAIImportCardIdle("en", "AI parse failed.", "broker.csv", "amounts in USD")
	b, _ := json.Marshal(c)
	js := string(b)
	if !strings.Contains(js, `"prefill_filename":"broker.csv"`) {
		t.Fatal("missing prefill_filename")
	}
	if !strings.Contains(js, `AI parse failed.`) {
		t.Fatal("missing error banner message")
	}
	if !strings.Contains(js, `amounts in USD`) {
		t.Fatal("missing prefilled hint")
	}
}

func TestBuildRestoreCardIdle_DefaultState(t *testing.T) {
	loadTestLocales(t)
	c := BuildRestoreCardIdle("en", "", "")
	b, _ := json.Marshal(c)
	js := string(b)
	for _, want := range []string{
		`"id":"restore-card"`,
		`"type":"file_upload"`,
		`"max_size_bytes":10485760`,
		`"accept":".csv"`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("missing %q in restore card", want)
		}
	}
}
