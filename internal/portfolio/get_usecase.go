package portfolio

import (
	"context"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
)

// positionsFetcher is the interface the use case depends on; *Client satisfies it.
type positionsFetcher interface {
	GetPositions(ctx context.Context, authorization string) ([]Position, error)
}

type GetUseCase struct {
	client positionsFetcher
}

func NewGetUseCase(client positionsFetcher) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches positions from the backend, sorts them, and builds the
// portfolio SDUI tree. `now` is used for relative-time formatting.
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string, now time.Time) (components.Component, error) {
	positions, err := uc.client.GetPositions(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	SortPositions(positions)
	return BuildScreen(positions, lang, now), nil
}
