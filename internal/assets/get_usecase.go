package assets

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
)

// assetFetcher is the narrow client interface the use case depends on.
type assetFetcher interface {
	List(ctx context.Context, authorization string, p ListParams) (*ListResult, error)
}

type GetUseCase struct {
	client assetFetcher
}

func NewGetUseCase(client assetFetcher) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches and returns the full screen tree.
func (uc *GetUseCase) Execute(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	return BuildScreen(res, p, lang), nil
}

// ExecuteSection fetches and returns only the replaceable assets-section subtree.
func (uc *GetUseCase) ExecuteSection(ctx context.Context, authorization string, p ListParams, lang string) (components.Component, error) {
	res, err := uc.client.List(ctx, authorization, p)
	if err != nil {
		return components.Component{}, err
	}
	return BuildAssetsSection(res, p, lang), nil
}
