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

	body := components.ColumnWithGap("review-modal-body", "lg", children...)
	return components.Component{
		Type: "modal",
		ID:   "import-review-modal",
		Props: map[string]any{
			"visible":      true,
			"dismissible":  false,
			"presentation": "dialog",
		},
		Children: []components.Component{body},
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

// buildSummary renders the AI Summary card with structured stats up top
// (assets / trades / snapshots / warnings counts), the AI's prose summary,
// and the assumptions list inline (always visible — no toggle).
func buildSummary(lang string, sess *Session) components.Component {
	stats := buildStatsRow(lang, sess)

	summaryHeader := components.Text("summary-title", i18n.T(lang, "import.review.summary"), "md", "bold")
	summaryText := components.Text("summary-text", sess.AISummary, "sm", "normal")

	body := []components.Component{summaryHeader, stats, summaryText}

	if len(sess.Assumptions) > 0 {
		body = append(body, buildAssumptions(lang, sess.Assumptions))
	}

	content := components.ColumnWithGap("summary-card-body", "md", body...)
	return components.Card("summary-card", content)
}

// buildStatsRow renders four count "stat" cells (assets / trades / snapshots /
// warnings) as a horizontal row with each cell a small column of label + count.
func buildStatsRow(lang string, sess *Session) components.Component {
	statCell := func(id, labelKey string, count int) components.Component {
		label := components.TextStyled(id+"-label", i18n.T(lang, labelKey), "xs", "normal", "block", "muted", "", "")
		value := components.Text(id+"-value", strconv.Itoa(count), "lg", "bold")
		return components.ColumnWithGap(id, "xs", value, label)
	}

	return components.RowWithGap("review-stats",
		[]string{"1fr", "1fr", "1fr", "1fr"}, "md",
		statCell("stat-assets", "import.review.stats.assets", len(sess.Preview.Assets)),
		statCell("stat-trades", "import.review.stats.trades", len(sess.Preview.Trades)),
		statCell("stat-snapshots", "import.review.stats.snapshots", len(sess.Preview.Snapshots)),
		statCell("stat-warnings", "import.review.stats.warnings", sess.GapCounts.Warnings),
	)
}

// buildAssumptions renders an always-visible bullet list of the AI's stated
// assumptions. No toggle — the dismissed `toggle` primitive is the form switch,
// not a collapsible.
func buildAssumptions(lang string, assumptions []string) components.Component {
	header := components.TextStyled(
		"assumptions-title",
		i18n.T(lang, "import.review.assumptions"),
		"sm", "bold", "block", "muted", "", "",
	)
	bullets := make([]components.Component, 0, len(assumptions))
	for i, a := range assumptions {
		bullets = append(bullets, components.Text(
			fmt.Sprintf("assumption-%d", i),
			"• "+a, "sm", "normal",
		))
	}
	list := components.ColumnWithGap("assumptions-list", "xs", bullets...)
	return components.ColumnWithGap("assumptions-block", "xs", header, list)
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

	saveBtn := components.ButtonFull("issues-save-btn", i18n.T(lang, "import.gaps.save"),
		"", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/resolve_gaps",
			Method:   "POST",
			TargetID: "import-modal-slot",
			Loading:  "section",
		},
	)
	saveBtn.Props["size"] = "sm"
	saveActions := components.RowWithGap("issues-save-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("issues-save-spacer", "none"),
		saveBtn,
	)
	cards = append(cards, saveActions)

	return components.ColumnWithGap("issues-section", "sm", cards...)
}

func hasWarnings(sess *Session) bool {
	for _, g := range sess.Gaps {
		if g.Severity == "warning" {
			return true
		}
	}
	return false
}

// buildWarnings renders an always-visible list of warnings (no toggle).
func buildWarnings(lang string, sess *Session) components.Component {
	header := components.Text("warnings-title", i18n.T(lang, "import.review.warnings"), "md", "bold")
	rows := []components.Component{header}
	for _, g := range sess.Gaps {
		if g.Severity != "warning" {
			continue
		}
		rows = append(rows, components.RowWithGap("warning-"+g.ID,
			[]string{"auto", "1fr"}, "sm",
			components.Component{
				Type: "badge", ID: "warning-" + g.ID + "-badge",
				Props: map[string]any{"label": g.Type, "variant": "secondary"},
			},
			components.Text("warning-"+g.ID+"-desc", g.Description, "sm", "normal"),
		))
	}
	return components.ColumnWithGap("warnings-section", "sm", rows...)
}

// buildPreview renders three always-visible tables (Assets / Trades /
// Snapshots) using the proper Table + TableColumn + TableRow primitives.
func buildPreview(lang string, sess *Session) components.Component {
	header := components.Text("preview-title", i18n.T(lang, "import.review.preview"), "md", "bold")
	return components.ColumnWithGap("preview-section", "md",
		header,
		buildPreviewAssets(lang, sess.Preview.Assets),
		buildPreviewTrades(lang, sess.Preview.Trades),
		buildPreviewSnapshots(lang, sess.Preview.Snapshots),
	)
}

func sectionLabel(lang, key string, count int) string {
	return fmt.Sprintf("%s (%d)", i18n.T(lang, key), count)
}

