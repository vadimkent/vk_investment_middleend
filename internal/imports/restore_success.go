package imports

import (
	"fmt"
	"strconv"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildRestoreCardSuccess renders the success state of restore-card.
// The "Restore another file" button replaces the card with the idle subtree
// embedded in the response (no extra round-trip).
func BuildRestoreCardSuccess(lang string, r *RestoreResult) components.Component {
	headers := []map[string]any{
		{"label": ""},
		{"label": i18n.T(lang, "import.restore.col.imported"), "align": "right"},
		{"label": i18n.T(lang, "import.restore.col.skipped"), "align": "right"},
	}
	type rowSpec struct {
		key   string
		label string
		imp   int
		skip  int
	}
	rows := []rowSpec{
		{"assets", i18n.T(lang, "import.restore.row.assets"), r.AssetsImported, r.AssetsSkipped},
		{"trades", i18n.T(lang, "import.restore.row.trades"), r.TradesImported, r.TradesSkipped},
		{"snapshots", i18n.T(lang, "import.restore.row.snapshots"), r.SnapshotsImported, r.SnapshotsSkipped},
		{"snapshot_entries", i18n.T(lang, "import.restore.row.snapshot_entries"), r.SnapshotEntriesImported, r.SnapshotEntriesSkipped},
	}

	tableRows := make([]components.Component, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, components.Component{
			Type: "table_row", ID: "restore-success-row-" + row.key,
			Props: map[string]any{},
			Children: []components.Component{
				components.Text(fmt.Sprintf("restore-row-%s-label", row.key), row.label, "sm", "normal"),
				components.Text(fmt.Sprintf("restore-row-%s-imp", row.key), strconv.Itoa(row.imp), "sm", "medium"),
				components.Text(fmt.Sprintf("restore-row-%s-skip", row.key), strconv.Itoa(row.skip), "sm", "normal"),
			},
		})
	}

	tryAgainBtn := components.ButtonFull("restore-try-again-btn", i18n.T(lang, "import.restore.try_again"), "", "outline", "outline",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/import/restore_idle",
			TargetID: "restore-card",
			Loading:  "section",
		},
	)

	return components.Component{
		Type: "card", ID: "restore-card",
		Props: map[string]any{},
		Children: []components.Component{
			components.Text("restore-success-title", i18n.T(lang, "import.restore.success_title"), "lg", "bold"),
			{
				Type: "table", ID: "restore-success-table",
				Props: map[string]any{
					"headers": headers,
				},
				Children: tableRows,
			},
			tryAgainBtn,
		},
	}
}
