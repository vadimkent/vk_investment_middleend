package profile

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/i18n"
)

func TestMain(m *testing.M) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = i18n.Load(filepath.Join(repoRoot, "locales"))
	m.Run()
}

func ptr(s string) *string { return &s }

func sampleUser() *User {
	return &User{
		ID:          "u1",
		Email:       "vadim@example.com",
		DisplayName: ptr("Vadim"),
		Preferences: Preferences{DefaultCurrency: ptr("USD")},
		CreatedAt:   "2026-01-01T00:00:00Z",
	}
}

func sampleConfig() *AppConfig {
	return &AppConfig{Currencies: []string{"USD", "EUR", "ARS"}}
}

func asJSON(t *testing.T, c any) map[string]any {
	t.Helper()
	b, err := json.Marshal(c)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestBuildScreen_RootShape(t *testing.T) {
	tree := BuildScreen(sampleUser(), sampleConfig(), "en")
	m := asJSON(t, tree)
	assert.Equal(t, "screen", m["type"])
	assert.Equal(t, ScreenID, m["id"])
	props := m["props"].(map[string]any)
	assert.Equal(t, "Profile", props["title"])
	body, _ := json.Marshal(tree)
	bodyStr := string(body)
	for _, id := range []string{ProfileCardID, EmailCardID, PasswordCardID, DangerCardID, ModalSlotID} {
		assert.Contains(t, bodyStr, id)
	}
}

func TestBuildProfileCard_DefaultsFromUser(t *testing.T) {
	c := BuildProfileCard(sampleUser(), sampleConfig(), "en", "")
	body, _ := json.Marshal(c)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"default_value":"Vadim"`)
	assert.Contains(t, bodyStr, `"default_value":"USD"`)
	assert.NotContains(t, bodyStr, "profile-card-error")
}

func TestBuildProfileCard_WithError(t *testing.T) {
	c := BuildProfileCard(sampleUser(), sampleConfig(), "en", "Display name must be between 1 and 100 characters")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), "profile-card-error")
	assert.Contains(t, string(bodyStr), "must be between 1 and 100")
}

func TestBuildProfileCard_NoneOptionLabelInSpanish(t *testing.T) {
	c := BuildProfileCard(sampleUser(), sampleConfig(), "es", "")
	body, _ := json.Marshal(c)
	assert.Contains(t, string(body), "Ninguna")
}

func TestBuildEmailCard_CurrentEmailInterpolated(t *testing.T) {
	c := BuildEmailCard(sampleUser(), "en", "", "")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), "vadim@example.com")
	assert.False(t, strings.Contains(string(bodyStr), "{email}"), "interpolation token leaked")
}

func TestBuildEmailCard_PreservesNewEmail(t *testing.T) {
	c := BuildEmailCard(sampleUser(), "en", "preserved@x.y", "wrong password")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), `"default_value":"preserved@x.y"`)
	assert.Contains(t, string(bodyStr), "email-card-error")
}

func TestBuildPasswordCard_AlwaysEmpty(t *testing.T) {
	c := BuildPasswordCard("en", "")
	bodyStr, _ := json.Marshal(c)
	s := string(bodyStr)
	// InputFull omits default_value when empty, so none should appear.
	assert.NotContains(t, s, `"default_value"`)
	// All three password field names are present.
	assert.GreaterOrEqual(t, strings.Count(s, `"input_type":"password"`), 3)
}

func TestBuildDangerCard_HasDestructiveButton(t *testing.T) {
	c := BuildDangerCard("en")
	bodyStr, _ := json.Marshal(c)
	assert.Contains(t, string(bodyStr), `"variant":"destructive"`)
	assert.Contains(t, string(bodyStr), "/actions/profile/delete_modal")
}
