package assets

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// assetMutator is the narrow interface mutation handlers depend on.
type assetMutator interface {
	CreateAsset(ctx context.Context, authorization string, body map[string]any) (*Asset, error)
	UpdateAsset(ctx context.Context, authorization, id string, body map[string]any) (*Asset, error)
	DeleteAsset(ctx context.Context, authorization, id string, force bool) error
	GetAsset(ctx context.Context, authorization, id string) (*Asset, error)
	List(ctx context.Context, authorization string, params ListParams) (*ListResult, error)
}

type CreateHandler struct {
	client assetMutator
}

func NewCreateHandler(client assetMutator) *CreateHandler {
	return &CreateHandler{client: client}
}

func (h *CreateHandler) Post(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	var body map[string]any
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		body = map[string]any{}
	} else if err := json.Unmarshal(raw, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid JSON body"}})
		return
	}

	_, err = h.client.CreateAsset(c.Request.Context(), auth, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			modal := BuildCreateModal(params, lang, be.Message)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: "assets-modal-slot",
				Tree:     &modal,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not create asset"}})
		return
	}

	respondPostMutation(c, h.client, params, lang, i18n.T(lang, "assets.create.success"))
}

// respondPostMutation rebuilds assets-root with fresh list + empty modal slot + success feedback.
func respondPostMutation(c *gin.Context, client assetMutator, params ListParams, lang, successMsg string) {
	res, err := client.List(c.Request.Context(), c.GetHeader("Authorization"), params)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh list"}})
		return
	}
	section := BuildAssetsSection(res, params, lang)
	modalSlot := components.Column("assets-modal-slot")
	root := components.ColumnWithGap("assets-root", "lg", section, modalSlot)
	fb := components.Snackbar("feedback", successMsg, "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "assets-root",
		Tree:     &root,
		Feedback: &fb,
	})
}