func buildPreviewAssets(lang string, assets []PreviewAsset) components.Component {
	heading := components.Text("preview-assets-title",
		sectionLabel(lang, "import.review.preview.assets", len(assets)), "sm", "bold")
	cols := []components.TableColumn{
		{ID: "ticker", Header: "Ticker", Width: "100px"},
		{ID: "name", Header: "Name", Width: "1fr"},
		{ID: "type", Header: "Type", Width: "100px"},
		{ID: "currency", Header: "Currency", Width: "100px"},
		{ID: "action", Header: "Action", Width: "100px"},
	}
	rows := make([]components.Component, 0, len(assets))
	for i, a := range assets {
		rows = append(rows, components.TableRow(fmt.Sprintf("preview-asset-%d", i),
			components.Text(fmt.Sprintf("preview-asset-%d-ticker", i), a.Ticker, "sm", "medium"),
			components.Text(fmt.Sprintf("preview-asset-%d-name", i), a.Name, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-asset-%d-type", i), a.AssetType, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-asset-%d-currency", i), a.Currency, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-asset-%d-action", i), a.Action, "sm", "normal"),
		))
	}
	return components.ColumnWithGap("preview-assets-block", "xs", heading,
		components.Table("preview-assets", cols, rows...),
	)
}

func buildPreviewTrades(lang string, trades []PreviewTrade) components.Component {
	heading := components.Text("preview-trades-title",
		sectionLabel(lang, "import.review.preview.trades", len(trades)), "sm", "bold")
	cols := []components.TableColumn{
		{ID: "row", Header: "Row", Width: "60px", Align: "right"},
		{ID: "ticker", Header: "Ticker", Width: "100px"},
		{ID: "type", Header: "Type", Width: "80px"},
		{ID: "date", Header: "Date", Width: "120px"},
		{ID: "qty", Header: "Qty", Width: "80px", Align: "right"},
		{ID: "price", Header: "Price", Width: "100px", Align: "right"},
		{ID: "fees", Header: "Fees", Width: "80px", Align: "right"},
		{ID: "status", Header: "Status", Width: "100px"},
	}
	rows := make([]components.Component, 0, len(trades))
	for i, tr := range trades {
		rows = append(rows, components.TableRow(fmt.Sprintf("preview-trade-%d", i),
			components.Text(fmt.Sprintf("preview-trade-%d-row", i), strconv.Itoa(tr.Row), "sm", "normal"),
			components.Text(fmt.Sprintf("preview-trade-%d-ticker", i), tr.Ticker, "sm", "medium"),
			components.Text(fmt.Sprintf("preview-trade-%d-type", i), tr.TradeType, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-trade-%d-date", i), tr.Date, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-trade-%d-qty", i), derefOrDash(tr.Quantity), "sm", "normal"),
			components.Text(fmt.Sprintf("preview-trade-%d-price", i), derefOrDash(tr.PricePerUnit), "sm", "normal"),
			components.Text(fmt.Sprintf("preview-trade-%d-fees", i), tr.Fees, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-trade-%d-status", i), tr.Status, "sm", "normal"),
		))
	}
	return components.ColumnWithGap("preview-trades-block", "xs", heading,
		components.Table("preview-trades", cols, rows...),
	)
}

func buildPreviewSnapshots(lang string, snapshots []PreviewSnapshot) components.Component {
	heading := components.Text("preview-snapshots-title",
		sectionLabel(lang, "import.review.preview.snapshots", len(snapshots)), "sm", "bold")
	cols := []components.TableColumn{
		{ID: "date", Header: "Date", Width: "1fr"},
		{ID: "entries", Header: "Entries", Width: "100px", Align: "right"},
		{ID: "status", Header: "Status", Width: "120px"},
	}
	rows := make([]components.Component, 0, len(snapshots))
	for i, s := range snapshots {
		rows = append(rows, components.TableRow(fmt.Sprintf("preview-snapshot-%d", i),
			components.Text(fmt.Sprintf("preview-snapshot-%d-date", i), s.RecordedAt, "sm", "normal"),
			components.Text(fmt.Sprintf("preview-snapshot-%d-entries", i), strconv.Itoa(len(s.Entries)), "sm", "normal"),
			components.Text(fmt.Sprintf("preview-snapshot-%d-status", i), s.Status, "sm", "normal"),
		))
	}
	return components.ColumnWithGap("preview-snapshots-block", "xs", heading,
		components.Table("preview-snapshots", cols, rows...),
	)
}

func derefOrDash(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	return *s
}

// buildActionBar renders the modal's bottom row: status badge on the left,
// Cancel + Confirm on the right. Cancel uses Dismiss() — purely client-side,
// the session expires server-side via TTL. Confirm POSTs to the confirm
// endpoint and the response replaces import-root with a fresh tree.
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

	statusBadge := components.Component{
		Type: "badge", ID: "review-status-badge",
		Props: map[string]any{"label": statusLabel, "variant": statusVariant},
	}

	cancelBtn := components.ButtonFull("review-cancel-btn", i18n.T(lang, "import.review.cancel"),
		"", "secondary", "ghost",
		components.Dismiss(),
	)
	cancelBtn.Props["size"] = "sm"

	confirmBtn := components.ButtonFull("review-confirm-btn", i18n.T(lang, "import.review.confirm"),
		"", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/sessions/" + sess.ID + "/confirm",
			Method:   "POST",
			TargetID: "import-root",
			Loading:  "full",
		},
	)
	confirmBtn.Props["size"] = "sm"
	confirmBtn.Props["disabled"] = sess.Status != "ready"

	return components.RowWithGap("review-action-bar",
		[]string{"auto", "1fr", "auto", "auto"}, "sm",
		statusBadge,
		components.Spacer("review-action-spacer", "none"),
		cancelBtn,
		confirmBtn,
	)
}

func joinInts(rows []int) string {
	parts := make([]string, len(rows))
	for i, n := range rows {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}
