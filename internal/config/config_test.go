package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_RequiresJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_DefaultsJWTLeewayTo30(t *testing.T) {
	t.Setenv("JWT_SECRET", "s3cret")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.JWTLeewaySeconds)
	assert.Equal(t, "s3cret", cfg.JWTSecret)
}
