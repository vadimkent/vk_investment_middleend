package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFetcher struct {
	positions        []Position
	evolution        []EvolutionPoint
	chart            []EvolutionPoint
	posErr           error
	evoErr           error
	chartErr         error
	gotAuthP         string
	gotAuthE         string
	gotLastN         int
	gotIncludeClosed bool
	gotChartQuery    EvolutionQuery
}

func (f *fakeFetcher) GetPositions(ctx context.Context, auth string, includeClosed, live, refresh bool) (*PortfolioResponse, error) {
	f.gotAuthP = auth
	f.gotIncludeClosed = includeClosed
	return &PortfolioResponse{Positions: f.positions}, f.posErr
}

func (f *fakeFetcher) GetEvolutionLast(ctx context.Context, auth string, n int) ([]EvolutionPoint, error) {
	f.gotAuthE = auth
	f.gotLastN = n
	return f.evolution, f.evoErr
}

func (f *fakeFetcher) GetEvolution(ctx context.Context, auth string, q EvolutionQuery) ([]EvolutionPoint, error) {
	f.gotChartQuery = q
	return f.chart, f.chartErr
}

func TestGetUseCase_FetchesBothInParallel(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		evolution: []EvolutionPoint{
			{Currency: "USD", RecordedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), TotalValue: 100},
			{Currency: "USD", RecordedAt: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), TotalValue: 110},
		},
	}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "Bearer t", f.gotAuthP)
	assert.Equal(t, "Bearer t", f.gotAuthE)
	assert.Equal(t, 2, f.gotLastN)
}

func TestGetUseCase_EvolutionFailureDoesNotFail(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		evoErr:    ErrBackend,
	}
	uc := NewGetUseCase(f)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "screen", screen.Type)
}

func TestGetUseCase_PositionsFailurePropagates(t *testing.T) {
	f := &fakeFetcher{posErr: ErrUnauthorized}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestGetUseCase_EmptyPositionsReturnsEmptyScreen(t *testing.T) {
	f := &fakeFetcher{positions: []Position{}}
	uc := NewGetUseCase(f)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.NotNil(t, findDescendantByID(screen, "portfolio-empty"))
}

func TestGetUseCase_EvolutionAuthErrorTreatedAsPositionsAuthError(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		evoErr:    ErrUnauthorized,
	}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}

func TestGetUseCase_PassesIncludeClosedFalse(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}}}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.False(t, f.gotIncludeClosed)
}

func TestGetUseCase_FetchesChartEvolutionWith100Points(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}}}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, 100, f.gotChartQuery.Points)
	assert.Nil(t, f.gotChartQuery.From)
	assert.Equal(t, "", f.gotChartQuery.Currency)
}

func TestGetUseCase_ChartFetchFailureDoesNotFail(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		chartErr:  ErrBackend,
	}
	uc := NewGetUseCase(f)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "screen", screen.Type)
}

func TestGetUseCase_ChartAuthErrorPropagates(t *testing.T) {
	v := 100.0
	f := &fakeFetcher{
		positions: []Position{{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v}},
		chartErr:  ErrUnauthorized,
	}
	uc := NewGetUseCase(f)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
