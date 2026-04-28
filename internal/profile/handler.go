package profile

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

type Handler struct{ uc *GetUseCase }

func NewHandler(uc *GetUseCase) *Handler { return &Handler{uc: uc} }

// Get serves GET /screens/profile.
func (h *Handler) Get(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)
	tree, err := h.uc.Execute(c.Request.Context(), auth, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		respondBackendError(c, "could not load profile")
		return
	}
	c.JSON(http.StatusOK, tree)
}
