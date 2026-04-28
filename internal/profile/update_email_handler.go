package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type emailUpdater interface {
	UpdateEmail(ctx context.Context, authorization, newEmail, currentPassword string) error
}

type UpdateEmailHandler struct {
	updater emailUpdater
	me      meFetcher
}

func NewUpdateEmailHandler(updater emailUpdater, me meFetcher) *UpdateEmailHandler {
	return &UpdateEmailHandler{updater: updater, me: me}
}

func (h *UpdateEmailHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	in, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	newEmail, _ := in["new_email"].(string)
	currentPassword, _ := in["current_password"].(string)

	if err := h.updater.UpdateEmail(c.Request.Context(), auth, newEmail, currentPassword); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			tree := buildEmailCardWith(currentEmailFromMe(c, h.me, auth), newEmail, lang, i18n.T(lang, emailErrorKey(be.Code)))
			c.JSON(http.StatusOK, components.ActionResponse{
				Action: "replace", TargetID: EmailFormID, Tree: &tree,
			})
			return
		}
		respondBackendError(c, "could not update email")
		return
	}

	// Success: re-fetch /v1/user/me to get the updated email and re-render the card.
	updated, err := h.me.GetMe(c.Request.Context(), auth)
	if err != nil {
		respondBackendError(c, "could not refresh profile")
		return
	}
	tree := BuildEmailCard(updated, lang, "", "")
	fb := components.Snackbar("feedback", i18n.T(lang, "profile.email.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action: "replace", TargetID: EmailFormID, Tree: &tree, Feedback: &fb,
	})
}

// currentEmailFromMe returns the user's current email; on fetch failure it
// returns "" — the banner is the dominant signal in this code path.
func currentEmailFromMe(c *gin.Context, me meFetcher, auth string) string {
	u, err := me.GetMe(c.Request.Context(), auth)
	if err != nil || u == nil {
		return ""
	}
	return u.Email
}

func emailErrorKey(code string) string {
	switch code {
	case "MISSING_FIELDS":
		return "profile.email.error.missing_fields"
	case "INVALID_CREDENTIALS":
		return "profile.email.error.invalid_credentials"
	case "EMAIL_ALREADY_EXISTS":
		return "profile.email.error.email_exists"
	default:
		return "profile.email.error.missing_fields"
	}
}
