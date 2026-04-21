package snapshots

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

func TestMain(m *testing.M) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
	m.Run()
}

// --- helpers ---

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

func findText(c components.Component) string {
	if v, ok := c.Props["content"].(string); ok && v != "" {
		return v
	}
	for _, ch := range c.Children {
		if v := findText(ch); v != "" {
			return v
		}
	}
	return ""
}

func boolPtr(b bool) *bool { return &b }

func sampleCatalog() []assetscatalog.Asset {
	return []assetscatalog.Asset{
		{ID: "aaa-1", Ticker: "AAPL", Name: "Apple Inc", AssetType: "STOCK", Currency: "USD"},
		{ID: "bbb-2", Ticker: "ETH", Name: "Ethereum", AssetType: "CRYPTO", Currency: "USD"},
	}
}

func sampleSnapshots() []Snapshot {
	full := true
	_ = full
	return []Snapshot{
		{
			ID:             "s1",
			RecordedAt:     "2025-03-15T14:30:00Z",
			IsFullSnapshot: true,
			Notes:          "Quarter-end snapshot",
			Entries: []Entry{
				{AssetID: "aaa-1", Quantity: "10.5", CurrentPrice: "150.00", Source: "MANUAL"},
				{AssetID: "bbb-2", Quantity: "2", CurrentPrice: "3000.00", Source: "COINGECKO"},
			},
		},
		{
			ID:             "s2",
			RecordedAt:     "2025-03-10T09:00:00Z",
			IsFullSnapshot: false,
			Notes:          strings.Repeat("x", 60),
			Entries: []Entry{
				{AssetID: "aaa-1", Quantity: "10.5", CurrentPrice: "148.00", Source: "MANUAL"},
			},
		},
	}
}

// --- Test 1: BuildScreen shape and IDs ---

func TestBuildScreen_ShapeAndIDs(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	tree := BuildScreen(res, sampleCatalog(), ListParams{}, "en")

	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, ScreenID, tree.ID)

	section := findByID(tree, SectionID)
	require.NotNil(t, section)
	assert.Equal(t, "column", section.Type)

	modalSlot := findByID(tree, ModalSlotID)
	require.NotNil(t, modalSlot)
	assert.Equal(t, "column", modalSlot.Type)
	assert.Empty(t, modalSlot.Children)
}

func TestBuildScreen_RootHasThreeChildren(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	tree := BuildScreen(res, sampleCatalog(), ListParams{}, "en")

	// root column is the single child of the screen
	require.Len(t, tree.Children, 1)
	root := tree.Children[0]
	assert.Equal(t, "snapshots-root", root.ID)
	// header + section + modalSlot
	require.Len(t, root.Children, 3)
}

// --- Test 2: Header buttons ---

func TestBuildScreen_HeaderNewSnapshotButton(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	tree := BuildScreen(res, sampleCatalog(), ListParams{}, "en")

	btn := findByID(tree, "snapshots-new-btn")
	require.NotNil(t, btn)
	assert.Equal(t, "button", btn.Type)
	require.Len(t, btn.Actions, 1)
	act := btn.Actions[0]
	assert.Equal(t, "reload", act.Type)
	assert.Contains(t, act.Endpoint, "/actions/snapshots/create_wizard")
	assert.Equal(t, ModalSlotID, act.TargetID)
}

