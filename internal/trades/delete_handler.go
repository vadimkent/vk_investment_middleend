package trades

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// tradeDeleter is the narrow interface the Delete handler depends on. *Client
// (see mutate_client.go) satisfies it.
type tradeDeleter interface {
	DeleteTrade(ctx context.Context, authorization, id string) error
}

// DeleteHandler serves DELETE /actions/trades/:id: deletes the trade and,
// on success, rebuilds the trades screen with the preserved list context.
// Unlike assets, there is no force flag and no two-stage confirmation flow.
type DeleteHandler struct {
	deleter tradeDeleter
	uc      *GetUseCase
}

func NewDeleteHandler(deleter tradeDeleter, uc *GetUseCase) *DeleteHandler {
	return &DeleteHandler{deleter: deleter, uc: uc}
}

// Delete handles the delete-trade action.
func (h *DeleteHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondBadRequest(c, "missing id")
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	// respondTradeFetchError handles ErrUnauthorized / ErrTradeNotFound /
	// default 502. *BackendValidationError is exceptional here (trades delete
	// has no modal-replay flow) and falls through to the default 502 branch
	// because errors.Is won't match it — which is the correct behavior.
	if err := h.deleter.DeleteTrade(c.Request.Context(), auth, id); err != nil {
		if respondTradeFetchError(c, err, "could not delete trade") {
			return
		}
	}

	// Success — rebuild the screen tree and attach a success snackbar.
	tree, err := h.uc.Execute(c.Request.Context(), auth, params, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh trades"}})
		return
	}
	fb := components.Snackbar("feedback", i18n.T(lang, "trades.delete.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}
