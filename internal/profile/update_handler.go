package profile

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type profileUpdater interface {
	UpdateProfile(ctx context.Context, authorization string, body map[string]any) (*User, error)
}

type UpdateHandler struct {
	updater profileUpdater
	cfg     configFetcher
}

func NewUpdateHandler(updater profileUpdater, cfg configFetcher) *UpdateHandler {
	return &UpdateHandler{updater: updater, cfg: cfg}
}

func (h *UpdateHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	submitted, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}

	displayName, currency := readProfileFields(submitted)
	body := buildProfileUpdateBody(displayName, currency)

	updated, err := h.updater.UpdateProfile(c.Request.Context(), auth, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			h.respondValidation(c, lang, displayName, currency, be)
			return
		}
		respondBackendError(c, "could not update profile")
		return
	}

	cfg, err := h.cfg.GetConfig(c.Request.Context(), auth)
	if err != nil {
		respondBackendError(c, "could not load currencies")
		return
	}
	tree := BuildProfileCard(updated, cfg, lang, "")
	fb := components.Snackbar("feedback", i18n.T(lang, "profile.update.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ProfileFormID,
		Tree:     &tree,
		Feedback: &fb,
	})
}

// readProfileFields pulls the two fields from the form-serialized JSON body.
// Inputs are flat (display_name, default_currency) — the FE serializes form
// input names directly, not the nested shape the BE expects. Empty /
// whitespace strings are normalised to "".
func readProfileFields(in map[string]any) (displayName, currency string) {
	if v, ok := in["display_name"].(string); ok {
		displayName = strings.TrimSpace(v)
	}
	if v, ok := in["default_currency"].(string); ok {
		currency = strings.TrimSpace(v)
	}
	return
}

// buildProfileUpdateBody maps the form values to the BE payload, sending null
// for cleared fields.
func buildProfileUpdateBody(displayName, currency string) map[string]any {
	body := map[string]any{}
	if displayName == "" {
		body["display_name"] = nil
	} else {
		body["display_name"] = displayName
	}
	prefs := map[string]any{}
	if currency == "" {
		prefs["default_currency"] = nil
	} else {
		prefs["default_currency"] = currency
	}
	body["preferences"] = prefs
	return body
}

func (h *UpdateHandler) respondValidation(c *gin.Context, lang, displayName, currency string, be *BackendValidationError) {
	auth := c.GetHeader("Authorization")
	cfg, err := h.cfg.GetConfig(c.Request.Context(), auth)
	if err != nil {
		respondBackendError(c, "could not load currencies")
		return
	}
	msg := i18n.T(lang, profileErrorKey(be.Code))
	tree := buildProfileCardWith(displayName, currency, cfg, lang, msg)
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ProfileFormID,
		Tree:     &tree,
	})
}

// profileErrorKey maps BE validation codes to i18n banner keys.
func profileErrorKey(code string) string {
	switch code {
	case "INVALID_DISPLAY_NAME":
		return "profile.update.error.invalid_display_name"
	case "INVALID_CURRENCY":
		return "profile.update.error.invalid_currency"
	default:
		return "profile.update.error.invalid_display_name"
	}
}
