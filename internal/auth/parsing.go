package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// parseLang extracts the base language tag from Accept-Language, falling back
// to "en".
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
