package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/register"
)

const minPasswordLen = 8

// registrar is the contract the handler depends on. *Client implements it.
type registrar interface {
	Register(ctx context.Context, email, password string) error
}

type RegisterHandler struct {
	reg registrar
}

func NewRegisterHandler(reg registrar) *RegisterHandler {
	return &RegisterHandler{reg: reg}
}

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (h *RegisterHandler) Post(c *gin.Context) {
	lang := parseLang(c)

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondReplace(c, lang, "", "auth.error_validation", false)
		return
	}
	email := strings.TrimSpace(req.Email)

	// Middleend-side defense in depth — these errors are also gated client-side.
	if email == "" || req.Password == "" || req.ConfirmPassword == "" {
		respondReplace(c, lang, email, "auth.error_validation", false)
		return
	}
	if len(req.Password) < minPasswordLen {
		respondReplace(c, lang, email, "auth.error_validation", false)
		return
	}
	if req.Password != req.ConfirmPassword {
		respondReplace(c, lang, email, "auth.error_validation", false)
		return
	}

	err := h.reg.Register(c.Request.Context(), email, req.Password)
	switch {
	case err == nil:
		fb := components.Snackbar("feedback", i18n.T(lang, "auth.register_success"), "success")
		c.JSON(http.StatusOK, components.ActionResponse{
			Action: "navigate", TargetID: "/screens/login", Feedback: &fb,
		})
	case errors.Is(err, ErrEmailAlreadyExists):
		respondReplace(c, lang, email, "auth.error_email_exists", false)
	case errors.Is(err, ErrRegistrationDisabled):
		respondReplace(c, lang, "", "auth.error_registration_disabled", true)
	default:
		fb := components.Snackbar("feedback", i18n.T(lang, "auth.error_transient"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{
			Action: "none", Feedback: &fb,
		})
	}
}

func respondReplace(c *gin.Context, lang, prefillEmail, errorKey string, submitDisabled bool) {
	tree := register.BuildForm(lang, prefillEmail, i18n.T(lang, errorKey), submitDisabled)
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: register.FormID, Tree: &tree,
	})
}
