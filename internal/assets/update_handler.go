package assets

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

type UpdateHandler struct {
	client assetMutator
}

func NewUpdateHandler(client assetMutator) *UpdateHandler {
	return &UpdateHandler{client: client}
}

func (h *UpdateHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id is required"}})
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid body"}})
		return
	}
	var body map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid JSON"}})
			return
		}
	} else {
		body = map[string]any{}
	}

	_, err = h.client.UpdateAsset(c.Request.Context(), auth, id, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if errors.Is(err, ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND"}})
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			// Re-fetch asset to repopulate the edit modal.
			a, gerr := h.client.GetAsset(c.Request.Context(), auth, id)
			if gerr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refetch asset"}})
				return
			}
			modal := BuildEditModal(a, params, lang, be.Message)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: "assets-modal-slot",
				Tree:     &modal,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not update asset"}})
		return
	}

	respondPostMutation(c, h.client, params, lang, i18n.T(lang, "assets.edit.success"))
}
