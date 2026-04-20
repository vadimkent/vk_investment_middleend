package trades

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// DeleteModalHandler serves GET /actions/trades/delete_modal: fetches the
// trade (by id) and the asset catalog — both are needed because the
// confirmation message interpolates the trade's type/quantity/date and the
// ticker from the catalog. Returns the Delete confirmation modal tree.
type DeleteModalHandler struct {
	client  tradeGetter
	catalog catalogFetcher
}

func NewDeleteModalHandler(client tradeGetter, catalog catalogFetcher) *DeleteModalHandler {
	return &DeleteModalHandler{client: client, catalog: catalog}
}

// Get validates the id, parses list-context params, fetches the trade and
// the catalog, and returns an ActionResponse that replaces the
// trades-modal-slot with the Delete modal tree. Error priority mirrors the
// Edit handler: missing id → bad query → trade errors → catalog errors.
func (h *DeleteModalHandler) Get(c *gin.Context) {
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

	modal := BuildDeleteModal(*trade, cat, params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &modal,
	})
}
