package snapshots

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSnapshot returns a minimal Snapshot suitable for modal tests.
func testSnapshot() *Snapshot {
	return &Snapshot{
		ID:             "snap-abc",
		RecordedAt:     "2024-01-10T09:30:00Z",
		IsFullSnapshot: true,
	}
}

// Test 1: modal type, id, presentation, visible.
func TestBuildDeleteModal_TypeAndPresentation(t *testing.T) {
	s := testSnapshot()
	m := BuildDeleteModal(s, ListParams{}, "en")

	assert.Equal(t, "modal", m.Type)
	assert.Equal(t, DeleteModalID, m.ID)
	assert.Equal(t, "dialog", m.Props["presentation"])
	assert.Equal(t, true, m.Props["visible"])
}

// Test 2: title is the localised snapshots.delete.title string.
func TestBuildDeleteModal_Title(t *testing.T) {
	s := testSnapshot()
	m := BuildDeleteModal(s, ListParams{}, "en")

	title, ok := m.Props["title"].(string)
	require.True(t, ok, "expected title prop to be a string")
	// i18n key "snapshots.delete.title" resolves to the registered translation.
	assert.NotEmpty(t, title)
	assert.NotEqual(t, "snapshots.delete.title", title, "title must resolve to a translated string, not the raw key")
}

// Test 3: body text contains the formatted date from RecordedAt.
func TestBuildDeleteModal_BodyContainsDate(t *testing.T) {
	// RecordedAt "2024-01-10T09:30:00Z" → formatted date "2024-01-10"
	s := testSnapshot()
	m := BuildDeleteModal(s, ListParams{}, "en")

	messageNode := findByID(m, "snapshots-delete-message")
	require.NotNil(t, messageNode, "expected a node with id snapshots-delete-message")

	content, ok := messageNode.Props["content"].(string)
	require.True(t, ok, "expected content prop to be a string")
	assert.True(t, strings.Contains(content, "2024-01-10"),
		"expected body text to contain the formatted date '2024-01-10', got: %q", content)
}

// Test 4: cancel button has a replace action targeting ModalSlotID with an empty tree.
func TestBuildDeleteModal_CancelButton(t *testing.T) {
	s := testSnapshot()
	m := BuildDeleteModal(s, ListParams{}, "en")

	cancel := findByID(m, "snapshots-delete-cancel")
	require.NotNil(t, cancel, "expected a node with id snapshots-delete-cancel")

	require.Len(t, cancel.Actions, 1, "cancel button must have exactly one action")
	act := cancel.Actions[0]
	assert.Equal(t, "click", act.Trigger)
	assert.Equal(t, "replace", act.Type)
	assert.Equal(t, ModalSlotID, act.TargetID)
	assert.Empty(t, act.Endpoint, "dismiss replace must have no endpoint (empty tree)")
}

// Test 5: delete button has a submit action with method DELETE, correct endpoint, target ScreenID.
func TestBuildDeleteModal_DeleteButton(t *testing.T) {
	s := testSnapshot()
	isFull := false
	p := ListParams{IsFullSnapshot: &isFull, Offset: 10}
	m := BuildDeleteModal(s, p, "en")

	del := findByID(m, "snapshots-delete-submit")
	require.NotNil(t, del, "expected a node with id snapshots-delete-submit")

	require.Len(t, del.Actions, 1, "delete button must have exactly one action")
	act := del.Actions[0]
	assert.Equal(t, "click", act.Trigger)
	assert.Equal(t, "submit", act.Type)
	assert.Equal(t, "DELETE", act.Method)
	assert.Equal(t, ScreenID, act.TargetID)

	assert.Contains(t, act.Endpoint, "/actions/snapshots/snap-abc")
	assert.Contains(t, act.Endpoint, "is_full_snapshot=false")
	assert.Contains(t, act.Endpoint, "offset=10")
}