func TestBuildScreen_HeaderAutoSnapshotButton(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	// Pass a filter so its is_full_snapshot param appears in the URL.
	isTrue := true
	p := ListParams{IsFullSnapshot: &isTrue, Offset: 5}
	tree := BuildScreen(res, sampleCatalog(), p, "en")

	btn := findByID(tree, "snapshots-auto-btn")
	require.NotNil(t, btn)
	assert.Equal(t, "button", btn.Type)
	require.Len(t, btn.Actions, 1)
	act := btn.Actions[0]
	assert.Equal(t, "submit", act.Type)
	assert.Equal(t, "POST", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/snapshots/auto")
	assert.Equal(t, ScreenID, act.TargetID)
}

func TestBuildScreen_HeaderAutoButton_CarriesListParams(t *testing.T) {
	isTrue := true
	p := ListParams{IsFullSnapshot: &isTrue, Offset: 10}
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	tree := BuildScreen(res, sampleCatalog(), p, "en")

	btn := findByID(tree, "snapshots-auto-btn")
	require.NotNil(t, btn)
	ep := btn.Actions[0].Endpoint
	assert.Contains(t, ep, "is_full_snapshot=true")
	assert.Contains(t, ep, "offset=10")
}

// --- Test 3: Filter ---

func TestBuildSnapshotsSection_FilterSelect(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	sel := findByID(sec, "snapshots-filter-type")
	require.NotNil(t, sel)
	assert.Equal(t, "select", sel.Type)
	assert.Equal(t, "is_full_snapshot", sel.Props["name"])

	optsRaw, _ := json.Marshal(sel.Props["options"])
	var opts []components.SelectOption
	require.NoError(t, json.Unmarshal(optsRaw, &opts))
	require.Len(t, opts, 3)
	assert.Equal(t, "", opts[0].Value)
	assert.Equal(t, "true", opts[1].Value)
	assert.Equal(t, "false", opts[2].Value)

	require.Len(t, sel.Actions, 1)
	act := sel.Actions[0]
	assert.Equal(t, "change", act.Trigger)
	assert.Equal(t, "reload", act.Type)
	assert.Equal(t, SectionID, act.TargetID)
	assert.Contains(t, act.Endpoint, "/actions/snapshots/list")
	assert.Contains(t, act.Endpoint, "is_full_snapshot={value}")
}

func TestBuildSnapshotsSection_FilterSelect_DefaultValueNil(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{IsFullSnapshot: nil}, "en")
	sel := findByID(sec, "snapshots-filter-type")
	require.NotNil(t, sel)
	assert.Equal(t, "", sel.Props["default_value"])
}

func TestBuildSnapshotsSection_FilterSelect_DefaultValueTrue(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{IsFullSnapshot: boolPtr(true)}, "en")
	sel := findByID(sec, "snapshots-filter-type")
	require.NotNil(t, sel)
	assert.Equal(t, "true", sel.Props["default_value"])
}

func TestBuildSnapshotsSection_FilterSelect_DefaultValueFalse(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{IsFullSnapshot: boolPtr(false)}, "en")
	sel := findByID(sec, "snapshots-filter-type")
	require.NotNil(t, sel)
	assert.Equal(t, "false", sel.Props["default_value"])
}

// --- Test 4: Empty state (no filter) ---

func TestBuildSnapshotsSection_EmptyNoFilter(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	// Filter still present.
	assert.NotNil(t, findByID(sec, "snapshots-filter-type"))
	// No table, no pagination.
	assert.Nil(t, findByID(sec, "snapshots-table"))
	assert.Nil(t, findByID(sec, "snapshots-pagination"))

	empty := findByID(sec, "snapshots-empty")
	require.NotNil(t, empty)

	title := findByID(*empty, "empty-title")
	sub := findByID(*empty, "empty-subtitle")
	require.NotNil(t, title)
	require.NotNil(t, sub)
	// i18n keys (Task 11.1 not yet shipped → fallback to key; still correct)
	titleText := title.Props["content"].(string)
	subText := sub.Props["content"].(string)
	assert.True(t, strings.Contains(titleText, "snapshot") || titleText == "snapshots.empty_title",
		"expected snapshots.empty_title key or resolved value, got %q", titleText)
	assert.True(t, subText != "", "subtitle must not be empty")
}

// --- Test 5: Empty state (filter active) ---

func TestBuildSnapshotsSection_EmptyWithFilter(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{IsFullSnapshot: boolPtr(true)}, "en")

	// Filter still present.
	assert.NotNil(t, findByID(sec, "snapshots-filter-type"))

	empty := findByID(sec, "snapshots-empty")
	require.NotNil(t, empty)

	title := findByID(*empty, "empty-title")
	sub := findByID(*empty, "empty-subtitle")
	require.NotNil(t, title)
	require.NotNil(t, sub)

	titleText := title.Props["content"].(string)
	subText := sub.Props["content"].(string)
	// Must use the filtered keys (different from no-filter).
	assert.True(t, strings.Contains(titleText, "filter") || titleText == "snapshots.empty_filtered_title",
		"expected filtered empty title, got %q", titleText)
	assert.True(t, subText != "", "subtitle must not be empty")
}

