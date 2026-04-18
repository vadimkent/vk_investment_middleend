package assets

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

func TestMain(m *testing.M) {
	// Load locales for i18n.T resolution.
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile -> internal/assets/builder_test.go; walk up to repo root.
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
	m.Run()
}

func findByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if got := findByID(c.Children[i], id); got != nil {
			return got
		}
	}
	return nil
}

func sampleAsset(id, ticker string, isComplex bool, provider *string) Asset {
	return Asset{
		ID: id, Ticker: ticker, Name: "Name-" + ticker,
		AssetType: "STOCK", Currency: "USD",
		IsComplex: isComplex, PriceProvider: provider,
	}
}

func TestBuildScreen_ShapeAndTitle(t *testing.T) {
	provider := "TWELVE_DATA"
	result := &ListResult{
		Assets: []Asset{sampleAsset("a1", "AAPL", false, &provider)},
		Total:  1, Size: 10, Offset: 0,
	}
	tree := BuildScreen(result, ListParams{}, "en")

	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, "assets", tree.ID)
	assert.Equal(t, "Assets", tree.Props["title"])

	root := findByID(tree, "assets-root")
	require.NotNil(t, root)
	assert.Equal(t, "column", root.Type)

	section := findByID(tree, "assets-section")
	require.NotNil(t, section)
	assert.Equal(t, "column", section.Type)

	modalSlot := findByID(tree, "assets-modal-slot")
	require.NotNil(t, modalSlot)
	assert.Equal(t, "column", modalSlot.Type)
	assert.Empty(t, modalSlot.Children)
}

func TestBuildAssetsSection_FilterSelectAction(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{AssetType: "STOCK"}, "en")

	sel := findByID(section, "asset-type-select")
	require.NotNil(t, sel)
	assert.Equal(t, "select", sel.Type)
	assert.Equal(t, "STOCK", sel.Props["default_value"])

	require.Len(t, sel.Actions, 1)
	act := sel.Actions[0]
	assert.Equal(t, "change", act.Trigger)
	assert.Equal(t, "reload", act.Type)
	assert.Equal(t, "/actions/assets/list?asset_type={value}", act.Endpoint)
	assert.Equal(t, "assets-section", act.TargetID)
	assert.Equal(t, "section", act.Loading)
}

func TestBuildAssetsSection_FilterSelectOptions(t *testing.T) {
	section := BuildAssetsSection(&ListResult{Size: 10}, ListParams{}, "en")
	sel := findByID(section, "asset-type-select")
	require.NotNil(t, sel)

	opts, _ := json.Marshal(sel.Props["options"])
	var parsed []components.SelectOption
	require.NoError(t, json.Unmarshal(opts, &parsed))
	require.Len(t, parsed, 5)
	assert.Equal(t, "", parsed[0].Value)
	assert.Equal(t, "Any", parsed[0].Label)
	assert.Equal(t, "STOCK", parsed[1].Value)
	assert.Equal(t, "ETF", parsed[2].Value)
	assert.Equal(t, "CRYPTO", parsed[3].Value)
	assert.Equal(t, "BOND", parsed[4].Value)
}

func TestBuildAssetsSection_TableColumnsAndRows(t *testing.T) {
	provider := "TWELVE_DATA"
	result := &ListResult{
		Assets: []Asset{
			sampleAsset("a1", "AAPL", false, &provider),
			sampleAsset("a2", "HOUSE", true, nil),
			sampleAsset("a3", "TSLA", false, nil),
		},
		Total: 3, Size: 10, Offset: 0,
	}
	section := BuildAssetsSection(result, ListParams{}, "en")
	table := findByID(section, "assets-table")
	require.NotNil(t, table)
	assert.Equal(t, "table", table.Type)

	colsRaw, _ := json.Marshal(table.Props["columns"])
	var cols []components.TableColumn
	require.NoError(t, json.Unmarshal(colsRaw, &cols))
	require.Len(t, cols, 7)
	assert.Equal(t, []string{"ticker", "name", "type", "currency", "complex", "price_provider", "actions"},
		[]string{cols[0].ID, cols[1].ID, cols[2].ID, cols[3].ID, cols[4].ID, cols[5].ID, cols[6].ID})

	require.Len(t, table.Children, 3)

	// Row 1: AAPL, not complex, provider set.
	r1 := table.Children[0]
	require.Equal(t, "table_row", r1.Type)
	require.Len(t, r1.Children, 7)
	assert.Equal(t, "AAPL", r1.Children[0].Props["content"])
	assert.Equal(t, "—", r1.Children[4].Props["content"]) // complex=false renders "—"
	assert.Equal(t, "TWELVE_DATA", r1.Children[5].Props["content"])

	// Row 2: HOUSE, complex=true, provider null.
	r2 := table.Children[1]
	require.Len(t, r2.Children, 7)
	assert.Equal(t, "✓", r2.Children[4].Props["content"])
	assert.Equal(t, "—", r2.Children[5].Props["content"]) // complex -> dash

	// Row 3: TSLA, not complex, provider null.
	r3 := table.Children[2]
	require.Len(t, r3.Children, 7)
	assert.Equal(t, "—", r3.Children[4].Props["content"])
	assert.Equal(t, "—", r3.Children[5].Props["content"])

	actionsR1 := findByID(r1, "actions-a1")
	require.NotNil(t, actionsR1)
	editBtn := findByID(*actionsR1, "edit-a1")
	require.NotNil(t, editBtn)
	require.Len(t, editBtn.Actions, 1)
	assert.Equal(t, "reload", editBtn.Actions[0].Type)
	assert.Equal(t, "/actions/assets/edit_modal?id=a1", editBtn.Actions[0].Endpoint)
	assert.Equal(t, "assets-modal-slot", editBtn.Actions[0].TargetID)

	deleteBtn := findByID(*actionsR1, "delete-a1")
	require.NotNil(t, deleteBtn)
	assert.Equal(t, "/actions/assets/delete_modal?id=a1", deleteBtn.Actions[0].Endpoint)
}

