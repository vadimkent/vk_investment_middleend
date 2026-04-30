package imports

import (
	"fmt"
	"strconv"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildRestoreCardSuccess renders the success state of restore-card.
// The "Restore another file" button reloads the card's idle subtree.
func BuildRestoreCardSuccess(lang string, r *RestoreResult) components.Component {
	cols := []components.TableColumn{
		{ID: "label", Header: "", Width: "1fr"},
		{ID: "imported", Header: i18n.T(lang, "import.restore.col.imported"), Width: "120px", Align: "right"},
		{ID: "skipped", Header: i18n.T(lang, "import.restore.col.skipped"), Width: "120px", Align: "right"},
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
		tableRows = append(tableRows, components.TableRow(
			"restore-success-row-"+row.key,
			components.Text(fmt.Sprintf("restore-row-%s-label", row.key), row.label, "sm", "normal"),
			components.Text(fmt.Sprintf("restore-row-%s-imp", row.key), strconv.Itoa(row.imp), "sm", "medium"),
			components.Text(fmt.Sprintf("restore-row-%s-skip", row.key), strconv.Itoa(row.skip), "sm", "normal"),
		))
	}

	tryAgainBtn := components.ButtonFull("restore-try-again-btn", i18n.T(lang, "import.restore.try_again"),
		"", "secondary", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/import/restore_idle",
			TargetID: "restore-card",
			Loading:  "section",
		},
	)
	tryAgainBtn.Props["size"] = "sm"
	actions := components.RowWithGap("restore-success-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("restore-success-actions-spacer", "none"),
		tryAgainBtn,
	)

	body := components.ColumnWithGap("restore-success-body", "lg",
		components.Text("restore-success-title", i18n.T(lang, "import.restore.success_title"), "lg", "bold"),
		components.Table("restore-success-table", cols, tableRows...),
		actions,
	)
	return components.Card("restore-card", body)
}
