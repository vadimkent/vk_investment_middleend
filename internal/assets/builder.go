package assets

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildScreen returns the full SDUI tree for GET /screens/assets.
func BuildScreen(result *ListResult, params ListParams, lang string) components.Component {
	section := BuildAssetsSection(result, params, lang)
	root := components.ColumnWithGap("assets-root", "lg", section)
	return components.Screen("assets", i18n.T(lang, "assets.title"), root)
}

// BuildAssetsSection returns the replaceable subtree shared by both handlers.
func BuildAssetsSection(result *ListResult, params ListParams, lang string) components.Component {
	children := []components.Component{buildFilterForm(params, lang)}

	if len(result.Assets) == 0 {
		children = append(children, buildEmpty(params, lang))
	} else {
		children = append(children, buildTable(result.Assets, lang))
		if result.Total > result.Size {
			children = append(children, buildPagination(result, params, lang))
		}
	}

	return components.ColumnWithGap("assets-section", "sm", children...)
}

func buildFilterForm(params ListParams, lang string) components.Component {
	opts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "STOCK", Label: "STOCK"},
		{Value: "ETF", Label: "ETF"},
		{Value: "CRYPTO", Label: "CRYPTO"},
		{Value: "BOND", Label: "BOND"},
	}
	sel := components.Component{
		Type: "select",
		ID:   "asset-type-select",
		Props: map[string]any{
			"name":          "asset_type",
			"label":         i18n.T(lang, "assets.filter.type"),
			"default_value": params.AssetType,
			"options":       opts,
		},
		Actions: []components.Action{
			{
				Trigger:  "change",
				Type:     "submit",
				Method:   "GET",
				Endpoint: "/actions/assets/list",
				TargetID: "assets-filter-form",
				Loading:  "section",
			},
		},
	}
	filler := components.Spacer("filter-spacer", "none")
	row := components.Row("assets-filter-row", []string{"240px", "1fr"}, sel, filler)
	return components.Form("assets-filter-form", row)
}

func buildTable(assets []Asset, lang string) components.Component {
	cols := []components.TableColumn{
		{ID: "ticker", Header: i18n.T(lang, "assets.col.ticker"), Width: "120px"},
		{ID: "name", Header: i18n.T(lang, "assets.col.name"), Width: "1fr"},
		{ID: "type", Header: i18n.T(lang, "assets.col.type"), Width: "100px"},
		{ID: "currency", Header: i18n.T(lang, "assets.col.currency"), Width: "100px"},
		{ID: "complex", Header: i18n.T(lang, "assets.col.complex"), Width: "100px", Align: "center"},
		{ID: "price_provider", Header: i18n.T(lang, "assets.col.price_provider"), Width: "160px"},
	}
	rows := make([]components.Component, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, buildRow(a))
	}
	return components.Table("assets-table", cols, rows...)
}

func buildRow(a Asset) components.Component {
	cell := func(id, content string) components.Component {
		return components.Text(id, content, "sm", "normal")
	}
	ticker := components.Text("asset-"+a.ID+"-ticker", strings.ToUpper(a.Ticker), "sm", "bold")
	complexCell := "—"
	if a.IsComplex {
		complexCell = "✓"
	}
	providerCell := "—"
	if !a.IsComplex && a.PriceProvider != nil {
		providerCell = *a.PriceProvider
	}
	return components.TableRow("asset-"+a.ID,
		ticker,
		cell("asset-"+a.ID+"-name", a.Name),
		cell("asset-"+a.ID+"-type", a.AssetType),
		cell("asset-"+a.ID+"-currency", strings.ToUpper(a.Currency)),
		cell("asset-"+a.ID+"-complex", complexCell),
		cell("asset-"+a.ID+"-price_provider", providerCell),
	)
}

func buildPagination(result *ListResult, params ListParams, lang string) components.Component {
	size := result.Size
	if size <= 0 {
		size = 10
	}
	currentPage := (result.Offset / size) + 1
	totalPages := (result.Total + size - 1) / size

	prevOffset := result.Offset - size
	if prevOffset < 0 {
		prevOffset = 0
	}
	nextOffset := result.Offset + size

	prev := paginationButton("pagination-prev", i18n.T(lang, "assets.pagination.prev"),
		paginationURL(params.AssetType, prevOffset), result.Offset == 0)
	next := paginationButton("pagination-next", i18n.T(lang, "assets.pagination.next"),
		paginationURL(params.AssetType, nextOffset), result.Offset+size >= result.Total)

	infoText := renderPageOf(i18n.T(lang, "assets.pagination.page_of"), currentPage, totalPages)
	info := components.TextStyled("pagination-info", infoText, "sm", "normal", "", "muted", "", "")

	row := components.Row("assets-pagination", []string{"auto", "1fr", "auto"}, prev, info, next)
	row.Props["gap"] = "md"
	return row
}

func paginationButton(id, label, endpoint string, disabled bool) components.Component {
	btn := components.ButtonFull(id, label, "", "secondary", "ghost",
		components.Reload(endpoint, "assets-section"),
	)
	if disabled {
		btn.Props["disabled"] = true
	}
	return btn
}

func paginationURL(assetType string, offset int) string {
	v := url.Values{}
	if assetType != "" {
		v.Set("asset_type", assetType)
	}
	v.Set("offset", strconv.Itoa(offset))
	return "/actions/assets/list?" + v.Encode()
}

func renderPageOf(template string, current, total int) string {
	s := strings.ReplaceAll(template, "{current}", fmt.Sprintf("%d", current))
	s = strings.ReplaceAll(s, "{total}", fmt.Sprintf("%d", total))
	return s
}

func buildEmpty(params ListParams, lang string) components.Component {
	titleKey := "assets.empty_title"
	subKey := "assets.empty_subtitle"
	if params.AssetType != "" {
		titleKey = "assets.empty_filtered_title"
		subKey = "assets.empty_filtered_subtitle"
	}
	title := components.Text("empty-title", i18n.T(lang, titleKey), "lg", "bold")
	sub := components.TextStyled("empty-subtitle", i18n.T(lang, subKey), "md", "normal", "", "muted", "", "")
	return components.ColumnWithGap("assets-empty", "xs", title, sub)
}
