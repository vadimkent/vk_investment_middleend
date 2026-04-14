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
	posErr           error
	evoErr           error
	gotAuthP         string
	gotAuthE         string
	gotLastN         int
	gotIncludeClosed bool
}

func (f *fakeFetcher) GetPositions(ctx context.Context, auth string, includeClosed bool) ([]Position, error) {
	f.gotAuthP = auth
	f.gotIncludeClosed = includeClosed
	return f.positions, f.posErr
}

func (f *fakeFetcher) GetEvolutionLast(ctx context.Context, auth string, n int) ([]EvolutionPoint, error) {
	f.gotAuthE = auth
	f.gotLastN = n
	return f.evolution, f.evoErr
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
