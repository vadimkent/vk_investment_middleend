package imports

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

type CancelHandler struct {
	client *Client
}

func NewCancelHandler(c *Client) *CancelHandler { return &CancelHandler{client: c} }

func (h *CancelHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	id := c.Param("id")

	if err := h.client.CancelSession(c.Request.Context(), resolveAuth(c), id); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		fb := components.Snackbar("feedback", i18n.T(lang, "import.failure_generic"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildRootColumn(lang)
	fb := components.Snackbar("feedback", i18n.T(lang, "import.cancelled"), "info")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
}
