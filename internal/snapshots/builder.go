package snapshots

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
	"github.com/project/vk-investment-middleend/internal/shared/format"
)

// Exported identifiers used by screen/list/action handlers.
const (
	ScreenID    = "snapshots-screen"
	SectionID   = "snapshots-section"
	ModalSlotID = "snapshots-modal-slot"
)

// BuildScreen returns the full SDUI tree for GET /screens/snapshots.
func BuildScreen(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
	return BuildScreenWithModal(res, catalog, p, lang, components.Component{})
}

// BuildScreenWithModal renders the full screen tree with a specific modal
// subtree injected into the modal slot (ModalSlotID). Used by the auto-snapshot
// flow, which replaces the entire screen root with a refreshed list AND an
// open edit wizard in one ActionResponse.
func BuildScreenWithModal(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string, modal components.Component) components.Component {
	header := buildHeader(lang)
	section := BuildSnapshotsSection(res, catalog, p, lang)
	var modalSlot components.Component
	if modal.Type == "" {
		modalSlot = components.Column(ModalSlotID)
	} else {
		modalSlot = components.Column(ModalSlotID, modal)
	}
	root := components.ColumnWithGap("snapshots-root", "lg", header, section, modalSlot)
	return components.Screen(ScreenID, i18n.T(lang, "snapshots.title"), root)
}

// BuildSnapshotsSection returns the replaceable subtree: filter, then table (or
// empty state), then pagination (when total > size).
func BuildSnapshotsSection(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
	children := []components.Component{buildFilter(p, lang)}

	if len(res.Snapshots) == 0 {
		children = append(children, buildEmpty(p, lang))
	} else {
		byID := indexCatalog(catalog)
		children = append(children, buildTable(res.Snapshots, byID, p, lang))
		if res.Total > res.Size {
			children = append(children, buildPagination(res, p, lang))
		}
	}

	return components.ColumnWithGap(SectionID, "sm", children...)
}

// --- header ---

func buildHeader(lang string) components.Component {
	title := components.Text("snapshots-title", i18n.T(lang, "snapshots.title"), "lg", "bold")
	spacer := components.Column("snapshots-header-spacer")
	return components.Row("snapshots-header-row", []string{"auto", "1fr"}, title, spacer)
}

// --- filter row (select + Auto/New Snapshot buttons) ---

func buildFilter(p ListParams, lang string) components.Component {
	opts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "snapshots.filter.type_any")},
		{Value: "true", Label: i18n.T(lang, "snapshots.filter.type_full")},
		{Value: "false", Label: i18n.T(lang, "snapshots.filter.type_partial")},
	}

	defaultValue := boolPtrToString(p.IsFullSnapshot)

	// On-change resets offset to 0 and binds is_full_snapshot to the select's {value}.
	filterEndpoint := "/actions/snapshots/list?is_full_snapshot={value}&offset=0"

	sel := components.Component{
		Type: "select",
		ID:   "snapshots-filter-type",
		Props: map[string]any{
			"name":          "is_full_snapshot",
			"default_value": defaultValue,
			"options":       opts,
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "reload",
				Endpoint: filterEndpoint,
				TargetID: SectionID,
				Loading:  "section",
			},
		},
	}

	filler := components.Spacer("snapshots-filter-spacer", "none")

	autoEndpoint := buildListURL("/actions/snapshots/auto", p.IsFullSnapshot, p.Offset)
	autoBtn := components.ButtonFull("snapshots-auto-btn", i18n.T(lang, "snapshots.auto_btn"), "", "secondary", "solid",
		components.Submit(autoEndpoint, "POST", ScreenID),
	)
	autoBtn.Props["size"] = "sm"
	autoBtn.Props["align_self"] = "right"

	newEndpoint := buildListURL("/actions/snapshots/create_wizard", p.IsFullSnapshot, p.Offset)
	newBtn := components.ButtonFull("snapshots-new-btn", i18n.T(lang, "snapshots.new"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: newEndpoint,
			TargetID: ModalSlotID,
			Loading:  "section",
		},
	)
	newBtn.Props["size"] = "sm"
	newBtn.Props["align_self"] = "right"

	row := components.Row("snapshots-filter-row",
		[]string{"240px", "1fr", "auto", "auto"},
		sel, filler, autoBtn, newBtn,
	)
	row.Props["justify_items"] = "center"
	return row
}

