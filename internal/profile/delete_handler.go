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

type accountDeleter interface {
	DeleteAccount(ctx context.Context, authorization, password string) error
}

type DeleteHandler struct {
	deleter accountDeleter
}

func NewDeleteHandler(deleter accountDeleter) *DeleteHandler {
	return &DeleteHandler{deleter: deleter}
}

func (h *DeleteHandler) Post(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	in, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	password, _ := in["password"].(string)

	if err := h.deleter.DeleteAccount(c.Request.Context(), auth, password); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			modal := BuildDeleteModal(lang, i18n.T(lang, dangerErrorKey(be.Code)))
			c.JSON(http.StatusOK, components.ActionResponse{
				Action: "replace", TargetID: ModalSlotID, Tree: &modal,
			})
			return
		}
		respondBackendError(c, "could not delete account")
		return
	}

	c.JSON(http.StatusOK, components.LogoutResponse("/screens/login"))
}

func dangerErrorKey(code string) string {
	switch code {
	case "MISSING_FIELDS":
		return "profile.danger.error.missing_fields"
	case "INVALID_CREDENTIALS":
		return "profile.danger.error.invalid_credentials"
	default:
		return "profile.danger.error.invalid_credentials"
	}
}
