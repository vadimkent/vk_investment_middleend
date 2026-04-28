package imports

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler renders the screen tree for GET /screens/import.
type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Get(c *gin.Context) {
	lang := resolveLang(c)
	c.JSON(http.StatusOK, BuildRoot(lang))
}

func resolveLang(c *gin.Context) string {
	if l := c.Query("lang"); l != "" {
		return l
	}
	if l := c.GetHeader("Accept-Language"); l != "" {
		// First two letters of the first tag (e.g. "es-ES,en;q=0.9" → "es")
		if len(l) >= 2 {
			return l[:2]
		}
	}
	return "en"
}

func resolveAuth(c *gin.Context) string {
	if v, ok := c.Get("authorization"); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			return s
		}
	}
	return c.GetHeader("Authorization")
}
