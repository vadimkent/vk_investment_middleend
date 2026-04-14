package shell

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
)

type GetUseCase struct{}

func NewGetUseCase() *GetUseCase {
	return &GetUseCase{}
}

// Execute builds the app shell component tree with navigation adapted per platform.
func (uc *GetUseCase) Execute(ctx context.Context, lang, platform string) (components.Component, error) {
	return BuildShell(lang, platform), nil
}
