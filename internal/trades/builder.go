package trades

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
	"github.com/project/vk-investment-middleend/internal/shared/format"
)

// Exported identifiers used by the screen/list handlers and by the modal
// handlers to target subtrees for replacement.
const (
	ScreenID    = "trades-screen"
	SectionID   = "trades-section"
	ModalSlotID = "trades-modal-slot"
)

// BuildScreen returns the full SDUI tree for GET /screens/trades.
func BuildScreen(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
	header := buildHeader(lang)
	section := BuildTradesSection(res, catalog, p, lang)
	modalSlot := components.Column(ModalSlotID)
	root := components.ColumnWithGap("trades-root", "lg", header, section, modalSlot)
	return components.Screen(ScreenID, i18n.T(lang, "trades.title"), root)
}

// BuildTradesSection returns the replaceable subtree: filters, then table (or
// empty state), then pagination (when total > size).
func BuildTradesSection(res *ListResult, catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
	children := []components.Component{buildFilters(catalog, p, lang)}

	if len(res.Trades) == 0 {
		children = append(children, buildEmpty(p, lang))
	} else {
		byID := indexCatalog(catalog)
		children = append(children, buildTable(res.Trades, byID, p, lang))
		if res.Total > res.Size {
			children = append(children, buildPagination(res, p, lang))
		}
	}

	return components.ColumnWithGap(SectionID, "sm", children...)
}

// --- header ---

func buildHeader(lang string) components.Component {
	title := components.Text("trades-title", i18n.T(lang, "trades.title"), "lg", "bold")
	spacer := components.Column("trades-header-spacer")
	newBtn := components.ButtonFull("trades-new-btn", i18n.T(lang, "trades.new"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "reload",
			Endpoint: "/actions/trades/create_modal",
			TargetID: ModalSlotID,
			Loading:  "section",
		},
	)
	newBtn.Props["size"] = "sm"
	newBtn.Props["justify_self"] = "bottom"
	return components.Row("trades-header-row", []string{"auto", "1fr", "auto"}, title, spacer, newBtn)
}

// --- filters ---

func buildFilters(catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
	assetSel := buildAssetFilter(catalog, p, lang)
	typeSel := buildTypeFilter(p, lang)
	filler := components.Column("trades-filter-spacer")
	row := components.Row("trades-filter-row", []string{"240px", "200px", "1fr"}, assetSel, typeSel, filler)
	return row
}

func buildAssetFilter(catalog []assetscatalog.Asset, p ListParams, lang string) components.Component {
	opts := make([]components.SelectOption, 0, len(catalog)+1)
	opts = append(opts, components.SelectOption{Value: "", Label: i18n.T(lang, "trades.filter.asset_any")})
	for _, a := range catalog {
		opts = append(opts, components.SelectOption{Value: a.ID, Label: a.Ticker})
	}

	// on_change URL preserves current trade_type filter while replacing asset_id with {value}.
	endpoint := buildListURL(p, map[string]string{"asset_id": "{value}", "offset": ""})

	return components.Component{
		Type: "select",
		ID:   "trades-filter-asset",
		Props: map[string]any{
			"name":          "asset_id",
			"label":         i18n.T(lang, "trades.filter.asset"),
			"default_value": p.AssetID,
			"options":       opts,
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "reload",
				Endpoint: endpoint,
				TargetID: SectionID,
				Loading:  "section",
			},
		},
	}
}

func buildTypeFilter(p ListParams, lang string) components.Component {
	opts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "trades.filter.type_all")},
		{Value: "BUY", Label: i18n.T(lang, "trades.filter.type_buy")},
		{Value: "SELL", Label: i18n.T(lang, "trades.filter.type_sell")},
	}

	endpoint := buildListURL(p, map[string]string{"trade_type": "{value}", "offset": ""})

	return components.Component{
		Type: "select",
		ID:   "trades-filter-type",
		Props: map[string]any{
			"name":          "trade_type",
			"label":         i18n.T(lang, "trades.filter.type"),
			"default_value": p.TradeType,
			"options":       opts,
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "reload",
				Endpoint: endpoint,
				TargetID: SectionID,
				Loading:  "section",
			},
		},
	}
}

// --- table ---

