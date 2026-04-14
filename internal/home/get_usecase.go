package home

import (
	"context"

	"github.com/project/vk-investment-middleend/internal/components"
)

// GetUseCase retrieves data and builds the home screen component tree.
type GetUseCase struct {
	client *Client
}

func NewGetUseCase(client *Client) *GetUseCase {
	return &GetUseCase{client: client}
}

// Execute fetches backend data and returns the home screen SDUI component tree.
func (uc *GetUseCase) Execute(ctx context.Context, lang, platform string) (components.Component, error) {
	// TODO: fetch data from backend via uc.client
	return BuildScreen(lang, platform), nil
}
