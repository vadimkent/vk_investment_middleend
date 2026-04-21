package snapshots

import (
	"context"
	"errors"
	"testing"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSnapshotFetcher struct {
	fn    func(context.Context, string, ListParams) (*ListResult, error)
	calls int
}

func (f *fakeSnapshotFetcher) List(ctx context.Context, authorization string, p ListParams) (*ListResult, error) {
	f.calls++
	return f.fn(ctx, authorization, p)
}

type fakeCatalog struct {
	fn    func(context.Context, string) ([]assetscatalog.Asset, error)
	calls int
}

func (f *fakeCatalog) List(ctx context.Context, authorization string) ([]assetscatalog.Asset, error) {
	f.calls++
	return f.fn(ctx, authorization)
}

func okResult() *ListResult {
	return &ListResult{
		Snapshots: []Snapshot{{ID: "s1", RecordedAt: "2025-01-01T00:00:00Z", IsFullSnapshot: true}},
		Total:     1,
		Size:      10,
	}
}

func okCatalog() []assetscatalog.Asset {
	return []assetscatalog.Asset{{ID: "a1", Ticker: "AAPL", Currency: "USD"}}
}

func TestGetUseCase_Execute_HappyPath(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return okResult(), nil
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return okCatalog(), nil
	}}
	uc := NewGetUseCase(sf, cf)

	tree, err := uc.Execute(context.Background(), "Bearer x", ListParams{}, "en")
	require.NoError(t, err)
	assert.NotEqual(t, components.Component{}, tree)
	assert.Equal(t, 1, sf.calls)
	assert.Equal(t, 1, cf.calls)
}

func TestGetUseCase_Execute_SnapshotError_Propagates(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return nil, ErrUnauthorized
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return okCatalog(), nil
	}}
	uc := NewGetUseCase(sf, cf)

	tree, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
	assert.Equal(t, components.Component{}, tree)
}

func TestGetUseCase_Execute_CatalogError_Propagates(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return okResult(), nil
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return nil, assetscatalog.ErrBackend
	}}
	uc := NewGetUseCase(sf, cf)

	tree, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetscatalog.ErrBackend))
	assert.Equal(t, components.Component{}, tree)
}

func TestGetUseCase_Execute_SnapshotError_ShortCircuitsCatalog(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return nil, ErrUnauthorized
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return okCatalog(), nil
	}}
	uc := NewGetUseCase(sf, cf)

	_, err := uc.Execute(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.Equal(t, 0, cf.calls, "catalog must not be called when snapshot fetch fails")
}

func TestGetUseCase_ExecuteSection_HappyPath(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return okResult(), nil
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return okCatalog(), nil
	}}
	uc := NewGetUseCase(sf, cf)

	section, err := uc.ExecuteSection(context.Background(), "Bearer x", ListParams{}, "en")
	require.NoError(t, err)
	assert.NotEqual(t, components.Component{}, section)
	assert.Equal(t, 1, sf.calls)
	assert.Equal(t, 1, cf.calls)
}

func TestGetUseCase_ExecuteSection_SnapshotError_ShortCircuitsCatalog(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return nil, ErrUnauthorized
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return okCatalog(), nil
	}}
	uc := NewGetUseCase(sf, cf)

	section, err := uc.ExecuteSection(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
	assert.Equal(t, components.Component{}, section)
	assert.Equal(t, 0, cf.calls, "catalog must not be called when snapshot fetch fails")
}

func TestGetUseCase_ExecuteSection_CatalogError_Propagates(t *testing.T) {
	sf := &fakeSnapshotFetcher{fn: func(_ context.Context, _ string, _ ListParams) (*ListResult, error) {
		return okResult(), nil
	}}
	cf := &fakeCatalog{fn: func(_ context.Context, _ string) ([]assetscatalog.Asset, error) {
		return nil, assetscatalog.ErrBackend
	}}
	uc := NewGetUseCase(sf, cf)

	_, err := uc.ExecuteSection(context.Background(), "", ListParams{}, "en")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetscatalog.ErrBackend))
}
