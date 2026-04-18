package assets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCreateModal_Shape(t *testing.T) {
	m := BuildCreateModal(ListParams{AssetType: "STOCK", Offset: 10}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, "assets-create-modal", m.ID)
	assert.Equal(t, true, m.Props["visible"])
	assert.Equal(t, "Create Asset", m.Props["title"])
	assert.Equal(t, "dialog", m.Props["presentation"])

	form := findByID(m, "assets-create-form")
	require.NotNil(t, form)

	ticker := findByID(m, "create-ticker")
	require.NotNil(t, ticker)
	assert.Equal(t, "input", ticker.Type)
	assert.Equal(t, "ticker", ticker.Props["name"])
	assert.Equal(t, true, ticker.Props["required"])
	assert.Equal(t, 20, ticker.Props["max_length"])
	assert.Equal(t, `^[A-Z0-9.\-]+$`, ticker.Props["pattern"])
	assert.Equal(t, true, ticker.Props["auto_uppercase"])

	pp := findByID(m, "create-price-provider")
	require.NotNil(t, pp)
	vw, ok := pp.Props["visible_when"].(VisibleWhenValue)
	require.True(t, ok, "visible_when must be set on price_provider")
	assert.Equal(t, "is_complex", vw.Field)
	assert.Equal(t, "eq", vw.Op)
	assert.Equal(t, false, vw.Value)

	ext := findByID(m, "create-external-ticker")
	require.NotNil(t, ext)
	vw2, ok := ext.Props["visible_when"].(VisibleWhenValue)
	require.True(t, ok)
	assert.Equal(t, "price_provider", vw2.Field)
	assert.Equal(t, "ne", vw2.Op)
	assert.Equal(t, "", vw2.Value)

	submit := findByID(m, "create-submit")
	require.NotNil(t, submit)
	require.Len(t, submit.Actions, 1)
	act := submit.Actions[0]
	assert.Equal(t, "submit", act.Type)
	assert.Equal(t, "POST", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/assets/create")
	assert.Contains(t, act.Endpoint, "asset_type=STOCK")
	assert.Contains(t, act.Endpoint, "offset=10")
	assert.Equal(t, "assets-create-form", act.TargetID)
}

func TestBuildCreateModal_WithError(t *testing.T) {
	m := BuildCreateModal(ListParams{}, "en", "Ticker already registered")
	err := findByID(m, "modal-error")
	require.NotNil(t, err)
	assert.Equal(t, "Ticker already registered", err.Props["content"])
	assert.Equal(t, "negative", err.Props["color"])
}

func TestBuildEditModal_ImmutableFieldsAsText(t *testing.T) {
	provider := "TWELVE_DATA"
	ext := "AAPL"
	a := &Asset{
		ID: "a1", Ticker: "AAPL", Name: "Apple", AssetType: "STOCK",
		Currency: "USD", IsComplex: false,
		PriceProvider: &provider, ExternalTicker: &ext,
	}
	m := BuildEditModal(a, ListParams{}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, "assets-edit-modal", m.ID)
	assert.Equal(t, "Edit AAPL", m.Props["title"])

	// Immutable fields as text, not input
	tickerStatic := findByID(m, "edit-ticker-static")
	require.NotNil(t, tickerStatic)
	assert.Equal(t, "text", tickerStatic.Type)

	// Mutable name as input
	nameInput := findByID(m, "edit-name")
	require.NotNil(t, nameInput)
	assert.Equal(t, "input", nameInput.Type)
	assert.Equal(t, "Apple", nameInput.Props["default_value"])

	submit := findByID(m, "edit-submit")
	require.NotNil(t, submit)
	act := submit.Actions[0]
	assert.Equal(t, "PATCH", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/assets/a1")
}

func TestBuildDeleteModal_Shape(t *testing.T) {
	m := BuildDeleteModal("a1", "AAPL", ListParams{AssetType: "STOCK", Offset: 0}, "en", "")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, "assets-delete-modal", m.ID)
	assert.Equal(t, "Delete Asset", m.Props["title"])

	msg := findByID(m, "delete-message")
	require.NotNil(t, msg)
	assert.Equal(t, "Delete AAPL? This cannot be undone.", msg.Props["content"])

	force := findByID(m, "delete-force")
	require.NotNil(t, force)
	assert.Equal(t, "checkbox", force.Type)
	assert.Equal(t, "force", force.Props["name"])

	submit := findByID(m, "delete-submit")
	require.NotNil(t, submit)
	act := submit.Actions[0]
	assert.Equal(t, "DELETE", act.Method)
	assert.Contains(t, act.Endpoint, "/actions/assets/a1")
}
