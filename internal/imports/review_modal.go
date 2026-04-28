package imports

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildReviewModal renders the modal subtree injected into import-modal-slot
// after a successful analyze or after a resolve_gaps round-trip. errorMessage
// is optional — when non-empty an inline error banner is rendered at the top.
func BuildReviewModal(lang string, sess *Session, errorMessage string) components.Component {
	children := make([]components.Component, 0, 8)

	if errorMessage != "" {
		children = append(children, components.Component{
			Type: "banner", ID: "review-error",
			Props: map[string]any{"variant": "error", "message": errorMessage},
		})
	}

	children = append(children, buildBanner(lang, sess))
	children = append(children, buildSummary(lang, sess))
	if sess.GapCounts.Blocking > 0 {
		children = append(children, buildIssuesSection(lang, sess))
	}
	if hasWarnings(sess) {
		children = append(children, buildWarnings(lang, sess))
	}
	children = append(children, buildPreview(lang, sess))
	children = append(children, buildActionBar(lang, sess))

	return components.Component{
		Type: "modal",
		ID:   "import-review-modal",
		Props: map[string]any{
			"visible":      true,
			"dismissible":  false,
			"presentation": "dialog",
		},
		Children: children,
	}
}

func buildBanner(lang string, sess *Session) components.Component {
	if sess.GapCounts.Blocking > 0 {
		msg := strings.ReplaceAll(i18n.T(lang, "import.review.blocking_banner"),
			"{n}", strconv.Itoa(sess.GapCounts.Blocking))
		return components.Component{
			Type: "banner", ID: "review-banner",
			Props: map[string]any{"variant": "warning", "message": msg, "dismissible": false},
		}
	}
	return components.Component{
		Type: "banner", ID: "review-banner",
		Props: map[string]any{
			"variant":     "info",
			"message":     i18n.T(lang, "import.review.ready_banner"),
			"dismissible": false,
		},
	}
}

func buildSummary(lang string, sess *Session) components.Component {
	children := []components.Component{
		components.Text("summary-title", i18n.T(lang, "import.review.summary"), "md", "bold"),
		components.Text("summary-text", sess.AISummary, "sm", "normal"),
	}
	if len(sess.Assumptions) > 0 {
		title := strings.ReplaceAll(i18n.T(lang, "import.review.assumptions"),
			"{n}", strconv.Itoa(len(sess.Assumptions)))
		bullets := make([]components.Component, 0, len(sess.Assumptions))
		for i, a := range sess.Assumptions {
			bullets = append(bullets, components.Text(fmt.Sprintf("assumption-%d", i), "• "+a, "sm", "normal"))
		}
		children = append(children, components.Component{
			Type: "toggle", ID: "assumptions-toggle",
			Props:    map[string]any{"label": title, "default_open": false},
			Children: bullets,
		})
	}
	return components.Component{
		Type: "card", ID: "summary-card",
		Props:    map[string]any{},
		Children: children,
	}
}

func buildIssuesSection(lang string, sess *Session) components.Component {
	cards := []components.Component{
		components.Text("issues-title", i18n.T(lang, "import.review.issues"), "md", "bold"),
	}
	for _, g := range sess.Gaps {
		if g.Severity != "blocking" {
			continue
		}
		rowsStr := strings.ReplaceAll(i18n.T(lang, "import.gaps.affected_rows"),
			"{rows}", joinInts(g.AffectedRows))
		preset := ""
		if g.Resolution != nil {
			preset = *g.Resolution
		}
		cards = append(cards, components.Component{
			Type: "card", ID: "gap-" + g.ID,
			Props: map[string]any{"variant": "destructive_outline"},
			Children: []components.Component{
				{
					Type: "badge", ID: "gap-" + g.ID + "-type-badge",
					Props: map[string]any{"label": g.Type, "variant": "destructive"},
				},
				components.Text("gap-"+g.ID+"-desc", g.Description, "sm", "normal"),
				components.Text("gap-"+g.ID+"-rows", rowsStr, "xs", "normal"),
				components.Text("gap-"+g.ID+"-suggestion", g.Suggestion, "xs", "italic"),
				{
					Type: "input", ID: "gap-" + g.ID + "-input",
					Props: map[string]any{
						"name":        "resolutions[" + g.ID + "]",
						"input_type":  "text",
						"placeholder": i18n.T(lang, "import.gaps.input_placeholder"),
						"value":       preset,
					},
				},
			},
		})
	}

	saveBtn := components.Button("issues-save-btn", i18n.T(lang, "import.gaps.save"),
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/resolve_gaps",
			Method:   "POST",
			TargetID: "import-modal-slot",
			Loading:  "section",
		},
	)
	cards = append(cards, saveBtn)

	return components.Component{
		Type:     "column",
		ID:       "issues-section",
		Props:    map[string]any{"gap": "sm"},
		Children: cards,
	}
}

func hasWarnings(sess *Session) bool {
	for _, g := range sess.Gaps {
		if g.Severity == "warning" {
			return true
		}
	}
	return false
}

func buildWarnings(lang string, sess *Session) components.Component {
	count := 0
	for _, g := range sess.Gaps {
		if g.Severity == "warning" {
			count++
		}
	}
	label := strings.ReplaceAll(i18n.T(lang, "import.review.warnings"), "{n}", strconv.Itoa(count))
	rows := []components.Component{}
	for _, g := range sess.Gaps {
		if g.Severity != "warning" {
			continue
		}
		rows = append(rows, components.Component{
			Type: "row", ID: "warning-" + g.ID,
			Props: map[string]any{"gap": "sm", "align_items": "start"},
			Children: []components.Component{
				{
					Type: "badge", ID: "warning-" + g.ID + "-badge",
					Props: map[string]any{"label": g.Type, "variant": "secondary"},
				},
				components.Text("warning-"+g.ID+"-desc", g.Description, "sm", "normal"),
			},
		})
	}
	return components.Component{
		Type: "toggle", ID: "warnings-toggle",
		Props:    map[string]any{"label": label, "default_open": false},
		Children: rows,
	}
}

