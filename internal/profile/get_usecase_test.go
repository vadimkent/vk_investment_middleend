package profile

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMe struct {
	res     *User
	err     error
	calls   int
	gotAuth string
}

func (s *stubMe) GetMe(_ context.Context, auth string) (*User, error) {
	s.calls++
	s.gotAuth = auth
	return s.res, s.err
}

type stubCfg struct {
	res     *AppConfig
	err     error
	calls   int
	gotAuth string
}

func (s *stubCfg) GetConfig(_ context.Context, auth string) (*AppConfig, error) {
	s.calls++
	s.gotAuth = auth
	return s.res, s.err
}

func TestGetUseCase_Happy(t *testing.T) {
	m := &stubMe{res: sampleUser()}
	cfg := &stubCfg{res: sampleConfig()}
	uc := NewGetUseCase(m, cfg)
	tree, err := uc.Execute(context.Background(), "Bearer t", "en")
	require.NoError(t, err)
	assert.Equal(t, "Bearer t", m.gotAuth)
	assert.Equal(t, "Bearer t", cfg.gotAuth)
	assert.Equal(t, "screen", asJSON(t, tree)["type"])
}

func TestGetUseCase_MeUnauthorized_ShortCircuits(t *testing.T) {
	m := &stubMe{err: ErrUnauthorized}
	cfg := &stubCfg{}
	uc := NewGetUseCase(m, cfg)
	_, err := uc.Execute(context.Background(), "", "en")
	assert.True(t, errors.Is(err, ErrUnauthorized))
	assert.Equal(t, 0, cfg.calls, "config should not be called after me failed")
}

func TestGetUseCase_ConfigError(t *testing.T) {
	m := &stubMe{res: sampleUser()}
	cfg := &stubCfg{err: ErrBackend}
	uc := NewGetUseCase(m, cfg)
	_, err := uc.Execute(context.Background(), "", "en")
	assert.True(t, errors.Is(err, ErrBackend))
}
