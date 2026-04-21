package snapshots

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func newGinCtxWithBody(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

// --- parseJSONBody ---

// 9. empty body → empty map, nil error
func TestParseJSONBody_EmptyBody(t *testing.T) {
	c := newGinCtxWithBody("")
	m, err := parseJSONBody(c)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, m)
}

// 10. invalid JSON → nil, error
func TestParseJSONBody_InvalidJSON(t *testing.T) {
	c := newGinCtxWithBody("not json")
	m, err := parseJSONBody(c)
	assert.Nil(t, m)
	assert.Error(t, err)
}

// 11. non-object JSON (array) → nil or empty map (match trades behavior: json.Unmarshal into map[string]any returns error)
func TestParseJSONBody_ArrayJSON(t *testing.T) {
	c := newGinCtxWithBody(`[]`)
	m, err := parseJSONBody(c)
	// trades: json.Unmarshal([]byte("[]"), &map[string]any{}) returns error
	assert.Nil(t, m)
	assert.Error(t, err)
}

// 11b. non-object JSON (string literal) → error
func TestParseJSONBody_StringJSON(t *testing.T) {
	c := newGinCtxWithBody(`"hello"`)
	m, err := parseJSONBody(c)
	assert.Nil(t, m)
	assert.Error(t, err)
}

// --- asString ---

// 12. present non-string → ""
func TestAsString_PresentNonString(t *testing.T) {
	m := map[string]any{"key": 42}
	assert.Equal(t, "", asString(m, "key"))
}

// 13. missing key → ""
func TestAsString_MissingKey(t *testing.T) {
	m := map[string]any{}
	assert.Equal(t, "", asString(m, "key"))
}

// 14. present string → value
func TestAsString_PresentString(t *testing.T) {
	m := map[string]any{"key": "hello"}
	assert.Equal(t, "hello", asString(m, "key"))
}

// --- parseWizardEntries ---

// 1. no entries → empty (non-nil) slice
func TestParseWizardEntries_NoEntries(t *testing.T) {
	result := parseWizardEntries(map[string]any{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// 2. single complex asset — override only, no mode, no current_price
func TestParseWizardEntries_SingleComplexAsset(t *testing.T) {
	body := map[string]any{
		"entries[a1].current_value_override": "1000",
	}
	result := parseWizardEntries(body)
	require.Len(t, result, 1)
	assert.Equal(t, wizardEntry{
		AssetID:              "a1",
		Mode:                 "",
		CurrentPrice:         "",
		CurrentValueOverride: "1000",
	}, result[0])
}

// 3. single non-complex asset with mode=price
func TestParseWizardEntries_ModePriceAsset(t *testing.T) {
	body := map[string]any{
		"entries[a1].mode":          "price",
		"entries[a1].current_price": "150",
	}
	result := parseWizardEntries(body)
	require.Len(t, result, 1)
	assert.Equal(t, wizardEntry{
		AssetID:              "a1",
		Mode:                 "price",
		CurrentPrice:         "150",
		CurrentValueOverride: "",
	}, result[0])
}

// 4. single non-complex asset with mode=override
func TestParseWizardEntries_ModeOverrideAsset(t *testing.T) {
	body := map[string]any{
		"entries[a1].mode":                  "override",
		"entries[a1].current_value_override": "1000",
	}
	result := parseWizardEntries(body)
	require.Len(t, result, 1)
	assert.Equal(t, wizardEntry{
		AssetID:              "a1",
		Mode:                 "override",
		CurrentPrice:         "",
		CurrentValueOverride: "1000",
	}, result[0])
}

// 5. mix — two assets, one complex (override-only) and one non-complex (price)
func TestParseWizardEntries_MixTwoAssets(t *testing.T) {
	body := map[string]any{
		"entries[a1].current_value_override": "2000",
		"entries[a2].mode":                  "price",
		"entries[a2].current_price":          "150",
	}
	result := parseWizardEntries(body)
	require.Len(t, result, 2)
	// sorted by AssetID ascending: a1, a2
	assert.Equal(t, "a1", result[0].AssetID)
	assert.Equal(t, "2000", result[0].CurrentValueOverride)
	assert.Equal(t, "", result[0].Mode)
	assert.Equal(t, "a2", result[1].AssetID)
	assert.Equal(t, "price", result[1].Mode)
	assert.Equal(t, "150", result[1].CurrentPrice)
}

// 6. malformed keys — silently dropped, no panic
func TestParseWizardEntries_MalformedKeys(t *testing.T) {
	body := map[string]any{
		"entries[].mode":    "price",  // empty asset_id — regex requires [^\]]+ (one or more)
		"entries.a1.mode":  "price",  // wrong bracket notation
		"recorded_at":      "2024-01-10T10:00:00Z",
	}
	result := parseWizardEntries(body)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// 7. same asset_id with all three fields
func TestParseWizardEntries_AllThreeFields(t *testing.T) {
	body := map[string]any{
		"entries[a1].mode":                  "price",
		"entries[a1].current_price":          "150",
		"entries[a1].current_value_override": "1000",
	}
	result := parseWizardEntries(body)
	require.Len(t, result, 1)
	assert.Equal(t, wizardEntry{
		AssetID:              "a1",
		Mode:                 "price",
		CurrentPrice:         "150",
		CurrentValueOverride: "1000",
	}, result[0])
}

// 8. non-string value — silently ignored via asString coercion
func TestParseWizardEntries_NonStringValue(t *testing.T) {
	body := map[string]any{
		"entries[a1].mode": 42,
	}
	result := parseWizardEntries(body)
	// The entry is keyed by asset_id but mode is "" (non-string → ""),
	// and no other fields set — no non-empty fields → still produces entry
	// but with all empty fields. Handler is responsible for filtering.
	// parseWizardEntries itself still creates the entry (the key matched the regex).
	require.Len(t, result, 1)
	assert.Equal(t, wizardEntry{
		AssetID:              "a1",
		Mode:                 "",
		CurrentPrice:         "",
		CurrentValueOverride: "",
	}, result[0])
}

// ordering: multiple assets returned in ascending AssetID order
func TestParseWizardEntries_Ordering(t *testing.T) {
	body := map[string]any{
		"entries[z9].mode":  "price",
		"entries[a1].mode":  "override",
		"entries[m5].mode":  "price",
	}
	result := parseWizardEntries(body)
	require.Len(t, result, 3)
	assert.Equal(t, "a1", result[0].AssetID)
	assert.Equal(t, "m5", result[1].AssetID)
	assert.Equal(t, "z9", result[2].AssetID)
}
