package profile

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseLang extracts the base language tag from Accept-Language, falling back
// to "en" when the header is missing or blank.
// Mirrors the snapshots package convention; no canonical helper exists in internal/i18n.
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

// parseJSONBody decodes the request body into a generic map. Returns a
// non-nil error if Content-Type isn't application/json or the body is malformed.
func parseJSONBody(c *gin.Context) (map[string]any, error) {
	if c.GetHeader("Content-Type") != "application/json" {
		return nil, errors.New("expected application/json")
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// respondBadRequest aborts with 400 BAD_REQUEST envelope.
func respondBadRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": msg}})
}

// respondBackendError aborts with 502 BACKEND_ERROR envelope.
func respondBackendError(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(502, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": msg}})
}
