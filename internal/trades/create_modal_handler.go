package trades

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// CreateModalHandler serves GET /actions/trades/create_modal: returns the
// empty create-trade modal (no trade fetch needed). The asset catalog is
// required to populate the asset-picker.
type CreateModalHandler struct{ catalog catalogFetcher }

func NewCreateModalHandler(catalog catalogFetcher) *CreateModalHandler {
	return &CreateModalHandler{catalog: catalog}
}

// Get validates list-context query params (preserved on submit), fetches the
// catalog, and returns an ActionResponse that replaces the trades-modal-slot
// with the Create modal tree.
func (h *CreateModalHandler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	cat, err := h.catalog.List(c.Request.Context(), auth)
	if err != nil {
		if errors.Is(err, assetscatalog.ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load assets"}})
		return
	}

	modal := BuildCreateModal(cat, params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &modal,
	})
}
