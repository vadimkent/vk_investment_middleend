package home

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUseCase_Execute(t *testing.T) {
	client := NewClient("http://localhost:8080")
	uc := NewGetUseCase(client)

	screen, err := uc.Execute(context.Background(), "en", "web")
	require.NoError(t, err)
	assert.Equal(t, "screen", screen.Type)
	assert.Equal(t, "home", screen.ID)
}