// --- table ---

func buildTable(snaps []Snapshot, byID map[string]assetscatalog.Asset, p ListParams, lang string) components.Component {
	cols := []components.TableColumn{
		{ID: "date", Header: i18n.T(lang, "snapshots.col.date"), Width: "180px"},
		{ID: "type", Header: i18n.T(lang, "snapshots.col.type"), Width: "100px"},
		{ID: "entries", Header: i18n.T(lang, "snapshots.col.entries"), Width: "80px"},
		{ID: "sources", Header: i18n.T(lang, "snapshots.col.sources"), Width: "1fr"},
		{ID: "notes", Header: i18n.T(lang, "snapshots.col.notes"), Width: "1fr"},
		{ID: "actions", Header: "", Width: "80px"},
	}
	rows := make([]components.Component, 0, len(snaps))
	for _, s := range snaps {
		rows = append(rows, buildRow(s, byID, lang))
	}
	return components.Table("snapshots-table", cols, rows...)
}

func buildRow(s Snapshot, byID map[string]assetscatalog.Asset, lang string) components.Component {
	dateCell := components.Text("snapshot-"+s.ID+"-date", formatDateTime(s.RecordedAt), "sm", "normal")
	typeCell := buildTypeCell(s, lang)
	entriesCell := components.Text("snapshot-"+s.ID+"-entries", fmt.Sprintf("%d", len(s.Entries)), "sm", "normal")
	sourcesCell := components.Text("snapshot-"+s.ID+"-sources", buildSourcesCompact(s.Entries), "sm", "normal")
	notesCell := components.Text("snapshot-"+s.ID+"-notes", notesString(s.Notes), "sm", "normal")
	actionsCell := buildRowActions(s.ID)

	cells := []components.Component{dateCell, typeCell, entriesCell, sourcesCell, notesCell, actionsCell}
	details := buildEntryTable(s, byID, lang)

	return components.TableRowExpandable("snapshot-"+s.ID, cells, details)
}

func buildTypeCell(s Snapshot, lang string) components.Component {
	color := "neutral"
	textKey := "snapshots.type.partial"
	if s.IsFullSnapshot {
		color = "positive"
		textKey = "snapshots.type.full"
	}
	return components.TextStyled(
		"snapshot-"+s.ID+"-type",
		i18n.T(lang, textKey),
		"sm", "bold", "", color, "", "",
	)
}

func buildRowActions(snapshotID string) components.Component {
	editEndpoint := "/actions/snapshots/edit_wizard?id=" + snapshotID
	deleteEndpoint := "/actions/snapshots/delete_modal?id=" + snapshotID

	editBtn := components.ButtonFull("snapshot-edit-"+snapshotID, "", "", "secondary", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: editEndpoint,
			TargetID: ModalSlotID,
			Loading:  "section",
		},
	)
	editBtn.Props["icon"] = "pencil"
	editBtn.Props["size"] = "sm"

	deleteBtn := components.ButtonFull("snapshot-delete-"+snapshotID, "", "", "destructive", "ghost",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: deleteEndpoint,
			TargetID: ModalSlotID,
			Loading:  "section",
		},
	)
	deleteBtn.Props["icon"] = "trash"
	deleteBtn.Props["size"] = "sm"

	return components.RowWithGap("snapshot-actions-"+snapshotID, []string{"auto", "auto"}, "sm", editBtn, deleteBtn)
}

// --- entry table (expanded details) ---