func TestBuildSnapshotsSection_EmptyFilteredAndNoFilter_DifferentKeys(t *testing.T) {
	res := &ListResult{Snapshots: []Snapshot{}, Total: 0, Size: 10, Offset: 0}
	secNo := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")
	secFil := BuildSnapshotsSection(res, sampleCatalog(), ListParams{IsFullSnapshot: boolPtr(false)}, "en")

	emptyNo := findByID(secNo, "snapshots-empty")
	emptyFil := findByID(secFil, "snapshots-empty")
	require.NotNil(t, emptyNo)
	require.NotNil(t, emptyFil)

	titleNo := findByID(*emptyNo, "empty-title").Props["content"].(string)
	titleFil := findByID(*emptyFil, "empty-title").Props["content"].(string)
	assert.NotEqual(t, titleNo, titleFil, "unfiltered and filtered empty states must use different title keys")
}

// --- Test 6: Table with expandable rows ---

func TestBuildSnapshotsSection_TableAndExpandableRows(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	assert.Equal(t, "table", table.Type)
	require.Len(t, table.Children, 2)

	for _, row := range table.Children {
		assert.Equal(t, "table_row", row.Type)
		assert.Equal(t, true, row.Props["expandable"])
		details, ok := row.Props["details"]
		require.True(t, ok, "row must carry details prop")
		// details is []Component
		detailSlice, ok := details.([]components.Component)
		require.True(t, ok, "details must be []components.Component")
		require.NotEmpty(t, detailSlice, "details must contain at least one component")
	}
}

// --- Test 7: Main row cells ---

func TestBuildSnapshotsSection_RowCells_DateFormatted(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	// Date cell is children[0]
	dateText := row.Children[0].Props["content"].(string)
	// YYYY-MM-DD HH:mm format
	assert.Equal(t, "2025-03-15 14:30", dateText)
}

func TestBuildSnapshotsSection_RowCells_TypeBadge_Full(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	typeCell := row.Children[1]
	// Type badge for full snapshot uses "positive" color
	assert.Equal(t, "positive", typeCell.Props["color"])
}

func TestBuildSnapshotsSection_RowCells_TypeBadge_Partial(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s2")
	require.NotNil(t, row)
	typeCell := row.Children[1]
	// Type badge for partial snapshot uses "neutral" color
	assert.Equal(t, "neutral", typeCell.Props["color"])
}

func TestBuildSnapshotsSection_RowCells_EntriesCount(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	// Entries count = 2
	assert.Equal(t, "2", row.Children[2].Props["content"])

	row2 := findByID(sec, "snapshot-s2")
	require.NotNil(t, row2)
	assert.Equal(t, "1", row2.Children[2].Props["content"])
}

func TestBuildSnapshotsSection_RowCells_SourcesCompact_UpTo3(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "aaa-1", Source: "MANUAL"},
				{AssetID: "bbb-2", Source: "COINGECKO"},
				{AssetID: "aaa-1", Source: "MANUAL"}, // duplicate — should dedupe
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	sourcesText := row.Children[3].Props["content"].(string)
	assert.Equal(t, "MANUAL · COINGECKO", sourcesText)
}

func TestBuildSnapshotsSection_RowCells_SourcesCompact_MoreThan3(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "aaa-1", Source: "MANUAL"},
				{AssetID: "aaa-1", Source: "COINGECKO"},
				{AssetID: "aaa-1", Source: "TWELVE_DATA"},
				{AssetID: "aaa-1", Source: "ALPHA_VANTAGE"},
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	sourcesText := row.Children[3].Props["content"].(string)
	assert.Equal(t, "MANUAL · COINGECKO · TWELVE_DATA +1", sourcesText)
}

