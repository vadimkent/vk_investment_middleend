package assets

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

type CreateModalHandler struct{}

func NewCreateModalHandler() *CreateModalHandler { return &CreateModalHandler{} }

func (h *CreateModalHandler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	lang := parseLang(c)
	modal := BuildCreateModal(params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-modal-slot",
		Tree:     &modal,
	})
}
