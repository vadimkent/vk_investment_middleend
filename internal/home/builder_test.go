package home

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildScreen_Web(t *testing.T) {
	screen := BuildScreen("en", "web")

	assert.Equal(t, "screen", screen.Type)
	assert.Equal(t, "home", screen.ID)
	assert.NotEmpty(t, screen.Children)
}

func TestBuildScreen_Mobile(t *testing.T) {
	screen := BuildScreen("en", "android")

	assert.Equal(t, "screen", screen.Type)
	assert.Equal(t, "home", screen.ID)
	assert.NotEmpty(t, screen.Children)
}