func TestBuildSnapshotsSection_RowCells_SourcesEmpty_Dash(t *testing.T) {
	snaps := []Snapshot{
		{ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true, Entries: []Entry{}},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	assert.Equal(t, "—", row.Children[3].Props["content"])
}

func TestBuildSnapshotsSection_RowCells_NotesTruncated(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s2")
	require.NotNil(t, row)
	notesText := row.Children[4].Props["content"].(string)
	assert.Equal(t, strings.Repeat("x", 40)+"\u2026", notesText)
}

func TestBuildSnapshotsSection_RowCells_NotesShort_NotTruncated(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	assert.Equal(t, "Quarter-end snapshot", row.Children[4].Props["content"])
}

func TestBuildSnapshotsSection_RowCells_NotesEmpty_Dash(t *testing.T) {
	snaps := []Snapshot{
		{ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true, Notes: ""},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)
	assert.Equal(t, "—", row.Children[4].Props["content"])
}

func TestBuildSnapshotsSection_RowActions(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	row := findByID(sec, "snapshot-s1")
	require.NotNil(t, row)

	edit := findByID(*row, "snapshot-edit-s1")
	require.NotNil(t, edit)
	require.Len(t, edit.Actions, 1)
	assert.Equal(t, "reload", edit.Actions[0].Type)
	assert.Equal(t, ModalSlotID, edit.Actions[0].TargetID)
	assert.Contains(t, edit.Actions[0].Endpoint, "/actions/snapshots/edit_wizard")
	assert.Contains(t, edit.Actions[0].Endpoint, "id=s1")

	del := findByID(*row, "snapshot-delete-s1")
	require.NotNil(t, del)
	require.Len(t, del.Actions, 1)
	assert.Equal(t, "reload", del.Actions[0].Type)
	assert.Equal(t, ModalSlotID, del.Actions[0].TargetID)
	assert.Contains(t, del.Actions[0].Endpoint, "/actions/snapshots/delete_modal")
	assert.Contains(t, del.Actions[0].Endpoint, "id=s1")
}

// --- Test 8: Expanded row entries table ---

func TestBuildSnapshotsSection_ExpandedEntries_TableShape(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)

	// First row (s1) details
	row := table.Children[0]
	details := row.Props["details"].([]components.Component)
	require.Len(t, details, 1)
	entryTable := details[0]
	assert.Equal(t, "table", entryTable.Type)
	assert.Equal(t, "snapshots-row-s1-entries", entryTable.ID)

	// 5 columns
	colsRaw, _ := json.Marshal(entryTable.Props["columns"])
	var cols []components.TableColumn
	require.NoError(t, json.Unmarshal(colsRaw, &cols))
	require.Len(t, cols, 5)
	assert.Equal(t, "asset", cols[0].ID)
	assert.Equal(t, "quantity", cols[1].ID)
	assert.Equal(t, "price", cols[2].ID)
	assert.Equal(t, "value_override", cols[3].ID)
	assert.Equal(t, "source", cols[4].ID)

	// s1 has 2 entries
	require.Len(t, entryTable.Children, 2)
}

func TestBuildSnapshotsSection_EntryRow_TickerResolution(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	entryTable := details[0]

	// First entry: asset aaa-1 → AAPL
	firstRow := entryTable.Children[0]
	assetCell := firstRow.Children[0]
	assert.Equal(t, "AAPL", assetCell.Props["content"])
	assert.Equal(t, "bold", assetCell.Props["weight"])
}

func TestBuildSnapshotsSection_EntryRow_TickerFallbackToUUID(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "unknown-uuid", Quantity: "1", CurrentPrice: "100", Source: "MANUAL"},
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	entryTable := details[0]
	firstRow := entryTable.Children[0]
	assert.Equal(t, "unknown-uuid", firstRow.Children[0].Props["content"])
}

func TestBuildSnapshotsSection_EntryRow_QuantityDash_WhenEmpty(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "aaa-1", Quantity: "", CurrentPrice: "100", Source: "MANUAL"},
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	entryTable := details[0]
	row := entryTable.Children[0]
	assert.Equal(t, "—", row.Children[1].Props["content"])
}

func TestBuildSnapshotsSection_EntryRow_PriceFormatted(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "aaa-1", Quantity: "5", CurrentPrice: "1234.56", Source: "MANUAL"},
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	row := details[0].Children[0]
	// aaa-1 currency is USD → $1,234.56
	assert.Equal(t, "$1,234.56", row.Children[2].Props["content"])
}

func TestBuildSnapshotsSection_EntryRow_PriceDash_WhenEmpty(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "aaa-1", Quantity: "5", CurrentPrice: "", Source: "MANUAL"},
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	row := details[0].Children[0]
	assert.Equal(t, "—", row.Children[2].Props["content"])
}

