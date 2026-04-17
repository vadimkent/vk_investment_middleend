package auth

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

// RequireAuth returns a Gin middleware that validates the Authorization Bearer
// JWT with the given secret and leeway. On success it stores the sub claim
// under "user_id" in the Gin context. On failure it aborts with 401 and a
// redirect hint pointing to loginRedirect (typically "/login").
func RequireAuth(secret string, leeway time.Duration, loginRedirect string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			shared.RespondUnauthorized(c, loginRedirect)
			return
		}
		token := strings.TrimSpace(header[len(prefix):])
		userID, err := Validate(token, secret, leeway)
		if err != nil {
			shared.RespondUnauthorized(c, loginRedirect)
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}
