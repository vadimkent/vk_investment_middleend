package snapshots

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

// CreateWizardHandler serves GET /actions/snapshots/create_wizard: returns the
// create-snapshot wizard (no snapshot fetch needed). The asset catalog is
// required to populate one entry step per asset.
type CreateWizardHandler struct{ catalog catalogFetcher }

func NewCreateWizardHandler(catalog catalogFetcher) *CreateWizardHandler {
	return &CreateWizardHandler{catalog: catalog}
}

// Get validates list-context query params (preserved on submit), fetches the
// catalog, and returns an ActionResponse that replaces the snapshots-modal-slot
// with the Create wizard tree.
func (h *CreateWizardHandler) Get(c *gin.Context) {
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

	wizard := BuildCreateWizard(cat, params, lang, "", "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &wizard,
	})
}
