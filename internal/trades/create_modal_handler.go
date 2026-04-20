package trades

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
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
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	cat, err := h.catalog.List(c.Request.Context(), auth)
	if respondCatalogFetchError(c, err, "could not load assets") {
		return
	}

	modal := BuildCreateModal(cat, params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &modal,
	})
}
