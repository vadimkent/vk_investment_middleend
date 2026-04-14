package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	positions []Position
	err       error
	gotAuth   string
}

func (f *fakeClient) GetPositions(ctx context.Context, auth string) ([]Position, error) {
	f.gotAuth = auth
	return f.positions, f.err
}

func TestGetUseCase_ReturnsBuiltScreen(t *testing.T) {
	v := 100.0
	client := &fakeClient{positions: []Position{
		{AssetID: "a1", Ticker: "AAPL", Name: "Apple", Currency: "USD", CurrentValue: &v},
	}}
	uc := NewGetUseCase(client)
	now := time.Now()
	screen, err := uc.Execute(context.Background(), "Bearer tok", "en", now)
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok", client.gotAuth)
	assert.Equal(t, "screen", screen.Type)
	assert.Equal(t, "portfolio", screen.ID)
}

func TestGetUseCase_SortsBeforeBuilding(t *testing.T) {
	v1, v2 := 100.0, 500.0
	client := &fakeClient{positions: []Position{
		{AssetID: "a1", Ticker: "A", Currency: "USD", CurrentValue: &v1},
		{AssetID: "b1", Ticker: "B", Currency: "USD", CurrentValue: &v2},
	}}
	uc := NewGetUseCase(client)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	body := findDescendantByID(screen, "positions-body")
	require.NotNil(t, body)
	require.Len(t, body.Children, 2)
	assert.Equal(t, "position-b1", body.Children[0].ID)
	assert.Equal(t, "position-a1", body.Children[1].ID)
}

func TestGetUseCase_EmptyPositions(t *testing.T) {
	client := &fakeClient{positions: []Position{}}
	uc := NewGetUseCase(client)
	screen, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.NoError(t, err)
	assert.NotNil(t, findDescendantByID(screen, "portfolio-empty"))
}

func TestGetUseCase_PropagatesErrors(t *testing.T) {
	client := &fakeClient{err: ErrUnauthorized}
	uc := NewGetUseCase(client)
	_, err := uc.Execute(context.Background(), "Bearer t", "en", time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthorized))
}
