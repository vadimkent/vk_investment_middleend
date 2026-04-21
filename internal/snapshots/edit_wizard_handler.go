package snapshots

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/project/vk-investment-middleend/internal/components"
)

// EditWizardHandler serves GET /actions/snapshots/edit_wizard: fetches the
// snapshot (by id) and the asset catalog, then returns the Edit wizard tree
// pre-populated with the snapshot's values. List context (is_full_snapshot,
// offset) is preserved in the wizard's submit URL so the post-mutation list
// refresh re-renders the same filter/offset the user came from.
type EditWizardHandler struct {
	fetcher snapshotGetter
	catalog catalogFetcher
}

func NewEditWizardHandler(fetcher snapshotGetter, catalog catalogFetcher) *EditWizardHandler {
	return &EditWizardHandler{fetcher: fetcher, catalog: catalog}
}

// Get validates the id (UUID), parses list-context params, fetches the
// snapshot and the catalog, and returns an ActionResponse that replaces the
// snapshots-modal-slot with the Edit wizard tree. Error priority: missing id →
// invalid id → bad query → snapshot errors (401, 404, 502) → catalog errors (401, 502).
func (h *EditWizardHandler) Get(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		respondBadRequest(c, "missing id")
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		respondBadRequest(c, "invalid id")
		return
	}

	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	snap, err := h.fetcher.GetSnapshot(c.Request.Context(), auth, id)
	if respondSnapshotFetchError(c, err, "could not load snapshot") {
		return
	}

	cat, err := h.catalog.List(c.Request.Context(), auth)
	if respondCatalogFetchError(c, err, "could not load assets") {
		return
	}

	wizard := BuildEditWizard(snap, cat, params, lang, "", "", nil)
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &wizard,
	})
}
