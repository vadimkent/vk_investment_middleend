package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

type LoginHandler struct {
	client *Client
}

func NewLoginHandler(client *Client) *LoginHandler {
	return &LoginHandler{client: client}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *LoginHandler) Post(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid request body"}})
		return
	}
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "email and password are required"}})
		return
	}

	res, err := h.client.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			fb := components.Snackbar("login-error", "Invalid email or password", "error")
			c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
			return
		}
		fb := components.Snackbar("login-error", "Login failed. Please try again.", "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	fb := components.Snackbar("login-ok", "Welcome", "success")
	c.JSON(http.StatusOK, components.NavigateResponse("/portfolio", &fb).WithAuth(res.Token, res.ExpiresAt))
}
