package shell

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

type Handler struct {
	getUC *GetUseCase
}

func NewHandler(getUC *GetUseCase) *Handler {
	return &Handler{getUC: getUC}
}

// Get handles GET /shell — returns the app shell with navigation and layout.
func (h *Handler) Get(c *gin.Context) {
	lang := parseLang(c)
	platform := c.GetHeader("X-Platform")
	if platform == "" {
		platform = "web"
	}

	shell, err := h.getUC.Execute(c.Request.Context(), lang, platform)
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, shell)
}

func parseLang(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return "en"
	}
	parts := strings.SplitN(header, ",", 2)
	lang := strings.SplitN(parts[0], "-", 2)[0]
	lang = strings.SplitN(lang, ";", 2)[0]
	return strings.TrimSpace(lang)
}
