package snapshots

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/project/vk-investment-middleend/internal/components"
)

// DeleteModalHandler serves GET /actions/snapshots/delete_modal: fetches the
// snapshot (by id) and returns the Delete confirmation modal tree. The
// recorded_at date is interpolated into the confirmation message.
type DeleteModalHandler struct{ fetcher snapshotGetter }

func NewDeleteModalHandler(fetcher snapshotGetter) *DeleteModalHandler {
	return &DeleteModalHandler{fetcher: fetcher}
}

// Get validates the id (UUID), parses list-context params, fetches the
// snapshot, and returns an ActionResponse that replaces the
// snapshots-modal-slot with the Delete modal tree. Error priority mirrors the
// Edit handler: missing id → invalid id → bad query → snapshot errors (401, 404, 502).
func (h *DeleteModalHandler) Get(c *gin.Context) {
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

	modal := BuildDeleteModal(snap, params, lang)
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &modal,
	})
}
