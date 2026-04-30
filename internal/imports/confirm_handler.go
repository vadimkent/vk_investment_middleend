package imports

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

type ConfirmHandler struct {
	client *Client
}

func NewConfirmHandler(c *Client) *ConfirmHandler { return &ConfirmHandler{client: c} }

func (h *ConfirmHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	id := c.Param("id")

	res, err := h.client.ConfirmSession(c.Request.Context(), resolveAuth(c), id)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			fb := components.Snackbar("feedback", be.Message, "error")
			c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
			return
		}
		if errors.Is(err, ErrSessionNotFound) {
			tree := BuildRootColumn(lang)
			fb := components.Snackbar("feedback", i18n.T(lang, "import.session_expired"), "warning")
			c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		fb := components.Snackbar("feedback", i18n.T(lang, "import.failure_generic"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildRootColumn(lang)
	tmpl := i18n.T(lang, "import.success")
	msg := strings.NewReplacer(
		"{assets}", fmt.Sprintf("%d", res.AssetsCreated),
		"{trades}", fmt.Sprintf("%d", res.TradesImported),
		"{snapshots}", fmt.Sprintf("%d", res.SnapshotsImported),
		"{warnings}", fmt.Sprintf("%d", res.Warnings),
	).Replace(tmpl)
	fb := components.Snackbar("feedback", msg, "success")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
}