func buildEntryTable(s Snapshot, byID map[string]assetscatalog.Asset, lang string) components.Component {
	cols := []components.TableColumn{
		{ID: "asset", Header: i18n.T(lang, "snapshots.entries.col.asset"), Width: "120px"},
		{ID: "quantity", Header: i18n.T(lang, "snapshots.entries.col.quantity"), Width: "1fr", Align: "right"},
		{ID: "price", Header: i18n.T(lang, "snapshots.entries.col.price"), Width: "1fr", Align: "right"},
		{ID: "value_override", Header: i18n.T(lang, "snapshots.entries.col.value_override"), Width: "1fr", Align: "right"},
		{ID: "source", Header: i18n.T(lang, "snapshots.entries.col.source"), Width: "120px"},
	}
	rows := make([]components.Component, 0, len(s.Entries))
	for i, e := range s.Entries {
		rows = append(rows, buildEntryRow(s.ID, i, e, byID, lang))
	}
	return components.Table("snapshots-row-"+s.ID+"-entries", cols, rows...)
}

func buildEntryRow(snapshotID string, idx int, e Entry, byID map[string]assetscatalog.Asset, lang string) components.Component {
	rowID := fmt.Sprintf("snapshot-%s-entry-%d", snapshotID, idx)

	asset, hasAsset := byID[e.AssetID]
	tickerLabel := e.AssetID
	currency := ""
	if hasAsset {
		tickerLabel = asset.Ticker
		currency = asset.Currency
	}

	assetCell := components.Text(rowID+"-asset", tickerLabel, "sm", "bold")

	qtyVal := parseFloat(e.Quantity)
	qtyCell := components.Text(rowID+"-quantity", format.FormatQuantity(qtyVal, lang), "sm", "normal")

	priceVal := parseFloat(e.CurrentPrice)
	priceCell := components.Text(rowID+"-price", format.FormatMoney(priceVal, currency, lang), "sm", "normal")

	overrideVal := parseFloat(e.CurrentValueOverride)
	overrideCell := components.Text(rowID+"-value_override", format.FormatMoney(overrideVal, currency, lang), "sm", "normal")

	sourceCell := components.TextStyled(rowID+"-source", e.Source, "sm", "normal", "", "neutral", "", "")

	return components.TableRow(rowID, assetCell, qtyCell, priceCell, overrideCell, sourceCell)
}

// --- empty state ---

func buildEmpty(p ListParams, lang string) components.Component {
	titleKey := "snapshots.empty_title"
	subKey := "snapshots.empty_subtitle"
	if p.IsFullSnapshot != nil {
		titleKey = "snapshots.empty_filtered_title"
		subKey = "snapshots.empty_filtered_subtitle"
	}
	title := components.Text("empty-title", i18n.T(lang, titleKey), "lg", "bold")
	sub := components.TextStyled("empty-subtitle", i18n.T(lang, subKey), "md", "normal", "", "muted", "", "")
	return components.ColumnWithGap("snapshots-empty", "xs", title, sub)
}

// --- pagination ---

func buildPagination(res *ListResult, p ListParams, lang string) components.Component {
	size := res.Size
	if size <= 0 {
		size = 10
	}
	currentPage := (p.Offset / size) + 1
	totalPages := (res.Total + size - 1) / size

	prevOffset := p.Offset - size
	if prevOffset < 0 {
		prevOffset = 0
	}
	nextOffset := p.Offset + size

	prevURL := buildPaginationURL(p.IsFullSnapshot, prevOffset)
	nextURL := buildPaginationURL(p.IsFullSnapshot, nextOffset)

	prev := paginationButton("pagination-prev", i18n.T(lang, "snapshots.pagination.prev"), prevURL, p.Offset == 0)
	next := paginationButton("pagination-next", i18n.T(lang, "snapshots.pagination.next"), nextURL, p.Offset+size >= res.Total)

	infoText := renderPageOf(i18n.T(lang, "snapshots.pagination.page_of"), currentPage, totalPages)
	info := components.TextStyled("pagination-info", infoText, "sm", "normal", "", "muted", "", "")

	leftSpacer := components.Column("pagination-left-spacer")
	rightSpacer := components.Column("pagination-right-spacer")
	row := components.Row("snapshots-pagination",
		[]string{"1fr", "auto", "auto", "auto", "1fr"},
		leftSpacer, prev, info, next, rightSpacer,
	)
	row.Props["gap"] = "sm"
	row.Props["justify_items"] = "center"
	row.Props["align_items"] = "center"
	return row
}