func buildTable(trades []Trade, byID map[string]assetscatalog.Asset, p ListParams, lang string) components.Component {
	cols := []components.TableColumn{
		{ID: "date", Header: i18n.T(lang, "trades.col.date"), Width: "110px"},
		{ID: "asset", Header: i18n.T(lang, "trades.col.asset"), Width: "100px"},
		{ID: "type", Header: i18n.T(lang, "trades.col.type"), Width: "80px"},
		{ID: "quantity", Header: i18n.T(lang, "trades.col.quantity"), Width: "120px", Align: "right"},
		{ID: "price", Header: i18n.T(lang, "trades.col.price"), Width: "120px", Align: "right"},
		{ID: "total", Header: i18n.T(lang, "trades.col.total"), Width: "140px", Align: "right"},
		{ID: "fees", Header: i18n.T(lang, "trades.col.fees"), Width: "100px", Align: "right"},
		{ID: "source", Header: i18n.T(lang, "trades.col.source"), Width: "100px"},
		{ID: "notes", Header: i18n.T(lang, "trades.col.notes"), Width: "1fr"},
		{ID: "actions", Header: "", Width: "120px", Align: "right"},
	}
	rows := make([]components.Component, 0, len(trades))
	for _, t := range trades {
		rows = append(rows, buildRow(t, byID, p, lang))
	}
	return components.Table("trades-table", cols, rows...)
}

func buildRow(t Trade, byID map[string]assetscatalog.Asset, p ListParams, lang string) components.Component {
	cell := func(id, content string) components.Component {
		return components.Text(id, content, "sm", "normal")
	}

	asset, hasAsset := byID[t.AssetID]
	assetLabel := t.AssetID
	currency := ""
	if hasAsset {
		assetLabel = asset.Ticker
		currency = asset.Currency
	}

	// Type cell: colored text (codebase has no pill-badge primitive; Badge is a
	// dot-overlay component that wraps a child — not semantically a type label).
	// Using TextStyled with positive/negative colors matches existing codebase
	// conventions (see portfolio live/summary builders).
	typeColor := "positive"
	if strings.ToUpper(t.TradeType) == "SELL" {
		typeColor = "negative"
	}
	typeCell := components.TextStyled(
		"trade-"+t.ID+"-type",
		strings.ToUpper(t.TradeType),
		"sm", "bold", "", typeColor, "", "",
	)

	dateCell := cell("trade-"+t.ID+"-date", dateOnly(t.Date))
	assetCell := cell("trade-"+t.ID+"-asset", assetLabel)
	qtyCell := cell("trade-"+t.ID+"-quantity", quantityString(t.Quantity, lang))
	priceCell := cell("trade-"+t.ID+"-price", priceString(t.PricePerUnit, currency, lang))
	totalCell := cell("trade-"+t.ID+"-total", totalString(t.Quantity, t.PricePerUnit, currency, lang))
	feesCell := cell("trade-"+t.ID+"-fees", feesString(t.Fees, currency, lang))
	sourceCell := cell("trade-"+t.ID+"-source", strings.ToUpper(t.Source))
	notesCell := cell("trade-"+t.ID+"-notes", truncateNotes(t.Notes))

	actionsRow := buildRowActions(t.ID, p)

	return components.TableRow("trade-"+t.ID,
		dateCell, assetCell, typeCell, qtyCell, priceCell, totalCell, feesCell, sourceCell, notesCell, actionsRow,
	)
}

