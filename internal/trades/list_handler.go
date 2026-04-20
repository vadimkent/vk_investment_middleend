package trades

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type ListHandler struct{ uc *GetUseCase }

func NewListHandler(uc *GetUseCase) *ListHandler { return &ListHandler{uc: uc} }

// Get serves GET /actions/trades/list: validates query params, calls the use
// case for the list subtree, and returns an ActionResponse that replaces the
// trades-section node. Trades or catalog ErrUnauthorized → 401 with redirect;
// any other backend error → 502 BACKEND_ERROR.
func (h *ListHandler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	section, err := h.uc.ExecuteSection(c.Request.Context(), auth, params, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) || errors.Is(err, assetscatalog.ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load trades"}})
		return
	}
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: SectionID,
		Tree:     &section,
	})
}
