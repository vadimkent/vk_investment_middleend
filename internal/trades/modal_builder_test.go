package trades

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// findInput walks the tree looking for an input/select/textarea node with the
// given name prop. Returns nil when none is found.
func findFormFieldByName(c components.Component, name string) *components.Component {
	if c.Type == "input" || c.Type == "select" || c.Type == "textarea" {
		if got, ok := c.Props["name"].(string); ok && got == name {
			return &c
		}
	}
	for i := range c.Children {
		if got := findFormFieldByName(c.Children[i], name); got != nil {
			return got
		}
	}
	return nil
}

func modalCatalog() []assetscatalog.Asset {
	return []assetscatalog.Asset{
		{ID: "aaa-1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK", Currency: "USD", IsComplex: false},
		{ID: "ccc-3", Ticker: "CMPX", Name: "Complex Thing", AssetType: "ETF", Currency: "USD", IsComplex: true},
	}
}

func sampleTrade() Trade {
	return Trade{
		ID:           "trade-123",
		AssetID:      "aaa-1",
		TradeType:    "BUY",
		Quantity:     "10.5",
		PricePerUnit: "100.50",
		Fees:         "1.25",
		Date:         "2024-03-15T00:00:00Z",
		Source:       "MANUAL",
		Notes:        "Some notes",
	}
}

// --- Create modal ---

func TestBuildCreateModal_AllFieldsPresent(t *testing.T) {
	m := BuildCreateModal(modalCatalog(), ListParams{}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, ModalID, m.ID)

	// Form exists.
	form := findByID(m, "trades-create-form")
	require.NotNil(t, form)
	assert.Equal(t, "form", form.Type)

	// Seven form fields by name.
	names := []string{"asset_id", "trade_type", "quantity", "price_per_unit", "fees", "date", "notes"}
	for _, n := range names {
		f := findFormFieldByName(m, n)
		require.NotNil(t, f, "missing form field with name=%s", n)
	}

	// Asset select.
	asset := findFormFieldByName(m, "asset_id")
	require.NotNil(t, asset)
	assert.Equal(t, "select", asset.Type)
	assert.Equal(t, true, asset.Props["required"])

	// Trade type required, no Any option.
	tradeType := findFormFieldByName(m, "trade_type")
	require.NotNil(t, tradeType)
	assert.Equal(t, "select", tradeType.Type)
	assert.Equal(t, true, tradeType.Props["required"])

	// Quantity required text input.
	qty := findFormFieldByName(m, "quantity")
	require.NotNil(t, qty)
	assert.Equal(t, "input", qty.Type)
	assert.Equal(t, "text", qty.Props["input_type"])
	assert.Equal(t, true, qty.Props["required"])

	// Price per unit required.
	price := findFormFieldByName(m, "price_per_unit")
	require.NotNil(t, price)
	assert.Equal(t, true, price.Props["required"])

	// Fees optional with default "0".
	fees := findFormFieldByName(m, "fees")
	require.NotNil(t, fees)
	assert.NotEqual(t, true, fees.Props["required"])
	assert.Equal(t, "0", fees.Props["default_value"])

	// Date required, input_type=date, max = today.
	date := findFormFieldByName(m, "date")
	require.NotNil(t, date)
	assert.Equal(t, "date", date.Props["input_type"])
	assert.Equal(t, true, date.Props["required"])
	today := time.Now().UTC().Format("2006-01-02")
	assert.Equal(t, today, date.Props["max"])

	// Notes textarea, max_length 500.
	notes := findFormFieldByName(m, "notes")
	require.NotNil(t, notes)
	assert.Equal(t, "textarea", notes.Type)
	assert.Equal(t, 500, notes.Props["max_length"])
}

func TestBuildCreateModal_AssetOptionsExcludeComplex(t *testing.T) {
	m := BuildCreateModal(modalCatalog(), ListParams{}, "en", "")

	asset := findFormFieldByName(m, "asset_id")
	require.NotNil(t, asset)

	opts, ok := asset.Props["options"].([]components.SelectOption)
	require.True(t, ok, "options must be []components.SelectOption")
	// Expect placeholder empty + AAPL only (CMPX excluded).
	require.Len(t, opts, 2)
	assert.Equal(t, "", opts[0].Value)
	assert.Equal(t, "aaa-1", opts[1].Value)
	assert.Equal(t, "AAPL", opts[1].Label)
}

