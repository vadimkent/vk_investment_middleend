package profile

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
)

// Narrow interfaces — easier to stub in tests.
type meFetcher interface {
	GetMe(ctx context.Context, authorization string) (*User, error)
}

type configFetcher interface {
	GetConfig(ctx context.Context, authorization string) (*AppConfig, error)
}

type GetUseCase struct {
	me  meFetcher
	cfg configFetcher
}

func NewGetUseCase(me meFetcher, cfg configFetcher) *GetUseCase {
	return &GetUseCase{me: me, cfg: cfg}
}

// Execute fetches the current user and the config in sequence (matching the
// pattern used by snapshots — sequential with short-circuit on error).
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string) (components.Component, error) {
	me, err := uc.me.GetMe(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	cfg, err := uc.cfg.GetConfig(ctx, authorization)
	if err != nil {
		return components.Component{}, err
	}
	return BuildScreen(me, cfg, lang), nil
}
