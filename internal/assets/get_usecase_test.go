package assets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	res     *ListResult
	err     error
	gotAuth string
	gotP    ListParams
}

func (f *fakeClient) List(_ context.Context, auth string, p ListParams) (*ListResult, error) {
	f.gotAuth = auth
	f.gotP = p
	return f.res, f.err
}

func TestUseCase_Execute_HappyPath(t *testing.T) {
	fc := &fakeClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	uc := NewGetUseCase(fc)

	tree, err := uc.Execute(context.Background(), "Bearer x", ListParams{AssetType: "STOCK", Offset: 0}, "en")
	require.NoError(t, err)
	assert.Equal(t, "screen", tree.Type)
	assert.Equal(t, "assets", tree.ID)
	assert.Equal(t, "Bearer x", fc.gotAuth)
	assert.Equal(t, "STOCK", fc.gotP.AssetType)
}

func TestUseCase_Execute_UnauthorizedPropagates(t *testing.T) {
	fc := &fakeClient{err: ErrUnauthorized}
	uc := NewGetUseCase(fc)

	_, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestUseCase_ExecuteSection_ReturnsSubtree(t *testing.T) {
	fc := &fakeClient{res: &ListResult{Assets: []Asset{{ID: "a1", Ticker: "AAPL"}}, Total: 1, Size: 10}}
	uc := NewGetUseCase(fc)

	section, err := uc.ExecuteSection(context.Background(), "Bearer x", ListParams{}, "en")
	require.NoError(t, err)
	assert.Equal(t, "column", section.Type)
	assert.Equal(t, "assets-section", section.ID)
}
