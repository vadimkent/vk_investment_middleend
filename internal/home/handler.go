package home

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

// Handler handles the home screen HTTP requests.
// It only parses requests and returns responses — no business logic.
type Handler struct {
	getUC *GetUseCase
}

func NewHandler(getUC *GetUseCase) *Handler {
	return &Handler{getUC: getUC}
}

// Get handles GET /screens/home — returns the home screen component tree.
func (h *Handler) Get(c *gin.Context) {
	lang := parseLang(c)
	platform := c.GetHeader("X-Platform")
	if platform == "" {
		platform = "web"
	}

	screen, err := h.getUC.Execute(c.Request.Context(), lang, platform)
	if err != nil {
		shared.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, screen)
}

// parseLang extracts the primary language from the Accept-Language header.
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
