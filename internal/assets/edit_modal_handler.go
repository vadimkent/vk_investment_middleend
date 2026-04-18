package assets

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// assetByIDFetcher is the narrow interface the edit/delete modal handlers need.
type assetByIDFetcher interface {
	GetAsset(ctx context.Context, authorization, id string) (*Asset, error)
}

type EditModalHandler struct {
	client assetByIDFetcher
}

func NewEditModalHandler(client assetByIDFetcher) *EditModalHandler {
	return &EditModalHandler{client: client}
}

func (h *EditModalHandler) Get(c *gin.Context) {
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

	modal := BuildEditModal(a, params, parseLang(c), "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-modal-slot",
		Tree:     &modal,
	})
}
