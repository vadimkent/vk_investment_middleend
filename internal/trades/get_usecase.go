package trades

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// tradeFetcher is the narrow trades-list client interface the use case depends on.
type tradeFetcher interface {
	List(ctx context.Context, authorization string, p ListParams) (*ListResult, error)
}

// catalogFetcher is the narrow asset-catalog interface the use case depends on.
type catalogFetcher interface {
	List(ctx context.Context, authorization string) ([]assetscatalog.Asset, error)
}

type GetUseCase struct {
	client  tradeFetcher
	catalog catalogFetcher
}

func NewGetUseCase(client tradeFetcher, catalog catalogFetcher) *GetUseCase {
	return &GetUseCase{client: client, catalog: catalog}
}

// Execute fetches trades and the full catalog, returning the full screen tree.
// The first error is surfaced verbatim; the trades error short-circuits the
// catalog call so handlers can distinguish trades-401 from catalog-401.
func (uc *GetUseCase) Execute(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	cat, err := uc.catalog.List(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	return BuildScreen(res, cat, p, lang), nil
}

// ExecuteSection fetches the same data and returns only the list subtree.
func (uc *GetUseCase) ExecuteSection(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	cat, err := uc.catalog.List(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	return BuildTradesSection(res, cat, p, lang), nil
}
