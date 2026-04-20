package trades

import (
	"context"
	"errors"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTradeFetcher struct {
	res     *ListResult
	err     error
	calls   int
	gotAuth string
	gotP    ListParams
}

func (f *fakeTradeFetcher) List(_ context.Context, auth string, p ListParams) (*ListResult, error) {
	f.calls++
	f.gotAuth = auth
	f.gotP = p
	return f.res, f.err
}

type fakeCatalogFetcher struct {
	res     []assetscatalog.Asset
	err     error
	calls   int
	gotAuth string
}

func (f *fakeCatalogFetcher) List(_ context.Context, auth string) ([]assetscatalog.Asset, error) {
	f.calls++
	f.gotAuth = auth
	return f.res, f.err
}

// findChild returns true if any node in the tree (including root) has the given ID.
func findChild(c components.Component, id string) bool {
	if c.ID == id {
		return true
	}
	for _, ch := range c.Children {
		if findChild(ch, id) {
			return true
		}
	}
	return false
}

func TestGetUseCase_Execute_HappyPath(t *testing.T) {
	tf := &fakeTradeFetcher{res: &ListResult{Trades: []Trade{{ID: "t1", AssetID: "a1", TradeType: "BUY", Quantity: "1", PricePerUnit: "100", Date: "2025-01-01T00:00:00Z"}}, Total: 1, Size: 10}}
	cf := &fakeCatalogFetcher{res: []assetscatalog.Asset{{ID: "a1", Ticker: "AAPL", Currency: "USD"}}}
	uc := NewGetUseCase(tf, cf)

	tree, err := uc.Execute(context.Background(), "Bearer x", ListParams{AssetID: "a1"}, "en")
	require.NoError(t, err)
	assert.Equal(t, ScreenID, tree.ID)
	assert.True(t, findChild(tree, ModalSlotID), "screen tree should contain ModalSlotID")
	assert.Equal(t, "Bearer x", tf.gotAuth)
	assert.Equal(t, "Bearer x", cf.gotAuth)
	assert.Equal(t, "a1", tf.gotP.AssetID)
	assert.Equal(t, 1, tf.calls)
	assert.Equal(t, 1, cf.calls)
}

func TestGetUseCase_ExecuteSection_HappyPath(t *testing.T) {
	tf := &fakeTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &fakeCatalogFetcher{res: []assetscatalog.Asset{{ID: "a1", Ticker: "AAPL"}}}
	uc := NewGetUseCase(tf, cf)

	section, err := uc.ExecuteSection(context.Background(), "Bearer x", ListParams{}, "en")
	require.NoError(t, err)
	assert.Equal(t, SectionID, section.ID)
	assert.Equal(t, 1, tf.calls)
	assert.Equal(t, 1, cf.calls)
}

func TestGetUseCase_Execute_TradesUnauthorized_ShortCircuitsCatalog(t *testing.T) {
	tf := &fakeTradeFetcher{err: ErrUnauthorized}
	cf := &fakeCatalogFetcher{res: []assetscatalog.Asset{{ID: "a1", Ticker: "AAPL"}}}
	uc := NewGetUseCase(tf, cf)

	tree, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
	assert.Equal(t, components.Component{}, tree)
	assert.Equal(t, 0, cf.calls, "catalog must not be called when trades fail")
}

func TestGetUseCase_Execute_TradesBackendError(t *testing.T) {
	tf := &fakeTradeFetcher{err: ErrBackend}
	cf := &fakeCatalogFetcher{res: []assetscatalog.Asset{{ID: "a1", Ticker: "AAPL"}}}
	uc := NewGetUseCase(tf, cf)

	_, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackend))
	assert.Equal(t, 0, cf.calls, "catalog must not be called when trades fail")
}

func TestGetUseCase_Execute_CatalogUnauthorized(t *testing.T) {
	tf := &fakeTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &fakeCatalogFetcher{err: assetscatalog.ErrUnauthorized}
	uc := NewGetUseCase(tf, cf)

	tree, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetscatalog.ErrUnauthorized))
	assert.Equal(t, components.Component{}, tree)
}

func TestGetUseCase_Execute_CatalogBackendError(t *testing.T) {
	tf := &fakeTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &fakeCatalogFetcher{err: assetscatalog.ErrBackend}
	uc := NewGetUseCase(tf, cf)

	_, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetscatalog.ErrBackend))
}

func TestGetUseCase_ExecuteSection_TradesError_ShortCircuitsCatalog(t *testing.T) {
	tf := &fakeTradeFetcher{err: ErrUnauthorized}
	cf := &fakeCatalogFetcher{res: []assetscatalog.Asset{}}
	uc := NewGetUseCase(tf, cf)

	section, err := uc.ExecuteSection(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
	assert.Equal(t, components.Component{}, section)
	assert.Equal(t, 0, cf.calls)
}

func TestGetUseCase_ExecuteSection_CatalogError(t *testing.T) {
	tf := &fakeTradeFetcher{res: &ListResult{Trades: []Trade{}, Total: 0, Size: 10}}
	cf := &fakeCatalogFetcher{err: assetscatalog.ErrBackend}
	uc := NewGetUseCase(tf, cf)

	_, err := uc.ExecuteSection(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetscatalog.ErrBackend))
}
