package shared

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RespondError logs the error and returns a JSON error response.
func RespondError(c *gin.Context, status int, err error) {
	log.Error().Err(err).Str("path", c.Request.URL.Path).Msg("request error")
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

// RespondUnauthorized aborts the request with 401 and a redirect hint for the
// frontend. The middleend decides where unauthenticated users should land; the
// frontend reads `redirect` and navigates there.
func RespondUnauthorized(c *gin.Context, redirect string) {
	c.AbortWithStatusJSON(401, gin.H{
		"error":    "unauthorized",
		"redirect": redirect,
	})
}