func buildPreview(lang string, sess *Session) components.Component {
	return components.Component{
		Type: "column",
		ID:   "preview-section",
		Props: map[string]any{"gap": "md"},
		Children: []components.Component{
			components.Text("preview-title", i18n.T(lang, "import.review.preview"), "md", "bold"),
			buildPreviewAssets(lang, sess.Preview.Assets),
			buildPreviewTrades(lang, sess.Preview.Trades),
			buildPreviewSnapshots(lang, sess.Preview.Snapshots),
		},
	}
}

func buildPreviewAssets(lang string, assets []PreviewAsset) components.Component {
	label := strings.ReplaceAll(i18n.T(lang, "import.review.preview.assets"),
		"{n}", strconv.Itoa(len(assets)))
	headers := []string{"Ticker", "Name", "Type", "Currency", "Action"}
	rows := make([][]string, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, []string{a.Ticker, a.Name, a.AssetType, a.Currency, a.Action})
	}
	return wrapTable("preview-assets", label, headers, rows)
}

func buildPreviewTrades(lang string, trades []PreviewTrade) components.Component {
	label := strings.ReplaceAll(i18n.T(lang, "import.review.preview.trades"),
		"{n}", strconv.Itoa(len(trades)))
	headers := []string{"Row", "Ticker", "Type", "Date", "Qty", "Price", "Fees", "Status"}
	rows := make([][]string, 0, len(trades))
	for _, tr := range trades {
		rows = append(rows, []string{
			strconv.Itoa(tr.Row), tr.Ticker, tr.TradeType, tr.Date,
			derefOrDash(tr.Quantity), derefOrDash(tr.PricePerUnit), tr.Fees, tr.Status,
		})
	}
	return wrapTable("preview-trades", label, headers, rows)
}

func buildPreviewSnapshots(lang string, snapshots []PreviewSnapshot) components.Component {
	label := strings.ReplaceAll(i18n.T(lang, "import.review.preview.snapshots"),
		"{n}", strconv.Itoa(len(snapshots)))
	headers := []string{"Date", "Entries", "Status"}
	rows := make([][]string, 0, len(snapshots))
	for _, s := range snapshots {
		rows = append(rows, []string{s.RecordedAt, strconv.Itoa(len(s.Entries)), s.Status})
	}
	return wrapTable("preview-snapshots", label, headers, rows)
}

func derefOrDash(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	return *s
}

func wrapTable(id, label string, headers []string, rows [][]string) components.Component {
	tableHeaders := make([]map[string]any, len(headers))
	for i, h := range headers {
		tableHeaders[i] = map[string]any{"label": h}
	}
	tableRows := make([]components.Component, 0, len(rows))
	for i, r := range rows {
		cells := make([]components.Component, 0, len(r))
		for j, v := range r {
			cells = append(cells, components.Text(fmt.Sprintf("%s-r%d-c%d", id, i, j), v, "sm", "normal"))
		}
		tableRows = append(tableRows, components.Component{
			Type: "table_row", ID: fmt.Sprintf("%s-row-%d", id, i),
			Props:    map[string]any{},
			Children: cells,
		})
	}
	return components.Component{
		Type: "toggle", ID: id + "-toggle",
		Props: map[string]any{"label": label, "default_open": true},
		Children: []components.Component{
			{
				Type: "table", ID: id,
				Props: map[string]any{
					"headers": tableHeaders,
				},
				Children: tableRows,
			},
		},
	}
}

func buildActionBar(lang string, sess *Session) components.Component {
	statusKey := "import.review.status." + sess.Status
	statusLabel := i18n.T(lang, statusKey)
	if statusLabel == statusKey {
		statusLabel = sess.Status
	}
	statusVariant := "secondary"
	if sess.Status == "ready" {
		statusVariant = "success"
	} else if sess.Status == "needs_review" {
		statusVariant = "warning"
	}

	cancelBtn := components.ButtonFull("review-cancel-btn", i18n.T(lang, "import.review.cancel"), "", "ghost", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/cancel",
			Method:   "POST",
			TargetID: "import-root",
			Loading:  "full",
		},
	)
	confirmBtn := components.Button("review-confirm-btn", i18n.T(lang, "import.review.confirm"),
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/confirm",
			Method:   "POST",
			TargetID: "import-root",
			Loading:  "full",
		},
	)
	confirmBtn.Props["disabled"] = sess.Status != "ready"

	return components.Component{
		Type: "row",
		ID:   "review-action-bar",
		Props: map[string]any{
			"sticky":      "bottom",
			"justify":     "space_between",
			"align_items": "center",
			"gap":         "md",
		},
		Children: []components.Component{
			{
				Type: "badge", ID: "review-status-badge",
				Props: map[string]any{"label": statusLabel, "variant": statusVariant},
			},
			{
				Type:  "row",
				ID:    "review-action-buttons",
				Props: map[string]any{"gap": "sm"},
				Children: []components.Component{cancelBtn, confirmBtn},
			},
		},
	}
}

func joinInts(rows []int) string {
	parts := make([]string, len(rows))
	for i, n := range rows {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}
