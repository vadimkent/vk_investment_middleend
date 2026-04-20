package trades

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

// tradeCreator is the narrow interface the Create handler depends on. *Client
// (see mutate_client.go) satisfies it.
type tradeCreator interface {
	CreateTrade(ctx context.Context, authorization string, body map[string]any) (*Trade, error)
}

// CreateHandler serves POST /actions/trades/create: parses the submitted
// form, posts the trade to the backend, and either rebuilds the full trades
// screen on success or re-renders the Create modal with an inline error on
// backend validation failure.
type CreateHandler struct {
	creator tradeCreator
	uc      *GetUseCase
	catalog catalogFetcher
}

func NewCreateHandler(creator tradeCreator, uc *GetUseCase, catalog catalogFetcher) *CreateHandler {
	return &CreateHandler{creator: creator, uc: uc, catalog: catalog}
}

// Post handles the create-trade form submission.
func (h *CreateHandler) Post(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	body := buildCreateBody(c)

	_, err = h.creator.CreateTrade(c.Request.Context(), auth, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			// Re-fetch the catalog so the replayed modal still has the asset
			// options populated.
			cat, catErr := h.catalog.List(c.Request.Context(), auth)
			if respondCatalogFetchError(c, catErr, "could not load assets") {
				return
			}
			modal := BuildCreateModal(cat, params, lang, be.Message)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: ModalSlotID,
				Tree:     &modal,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not create trade"}})
		return
	}

	// Success — rebuild the screen tree and attach a success snackbar.
	tree, err := h.uc.Execute(c.Request.Context(), auth, params, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if respondCatalogFetchError(c, err, "could not refresh trades") {
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh trades"}})
		return
	}
	fb := components.Snackbar("feedback", i18n.T(lang, "trades.create.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}

// buildCreateBody maps the submitted form to the backend's trade body.
// Rules:
//   - asset_id, trade_type, quantity, price_per_unit, date pass through verbatim.
//   - fees defaults to "0" when empty.
//   - date is normalised: a bare "YYYY-MM-DD" becomes "YYYY-MM-DDT00:00:00Z";
//     anything else (already RFC3339) passes through.
//   - source is always "MANUAL" — NEVER taken from the form.
//   - notes is included only when non-empty.
func buildCreateBody(c *gin.Context) map[string]any {
	body := map[string]any{
		"asset_id":       c.PostForm("asset_id"),
		"trade_type":     c.PostForm("trade_type"),
		"quantity":       c.PostForm("quantity"),
		"price_per_unit": c.PostForm("price_per_unit"),
		"source":         "MANUAL",
	}

	fees := c.PostForm("fees")
	if fees == "" {
		fees = "0"
	}
	body["fees"] = fees

	body["date"] = normaliseDate(c.PostForm("date"))

	if notes := c.PostForm("notes"); notes != "" {
		body["notes"] = notes
	}

	return body
}

// normaliseDate converts a bare "YYYY-MM-DD" input to its RFC3339 UTC form;
// any other (already-timestamped) value is returned unchanged.
func normaliseDate(raw string) string {
	if len(raw) == 10 && strings.Count(raw, "-") == 2 {
		return raw + "T00:00:00Z"
	}
	return raw
}
