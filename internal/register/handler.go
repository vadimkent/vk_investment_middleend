package register

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Get(c *gin.Context) {
	lang := parseLang(c)
	c.JSON(http.StatusOK, BuildScreen(lang, ""))
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