func buildRowActions(tradeID string, p ListParams) components.Component {
	editEndpoint := buildActionURL("/actions/trades/edit_modal", p, map[string]string{"id": tradeID})
	deleteEndpoint := buildActionURL("/actions/trades/delete_modal", p, map[string]string{"id": tradeID})

	editBtn := components.ButtonFull("trade-edit-"+tradeID, "", "", "secondary", "ghost",
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

	deleteBtn := components.ButtonFull("trade-delete-"+tradeID, "", "", "destructive", "ghost",
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

	return components.RowWithGap("trade-actions-"+tradeID, []string{"auto", "auto"}, "sm", editBtn, deleteBtn)
}

// --- empty ---

func buildEmpty(p ListParams, lang string) components.Component {
	titleKey := "trades.empty_title"
	subKey := "trades.empty_subtitle"
	if p.AssetID != "" || p.TradeType != "" {
		titleKey = "trades.empty_filtered_title"
		subKey = "trades.empty_filtered_subtitle"
	}
	title := components.Text("empty-title", i18n.T(lang, titleKey), "lg", "bold")
	sub := components.TextStyled("empty-subtitle", i18n.T(lang, subKey), "md", "normal", "", "muted", "", "")
	return components.ColumnWithGap("trades-empty", "xs", title, sub)
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

	prev := paginationButton("pagination-prev", i18n.T(lang, "trades.pagination.prev"),
		buildListURL(p, map[string]string{"offset": strconv.Itoa(prevOffset)}), p.Offset == 0)
	next := paginationButton("pagination-next", i18n.T(lang, "trades.pagination.next"),
		buildListURL(p, map[string]string{"offset": strconv.Itoa(nextOffset)}), p.Offset+size >= res.Total)

	infoText := renderPageOf(i18n.T(lang, "trades.pagination.page_of"), currentPage, totalPages)
	info := components.TextStyled("pagination-info", infoText, "sm", "normal", "", "muted", "", "")

	leftSpacer := components.Column("pagination-left-spacer")
	rightSpacer := components.Column("pagination-right-spacer")
	row := components.Row("trades-pagination",
		[]string{"1fr", "auto", "auto", "auto", "1fr"},
		leftSpacer, prev, info, next, rightSpacer)
	row.Props["gap"] = "sm"
	row.Props["justify_items"] = "center"
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

// buildListURL returns /actions/trades/list?... with the current list params,
// applying overrides. An override whose value is "" drops the key; any other
// value (including "{value}" for Select on_change interpolation) is set verbatim.
func buildListURL(p ListParams, overrides map[string]string) string {
	return buildActionURL("/actions/trades/list", p, overrides)
}

// buildActionURL is the shared builder used by list, per-row edit/delete, and
// pagination: it starts from the current ListParams, applies overrides, drops
// empty keys, and returns `<base>?<encoded>` (no query string if all empty).
func buildActionURL(base string, p ListParams, overrides map[string]string) string {
	vals := map[string]string{
		"asset_id":   p.AssetID,
		"trade_type": p.TradeType,
	}
	if p.Offset > 0 {
		vals["offset"] = strconv.Itoa(p.Offset)
	}
	for k, v := range overrides {
		vals[k] = v
	}

	v := url.Values{}
	// Stable order: id, asset_id, trade_type, offset, then any extras.
	order := []string{"id", "asset_id", "trade_type", "offset"}
	seen := map[string]bool{}
	for _, k := range order {
		if val, ok := vals[k]; ok && val != "" {
			v.Set(k, val)
			seen[k] = true
		}
	}
	for k, val := range vals {
		if seen[k] || val == "" {
			continue
		}
		v.Set(k, val)
	}

	if len(v) == 0 {
		return base
	}
	encoded := v.Encode()
	// url.Values.Encode escapes { and } → we need literal {value} for Select interpolation.
	encoded = strings.ReplaceAll(encoded, "%7Bvalue%7D", "{value}")
	return base + "?" + encoded
}

func dateOnly(rfc3339 string) string {
	if len(rfc3339) >= 10 {
		return rfc3339[:10]
	}
	return rfc3339
}

func quantityString(qty, lang string) string {
	v, err := strconv.ParseFloat(qty, 64)
	if err != nil {
		return "—"
	}
	return format.FormatQuantity(&v, lang)
}

func priceString(ppu, currency, lang string) string {
	v, err := strconv.ParseFloat(ppu, 64)
	if err != nil {
		return "—"
	}
	return format.FormatMoney(&v, currency, lang)
}

func totalString(qty, ppu, currency, lang string) string {
	q, err := strconv.ParseFloat(qty, 64)
	if err != nil {
		return "—"
	}
	p, err := strconv.ParseFloat(ppu, 64)
	if err != nil {
		return "—"
	}
	total := q * p
	return format.FormatMoney(&total, currency, lang)
}

func feesString(fees, currency, lang string) string {
	if fees == "" || fees == "0" {
		return "—"
	}
	v, err := strconv.ParseFloat(fees, 64)
	if err != nil {
		return "—"
	}
	return format.FormatMoney(&v, currency, lang)
}

func truncateNotes(s string) string {
	if len(s) <= 40 {
		return s
	}
	return s[:40] + "\u2026"
}
