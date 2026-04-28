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

// Execute fetches the current user and the config in parallel, then builds the
// screen. If me errors, that error takes precedence; otherwise cfg's error is
// returned.
func (uc *GetUseCase) Execute(ctx context.Context, authorization, lang string) (components.Component, error) {
	type meResult struct {
		user *User
		err  error
	}
	type cfgResult struct {
		cfg *AppConfig
		err error
	}
	meCh := make(chan meResult, 1)
	cfgCh := make(chan cfgResult, 1)

	go func() {
		u, err := uc.me.GetMe(ctx, authorization)
		meCh <- meResult{u, err}
	}()
	go func() {
		c, err := uc.cfg.GetConfig(ctx, authorization)
		cfgCh <- cfgResult{c, err}
	}()

	mr := <-meCh
	cr := <-cfgCh

	if mr.err != nil {
		return components.Component{}, mr.err
	}
	if cr.err != nil {
		return components.Component{}, cr.err
	}
	return BuildScreen(mr.user, cr.cfg, lang), nil
}