func TestBuildSnapshotsSection_EntryRow_ValueOverride(t *testing.T) {
	snaps := []Snapshot{
		{
			ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true,
			Entries: []Entry{
				{AssetID: "aaa-1", Quantity: "5", CurrentValueOverride: "9999.00", Source: "MANUAL"},
			},
		},
	}
	res := &ListResult{Snapshots: snaps, Total: 1, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	row := details[0].Children[0]
	assert.Equal(t, "$9,999.00", row.Children[3].Props["content"])
}

func TestBuildSnapshotsSection_EntryRow_SourceBadge(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	table := findByID(sec, "snapshots-table")
	require.NotNil(t, table)
	details := table.Children[0].Props["details"].([]components.Component)
	entryTable := details[0]

	// first entry source is MANUAL
	firstRow := entryTable.Children[0]
	sourceCell := firstRow.Children[4]
	// Source rendered as TextStyled (neutral badge-style) or contains "MANUAL"
	sourceText := findText(sourceCell)
	assert.Equal(t, "MANUAL", sourceText)
}

// --- Test 9: Pagination ---

func TestBuildSnapshotsSection_PaginationOmitted_WhenTotalFits(t *testing.T) {
	res := &ListResult{Snapshots: sampleSnapshots(), Total: 2, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")
	assert.Nil(t, findByID(sec, "snapshots-pagination"))
}

func TestBuildSnapshotsSection_PaginationPresent_WhenTotalExceedsSize(t *testing.T) {
	snaps := make([]Snapshot, 10)
	for i := range snaps {
		snaps[i] = Snapshot{ID: "p", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true}
	}
	res := &ListResult{Snapshots: snaps, Total: 25, Size: 10, Offset: 10}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{Offset: 10}, "en")

	pag := findByID(sec, "snapshots-pagination")
	require.NotNil(t, pag)
}

func TestBuildSnapshotsSection_Pagination_PrevDisabled_AtOffset0(t *testing.T) {
	snaps := make([]Snapshot, 10)
	for i := range snaps {
		snaps[i] = Snapshot{ID: "p", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true}
	}
	res := &ListResult{Snapshots: snaps, Total: 11, Size: 10, Offset: 0}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{}, "en")

	pag := findByID(sec, "snapshots-pagination")
	require.NotNil(t, pag)

	prev := findByID(*pag, "pagination-prev")
	require.NotNil(t, prev)
	assert.Equal(t, true, prev.Props["disabled"])

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	assert.NotEqual(t, true, next.Props["disabled"])
}

func TestBuildSnapshotsSection_Pagination_NextDisabled_AtLastPage(t *testing.T) {
	snaps := make([]Snapshot, 5)
	for i := range snaps {
		snaps[i] = Snapshot{ID: "p", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true}
	}
	res := &ListResult{Snapshots: snaps, Total: 25, Size: 10, Offset: 20}
	sec := BuildSnapshotsSection(res, sampleCatalog(), ListParams{Offset: 20}, "en")

	pag := findByID(sec, "snapshots-pagination")
	require.NotNil(t, pag)

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	assert.Equal(t, true, next.Props["disabled"])
}

func TestBuildSnapshotsSection_Pagination_ButtonURLsCarryFilterAndOffset(t *testing.T) {
	snaps := make([]Snapshot, 10)
	for i := range snaps {
		snaps[i] = Snapshot{ID: "p", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true}
	}
	res := &ListResult{Snapshots: snaps, Total: 25, Size: 10, Offset: 10}
	p := ListParams{IsFullSnapshot: boolPtr(true), Offset: 10}
	sec := BuildSnapshotsSection(res, sampleCatalog(), p, "en")

	pag := findByID(sec, "snapshots-pagination")
	require.NotNil(t, pag)

	prev := findByID(*pag, "pagination-prev")
	require.NotNil(t, prev)
	prevURL := prev.Actions[0].Endpoint
	assert.Contains(t, prevURL, "offset=0")
	assert.Contains(t, prevURL, "is_full_snapshot=true")

	next := findByID(*pag, "pagination-next")
	require.NotNil(t, next)
	nextURL := next.Actions[0].Endpoint
	assert.Contains(t, nextURL, "offset=20")
	assert.Contains(t, nextURL, "is_full_snapshot=true")
}
