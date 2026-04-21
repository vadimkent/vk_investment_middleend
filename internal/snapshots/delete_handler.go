package snapshots

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// snapshotDeleter is the narrow interface the Delete handler depends on.
// *Client (see mutate_client.go) satisfies it.
type snapshotDeleter interface {
	DeleteSnapshot(ctx context.Context, authorization, id string) error
}

// DeleteHandler serves DELETE /actions/snapshots/:id: deletes the snapshot
// and, on success, rebuilds the snapshots screen with the preserved list
// context. There is no force flag and no two-stage confirmation flow.
type DeleteHandler struct {
	deleter snapshotDeleter
	uc      *GetUseCase
}

func NewDeleteHandler(deleter snapshotDeleter, uc *GetUseCase) *DeleteHandler {
	return &DeleteHandler{deleter: deleter, uc: uc}
}

// Delete handles the delete-snapshot action.
func (h *DeleteHandler) Delete(c *gin.Context) {
	id := c.Param("id")
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

	if err := h.deleter.DeleteSnapshot(c.Request.Context(), auth, id); err != nil {
		if respondSnapshotFetchError(c, err, "could not delete snapshot") {
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
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh snapshots"}})
		return
	}
	fb := components.Snackbar("feedback", i18n.T(lang, "snapshots.delete.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}
