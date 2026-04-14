package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RequireAuth returns a Gin middleware that validates the Authorization Bearer
// JWT with the given secret and leeway. On success it stores the sub claim
// under "user_id" in the Gin context.
func RequireAuth(secret string, leeway time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			unauthorized(c)
			return
		}
		token := strings.TrimSpace(header[len(prefix):])
		userID, err := Validate(token, secret, leeway)
		if err != nil {
			unauthorized(c)
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "UNAUTHORIZED", "message": "authentication required"},
	})
}
