package portfolio

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/project/vk-investment-middleend/internal/components"
)

// portfolioFetcher is the interface the use case depends on; *Client satisfies it.
type portfolioFetcher interface {
	GetPositions(ctx context.Context, authorization string) ([]Position, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
}

type GetUseCase struct {
	client portfolioFetcher
}

func NewGetUseCase(client portfolioFetcher) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches positions and evolution in parallel, sorts positions, computes
// summary metrics, and builds the SDUI tree. Positions is the critical path —
// its failure aborts. Evolution failure (unless it is an auth error) is
// tolerated and results in an empty evolution list.
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string, now time.Time) (components.Component, error) {
	var positions []Position
	var evolution []EvolutionPoint

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		p, err := uc.client.GetPositions(gctx, authorization)
		if err != nil {
			return err
		}
		positions = p
		return nil
	})

	g.Go(func() error {
		e, err := uc.client.GetEvolutionLast(gctx, authorization, 2)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				return err
			}
			// Non-auth evolution failure: tolerated.
			return nil
		}
		evolution = e
		return nil
	})

	if err := g.Wait(); err != nil {
		return components.Component{}, err
	}

	SortPositions(positions)
	return BuildScreen(positions, evolution, lang, now), nil
}
