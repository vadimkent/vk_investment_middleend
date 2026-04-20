package trades

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// tradeGetter is the narrow interface the edit/delete modal (and the
// mutation) handlers need to fetch a single trade by id. The concrete
// *Client (see mutate_client.go) satisfies it.
type tradeGetter interface {
	GetTrade(ctx context.Context, authorization, id string) (*Trade, error)
}

// EditModalHandler serves GET /actions/trades/edit_modal: fetches the trade
// (by id) and the asset catalog, then returns the Edit modal tree
// pre-populated with the trade's values. List context (asset_id, trade_type,
// offset) is preserved in the modal's submit URL so the post-mutation list
// refresh re-renders the same filter/offset the user came from.
type EditModalHandler struct {
	client  tradeGetter
	catalog catalogFetcher
}

func NewEditModalHandler(client tradeGetter, catalog catalogFetcher) *EditModalHandler {
	return &EditModalHandler{client: client, catalog: catalog}
}

// Get validates the id, parses list-context params, fetches the trade and
// the catalog, and returns an ActionResponse that replaces the
// trades-modal-slot with the Edit modal tree. Error priority: missing id →
// bad query → trade errors (401, 404, 502) → catalog errors (401, 502).
func (h *EditModalHandler) Get(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "missing id"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	trade, err := h.client.GetTrade(c.Request.Context(), auth, id)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrTradeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "trade not found"}})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load trade"}})
		return
	}

	cat, err := h.catalog.List(c.Request.Context(), auth)
	if err != nil {
		if errors.Is(err, assetscatalog.ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load assets"}})
		return
	}

	modal := BuildEditModal(*trade, cat, params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &modal,
	})
}