func TestBuildCreateModal_EmptyCatalogDisablesSubmit(t *testing.T) {
	catalog := []assetscatalog.Asset{
		{ID: "ccc-3", Ticker: "CMPX", IsComplex: true},
	}
	m := BuildCreateModal(catalog, ListParams{}, "en", "")

	submit := findByID(m, "trades-create-submit")
	require.NotNil(t, submit)
	assert.Equal(t, true, submit.Props["disabled"])

	hint := findByID(m, "trades-create-no-assets-hint")
	require.NotNil(t, hint)
	assert.Equal(t, "text", hint.Type)
	assert.Equal(t, "Register an asset first to record trades.", hint.Props["content"])
}

func TestBuildCreateModal_InlineErrorRendered(t *testing.T) {
	m := BuildCreateModal(modalCatalog(), ListParams{}, "en", "Insufficient quantity")

	errNode := findByID(m, "trades-modal-error")
	require.NotNil(t, errNode)
	assert.Equal(t, "text", errNode.Type)
	assert.Equal(t, "Insufficient quantity", errNode.Props["content"])
	assert.Equal(t, "negative", errNode.Props["color"])
}

func TestBuildCreateModal_SubmitURLPreservesListContext(t *testing.T) {
	m := BuildCreateModal(modalCatalog(), ListParams{AssetID: "aaa-1", TradeType: "SELL", Offset: 10}, "en", "")

	submit := findByID(m, "trades-create-submit")
	require.NotNil(t, submit)
	require.Len(t, submit.Actions, 1)
	act := submit.Actions[0]
	assert.Equal(t, "POST", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/trades/create")
	assert.Contains(t, act.Endpoint, "asset_id=aaa-1")
	assert.Contains(t, act.Endpoint, "trade_type=SELL")
	assert.Contains(t, act.Endpoint, "offset=10")
}

// --- Edit modal ---

func TestBuildEditModal_StaticDateAndSource(t *testing.T) {
	tr := sampleTrade()
	m := BuildEditModal(tr, modalCatalog(), ListParams{}, "en", "")

	// `date` and `source` must NOT appear as form input nodes.
	assert.Nil(t, findFormFieldByName(m, "date"))
	assert.Nil(t, findFormFieldByName(m, "source"))

	// They must appear as static text nodes that mention the value.
	dateStatic := findByID(m, "trades-edit-date-static")
	require.NotNil(t, dateStatic)
	assert.Equal(t, "text", dateStatic.Type)
	assert.Contains(t, dateStatic.Props["content"].(string), "2024-03-15")

	sourceStatic := findByID(m, "trades-edit-source-static")
	require.NotNil(t, sourceStatic)
	assert.Equal(t, "text", sourceStatic.Type)
	assert.Contains(t, sourceStatic.Props["content"].(string), "MANUAL")
}

func TestBuildEditModal_MutableFieldsPrepopulated(t *testing.T) {
	tr := sampleTrade()
	m := BuildEditModal(tr, modalCatalog(), ListParams{}, "en", "")

	asset := findFormFieldByName(m, "asset_id")
	require.NotNil(t, asset)
	assert.Equal(t, "aaa-1", asset.Props["default_value"])

	tt := findFormFieldByName(m, "trade_type")
	require.NotNil(t, tt)
	assert.Equal(t, "BUY", tt.Props["default_value"])

	qty := findFormFieldByName(m, "quantity")
	require.NotNil(t, qty)
	assert.Equal(t, "10.5", qty.Props["default_value"])

	price := findFormFieldByName(m, "price_per_unit")
	require.NotNil(t, price)
	assert.Equal(t, "100.50", price.Props["default_value"])

	fees := findFormFieldByName(m, "fees")
	require.NotNil(t, fees)
	assert.Equal(t, "1.25", fees.Props["default_value"])

	notes := findFormFieldByName(m, "notes")
	require.NotNil(t, notes)
	assert.Equal(t, "Some notes", notes.Props["default_value"])
}

func TestBuildEditModal_FeesDefaultsToZeroWhenEmpty(t *testing.T) {
	tr := sampleTrade()
	tr.Fees = ""
	m := BuildEditModal(tr, modalCatalog(), ListParams{}, "en", "")

	fees := findFormFieldByName(m, "fees")
	require.NotNil(t, fees)
	assert.Equal(t, "0", fees.Props["default_value"])
}

func TestBuildEditModal_TitleInterpolation(t *testing.T) {
	tr := sampleTrade()
	m := BuildEditModal(tr, modalCatalog(), ListParams{}, "en", "")

	title, ok := m.Props["title"].(string)
	require.True(t, ok)
	assert.Contains(t, title, "2024-03-15")
	assert.Contains(t, title, "AAPL")
}

func TestBuildEditModal_TitleUsesUUIDFallbackWhenAssetMissing(t *testing.T) {
	tr := sampleTrade()
	tr.AssetID = "unknown-uuid"
	m := BuildEditModal(tr, modalCatalog(), ListParams{}, "en", "")

	title, ok := m.Props["title"].(string)
	require.True(t, ok)
	assert.Contains(t, title, "unknown-uuid")
}

func TestBuildEditModal_SubmitURLUsesTradeIDAndListContext(t *testing.T) {
	tr := sampleTrade()
	m := BuildEditModal(tr, modalCatalog(), ListParams{AssetID: "aaa-1", TradeType: "SELL", Offset: 10}, "en", "")

	submit := findByID(m, "trades-edit-submit")
	require.NotNil(t, submit)
	require.Len(t, submit.Actions, 1)
	act := submit.Actions[0]
	assert.Equal(t, "PATCH", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/trades/trade-123")
	assert.Contains(t, act.Endpoint, "asset_id=aaa-1")
	assert.Contains(t, act.Endpoint, "trade_type=SELL")
	assert.Contains(t, act.Endpoint, "offset=10")
}

func TestBuildEditModal_InlineErrorRendered(t *testing.T) {
	tr := sampleTrade()
	m := BuildEditModal(tr, modalCatalog(), ListParams{}, "en", "Invalid price")

	errNode := findByID(m, "trades-modal-error")
	require.NotNil(t, errNode)
	assert.Equal(t, "Invalid price", errNode.Props["content"])
	assert.Equal(t, "negative", errNode.Props["color"])
}

// --- Delete modal ---

func TestBuildDeleteModal_ConfirmInterpolation(t *testing.T) {
	tr := sampleTrade()
	m := BuildDeleteModal(tr, modalCatalog(), ListParams{}, "en", "")

	msg := findByID(m, "trades-delete-message")
	require.NotNil(t, msg)
	content, ok := msg.Props["content"].(string)
	require.True(t, ok)
	// Per spec: type, quantity, ticker, date.
	assert.Contains(t, content, "BUY")
	assert.Contains(t, content, "10.5")
	assert.Contains(t, content, "AAPL")
	assert.Contains(t, content, "2024-03-15")
}

func TestBuildDeleteModal_DestructiveVariant(t *testing.T) {
	tr := sampleTrade()
	m := BuildDeleteModal(tr, modalCatalog(), ListParams{}, "en", "")

	submit := findByID(m, "trades-delete-submit")
	require.NotNil(t, submit)
	assert.Equal(t, "destructive", submit.Props["variant"])
}

func TestBuildDeleteModal_SubmitURL(t *testing.T) {
	tr := sampleTrade()
	m := BuildDeleteModal(tr, modalCatalog(), ListParams{AssetID: "aaa-1", TradeType: "SELL", Offset: 20}, "en", "")

	submit := findByID(m, "trades-delete-submit")
	require.NotNil(t, submit)
	require.Len(t, submit.Actions, 1)
	act := submit.Actions[0]
	assert.Equal(t, "DELETE", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/trades/trade-123")
	assert.Contains(t, act.Endpoint, "asset_id=aaa-1")
	assert.Contains(t, act.Endpoint, "trade_type=SELL")
	assert.Contains(t, act.Endpoint, "offset=20")
}

func TestBuildDeleteModal_InlineErrorRendered(t *testing.T) {
	tr := sampleTrade()
	m := BuildDeleteModal(tr, modalCatalog(), ListParams{}, "en", "Cannot delete")

	errNode := findByID(m, "trades-modal-error")
	require.NotNil(t, errNode)
	assert.Equal(t, "Cannot delete", errNode.Props["content"])
}

// --- buildSubmitURL helper ---

func TestBuildSubmitURL_OmitsEmpty(t *testing.T) {
	got := buildSubmitURL("/actions/trades/create", ListParams{})
	assert.Equal(t, "/actions/trades/create", got)
}

func TestBuildSubmitURL_IncludesAllWhenSet(t *testing.T) {
	got := buildSubmitURL("/actions/trades/create", ListParams{AssetID: "aaa-1", TradeType: "BUY", Offset: 30})
	assert.True(t, strings.HasPrefix(got, "/actions/trades/create?"))
	assert.Contains(t, got, "asset_id=aaa-1")
	assert.Contains(t, got, "trade_type=BUY")
	assert.Contains(t, got, "offset=30")
}
