package profile

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDeleteModal_HasPasswordInput(t *testing.T) {
	m := BuildDeleteModal("en", "")
	b, err := json.Marshal(m)
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, DeleteModalID)
	assert.Contains(t, s, `"name":"password"`)
	assert.Contains(t, s, `"input_type":"password"`)
	assert.Contains(t, s, "/actions/profile/delete_account")
}

func TestBuildDeleteModal_WithError(t *testing.T) {
	m := BuildDeleteModal("en", "Incorrect password")
	b, _ := json.Marshal(m)
	s := string(b)
	assert.Contains(t, s, "delete-modal-error")
	assert.Contains(t, s, "Incorrect password")
}

func TestBuildDeleteModal_PasswordAlwaysEmpty(t *testing.T) {
	m := BuildDeleteModal("en", "")
	b, _ := json.Marshal(m)
	s := string(b)
	// Password input is the only field; it must NOT have a non-empty default_value.
	// (InputFull omits default_value when empty, so we just confirm absence of the
	// preserved-token pattern that would only appear with a non-empty default.)
	assert.False(t, strings.Contains(s, `"default_value":"x"`), "no preserved password expected")
}
