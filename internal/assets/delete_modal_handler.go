package assets

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type DeleteModalHandler struct {
	client assetByIDFetcher
}

func NewDeleteModalHandler(client assetByIDFetcher) *DeleteModalHandler {
	return &DeleteModalHandler{client: client}
}

func (h *DeleteModalHandler) Get(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id is required"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}

	a, err := h.client.GetAsset(c.Request.Context(), c.GetHeader("Authorization"), id)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "asset not found"}})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load asset"}})
		return
	}

	modal := BuildDeleteModal(a.ID, a.Ticker, params, parseLang(c), "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-modal-slot",
		Tree:     &modal,
	})
}