func TestBuildAssetsSection_PaginationOmittedWhenTotalFits(t *testing.T) {
	result := &ListResult{
		Assets: []Asset{sampleAsset("a1", "AAPL", false, nil)},
		Total:  1, Size: 10, Offset: 0,
	}
	section := BuildAssetsSection(result, ListParams{}, "en")
	assert.Nil(t, findByID(section, "assets-pagination"))
}

func TestBuildAssetsSection_PaginationFirstPage(t *testing.T) {
	assets := make([]Asset, 10)
	for i := range assets {
		assets[i] = sampleAsset("a", "T", false, nil)
	}
	result := &ListResult{Assets: assets, Total: 25, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{AssetType: "STOCK"}, "en")

	pag := findByID(section, "assets-pagination")
	require.NotNil(t, pag)

	prev := findByID(*pag, "pagination-prev")
	require.NotNil(t, prev)
	assert.Equal(t, true, prev.Props["disabled"])

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	assert.NotEqual(t, true, next.Props["disabled"])
	require.Len(t, next.Actions, 1)
	assert.Equal(t, "reload", next.Actions[0].Type)
	assert.Contains(t, next.Actions[0].Endpoint, "asset_type=STOCK")
	assert.Contains(t, next.Actions[0].Endpoint, "offset=10")
	assert.Equal(t, "assets-section", next.Actions[0].TargetID)

	info := findByID(*pag, "pagination-info")
	require.NotNil(t, info)
	assert.Equal(t, "Page 1 of 3", info.Props["content"])
}

func TestBuildAssetsSection_PaginationLastPage(t *testing.T) {
	assets := make([]Asset, 5)
	for i := range assets {
		assets[i] = sampleAsset("a", "T", false, nil)
	}
	result := &ListResult{Assets: assets, Total: 25, Size: 10, Offset: 20}
	section := BuildAssetsSection(result, ListParams{}, "en")
	pag := findByID(section, "assets-pagination")
	require.NotNil(t, pag)

	prev := findByID(*pag, "pagination-prev")
	require.NotNil(t, prev)
	assert.NotEqual(t, true, prev.Props["disabled"])
	require.Len(t, prev.Actions, 1)
	assert.Contains(t, prev.Actions[0].Endpoint, "offset=10")

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	assert.Equal(t, true, next.Props["disabled"])

	info := findByID(*pag, "pagination-info")
	require.NotNil(t, info)
	assert.Equal(t, "Page 3 of 3", info.Props["content"])
}

func TestBuildAssetsSection_EmptyNoFilter(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{}, "en")

	// Filter row still present.
	assert.NotNil(t, findByID(section, "assets-filter-row"))
	assert.NotNil(t, findByID(section, "asset-type-select"))
	// No table, no pagination.
	assert.Nil(t, findByID(section, "assets-table"))
	assert.Nil(t, findByID(section, "assets-pagination"))

	empty := findByID(section, "assets-empty")
	require.NotNil(t, empty)
	title := findByID(*empty, "empty-title")
	sub := findByID(*empty, "empty-subtitle")
	require.NotNil(t, title)
	require.NotNil(t, sub)
	assert.Equal(t, "No assets registered yet", title.Props["content"])
	assert.Equal(t, "Once you register assets, they will appear here.", sub.Props["content"])
}

func TestBuildAssetsSection_EmptyWithFilter(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	section := BuildAssetsSection(result, ListParams{AssetType: "STOCK"}, "en")

	title := findByID(section, "empty-title")
	sub := findByID(section, "empty-subtitle")
	require.NotNil(t, title)
	require.NotNil(t, sub)
	assert.Equal(t, "No assets match the filter", title.Props["content"])
	assert.Equal(t, "Try changing or clearing the filter.", sub.Props["content"])
}

func TestBuildScreen_SpanishTitle(t *testing.T) {
	result := &ListResult{Assets: []Asset{}, Total: 0, Size: 10, Offset: 0}
	tree := BuildScreen(result, ListParams{}, "es")
	assert.Equal(t, "Activos", tree.Props["title"])
}

func TestBuildAssetsSection_NewAssetButton(t *testing.T) {
	section := BuildAssetsSection(&ListResult{Size: 10}, ListParams{}, "en")
	btn := findByID(section, "assets-new-btn")
	require.NotNil(t, btn)
	assert.Equal(t, "button", btn.Type)
	assert.Equal(t, "New Asset", btn.Props["label"])
	require.Len(t, btn.Actions, 1)
	assert.Equal(t, "reload", btn.Actions[0].Type)
	assert.Equal(t, "/actions/assets/create_modal", btn.Actions[0].Endpoint)
	assert.Equal(t, "assets-modal-slot", btn.Actions[0].TargetID)
}
