package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

type RegisterHandler struct {
	client *Client
}

func NewRegisterHandler(client *Client) *RegisterHandler {
	return &RegisterHandler{client: client}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *RegisterHandler) Post(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid request body"}})
		return
	}
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "email and password are required"}})
		return
	}

	err := h.client.Register(c.Request.Context(), req.Email, req.Password)
	switch {
	case err == nil:
		fb := components.Snackbar("register-ok", "Account created. Please log in.", "success")
		c.JSON(http.StatusOK, components.ActionResponse{
			Action:   "navigate",
			TargetID: "/login",
			Feedback: &fb,
		})
	case errors.Is(err, ErrRegistrationDisabled):
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "REGISTRATION_DISABLED", "message": "registration is disabled"}})
	case errors.Is(err, ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "EMAIL_ALREADY_EXISTS", "message": "email already registered"}})
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "registration failed"}})
	}
}
