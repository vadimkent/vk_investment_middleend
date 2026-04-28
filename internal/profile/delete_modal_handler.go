package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

// DeleteModalHandler serves GET /actions/profile/delete_modal: returns the
// delete-account confirmation modal into the modal slot. Requires no backend
// call — the modal is purely a UI surface.
type DeleteModalHandler struct{}

func NewDeleteModalHandler() *DeleteModalHandler { return &DeleteModalHandler{} }

func (h *DeleteModalHandler) Get(c *gin.Context) {
	lang := parseLang(c)
	modal := BuildDeleteModal(lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: ModalSlotID, Tree: &modal,
	})
}
