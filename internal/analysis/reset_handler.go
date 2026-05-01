package analysis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
)

type ResetHandler struct{}

func NewResetHandler() *ResetHandler { return &ResetHandler{} }

func (h *ResetHandler) Get(c *gin.Context) {
	lang := resolveLang(c)
	tree := BuildContentStart(lang, "", "")
	c.JSON(http.StatusOK, components.ReplaceResponse("analysis-content", tree, nil))
}
