package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputAdvanced_RequiredFieldsOnly(t *testing.T) {
	c := InputAdvanced(InputOptions{ID: "p", Name: "p", InputType: "password"})

	assert.Equal(t, "input", c.Type)
	assert.Equal(t, "p", c.ID)
	assert.Equal(t, "p", c.Props["name"])
	assert.Equal(t, "password", c.Props["input_type"])
	_, hasMin := c.Props["min_length"]
	_, hasMatch := c.Props["match_field"]
	assert.False(t, hasMin, "min_length must be omitted when zero")
	assert.False(t, hasMatch, "match_field must be omitted when empty")
}

func TestInputAdvanced_MinLengthAndMatchField(t *testing.T) {
	c := InputAdvanced(InputOptions{
		ID: "confirm", Name: "confirm_password", InputType: "password",
		Required: true, MinLength: 8, MatchField: "password",
	})

	assert.Equal(t, true, c.Props["required"])
	assert.Equal(t, 8, c.Props["min_length"])
	assert.Equal(t, "password", c.Props["match_field"])
}

func TestInputFull_StillWorksUnchanged(t *testing.T) {
	c := InputFull("e", "email", "email", "Email", "you@example.com", "", true, false, 0)
	assert.Equal(t, "input", c.Type)
	assert.Equal(t, "email", c.Props["name"])
	assert.Equal(t, true, c.Props["required"])
	_, hasMin := c.Props["min_length"]
	assert.False(t, hasMin)
}