func paginationButton(id, label, endpoint string, disabled bool) components.Component {
	btn := components.ButtonFull(id, label, "", "secondary", "ghost",
		components.Reload(endpoint, SectionID),
	)
	if disabled {
		btn.Props["disabled"] = true
	}
	return btn
}

func renderPageOf(template string, current, total int) string {
	s := strings.ReplaceAll(template, "{current}", fmt.Sprintf("%d", current))
	s = strings.ReplaceAll(s, "{total}", fmt.Sprintf("%d", total))
	return s
}

// --- helpers ---

func indexCatalog(catalog []assetscatalog.Asset) map[string]assetscatalog.Asset {
	m := make(map[string]assetscatalog.Asset, len(catalog))
	for _, a := range catalog {
		m[a.ID] = a
	}
	return m
}

// buildListURL builds a URL like /actions/snapshots/list?is_full_snapshot=<f>&offset=<n>.
// isFull == nil → param omitted. offset == 0 → param omitted (for header buttons and filter).
func buildListURL(base string, isFull *bool, offset int) string {
	v := url.Values{}
	if isFull != nil {
		v.Set("is_full_snapshot", strconv.FormatBool(*isFull))
	}
	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
}

// buildPaginationURL is like buildListURL but always includes offset (even 0),
// so pagination buttons carry the explicit target offset.
func buildPaginationURL(isFull *bool, offset int) string {
	v := url.Values{}
	if isFull != nil {
		v.Set("is_full_snapshot", strconv.FormatBool(*isFull))
	}
	v.Set("offset", strconv.Itoa(offset))
	return "/actions/snapshots/list?" + v.Encode()
}

// formatDateTime converts an RFC3339 string to "YYYY-MM-DD HH:mm".
func formatDateTime(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		// Fallback: return as-is trimmed to 16 chars if possible.
		if len(rfc3339) >= 16 {
			return rfc3339[:10] + " " + rfc3339[11:16]
		}
		return rfc3339
	}
	return t.UTC().Format("2006-01-02 15:04")
}

// buildSourcesCompact collects unique sources in first-appearance order,
// returns at most 3 joined with " · ", and appends " +N" if more.
func buildSourcesCompact(entries []Entry) string {
	if len(entries) == 0 {
		return "—"
	}
	seen := make(map[string]bool)
	var ordered []string
	for _, e := range entries {
		if !seen[e.Source] {
			seen[e.Source] = true
			ordered = append(ordered, e.Source)
		}
	}
	if len(ordered) == 0 {
		return "—"
	}
	const maxVisible = 3
	if len(ordered) <= maxVisible {
		return strings.Join(ordered, " · ")
	}
	extra := len(ordered) - maxVisible
	return strings.Join(ordered[:maxVisible], " · ") + fmt.Sprintf(" +%d", extra)
}

// notesString returns the notes string truncated to 40 runes, or "—" if empty.
// Rune-aware so multibyte characters don't get sliced mid-sequence.
func notesString(s string) string {
	if s == "" {
		return "—"
	}
	runes := []rune(s)
	if len(runes) <= 40 {
		return s
	}
	return string(runes[:40]) + "\u2026"
}

// parseFloat converts a string to *float64. Empty string or parse error → nil.
func parseFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// boolPtrToString converts *bool to its string form for the select default_value.
// nil → ""; true → "true"; false → "false".
func boolPtrToString(b *bool) string {
	if b == nil {
		return ""
	}
	return strconv.FormatBool(*b)
}
