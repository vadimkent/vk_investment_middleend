package trades

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// tradeFetcherUpdater is the narrow interface the Update handler depends on:
// it needs both a GetTrade (to compute the diff and to replay the Edit modal
// on validation error) and an UpdateTrade (to send the PATCH). The concrete
// *Client satisfies it.
type tradeFetcherUpdater interface {
	GetTrade(ctx context.Context, authorization, id string) (*Trade, error)
	UpdateTrade(ctx context.Context, authorization, id string, body map[string]any) (*Trade, error)
}

// UpdateHandler serves PATCH /actions/trades/:id: fetches the original trade,
// diffs it against the submitted form to build a minimal PATCH body, calls
// the backend, and either rebuilds the trades screen on success or re-renders
// the Edit modal on backend validation failure.
type UpdateHandler struct {
	client  tradeFetcherUpdater
	uc      *GetUseCase
	catalog catalogFetcher
}

func NewUpdateHandler(client tradeFetcherUpdater, uc *GetUseCase, catalog catalogFetcher) *UpdateHandler {
	return &UpdateHandler{client: client, uc: uc, catalog: catalog}
}

// Patch handles the edit-trade form submission.
func (h *UpdateHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondBadRequest(c, "missing id")
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	original, err := h.client.GetTrade(c.Request.Context(), auth, id)
	if respondTradeFetchError(c, err, "could not load trade") {
		return
	}

	body := buildUpdateDiff(c, *original)

	if len(body) > 0 {
		_, err = h.client.UpdateTrade(c.Request.Context(), auth, id, body)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				shared.RespondUnauthorized(c, "/login")
				return
			}
			var be *BackendValidationError
			if errors.As(err, &be) {
				cat, catErr := h.catalog.List(c.Request.Context(), auth)
				if respondCatalogFetchError(c, catErr, "could not load assets") {
					return
				}
				modal := BuildEditModal(*original, cat, params, lang, be.Message)
				c.JSON(http.StatusOK, components.ActionResponse{
					Action:   "replace",
					TargetID: ModalSlotID,
					Tree:     &modal,
				})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not update trade"}})
			return
		}
	}

	// Success (or no-op): rebuild the screen and attach a success snackbar.
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
	fb := components.Snackbar("feedback", i18n.T(lang, "trades.edit.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}

// buildUpdateDiff compares the submitted form against the original trade and
// returns a PATCH body containing only the mutable fields whose submitted
// value differs from the original. The `date` and `source` form fields (if
// submitted at all) are silently ignored — they're immutable per the backend
// contract. `fees` is canonicalized: empty string and "0" are equivalent.
// `notes` empty string is treated as "no notes" (absent == empty).
func buildUpdateDiff(c *gin.Context, original Trade) map[string]any {
	body := map[string]any{}

	if v := c.PostForm("asset_id"); v != original.AssetID {
		body["asset_id"] = v
	}
	if v := c.PostForm("trade_type"); v != original.TradeType {
		body["trade_type"] = v
	}
	if v := c.PostForm("quantity"); v != original.Quantity {
		body["quantity"] = v
	}
	if v := c.PostForm("price_per_unit"); v != original.PricePerUnit {
		body["price_per_unit"] = v
	}

	// Fees: canonicalize "" to "0" on both sides so ""↔"0" are equivalent.
	submittedFees := canonicalizeFees(c.PostForm("fees"))
	originalFees := canonicalizeFees(original.Fees)
	if submittedFees != originalFees {
		body["fees"] = c.PostForm("fees")
	}

	// Notes: empty/absent are equivalent.
	if v := c.PostForm("notes"); v != original.Notes {
		body["notes"] = v
	}

	return body
}

func canonicalizeFees(v string) string {
	if v == "" {
		return "0"
	}
	return v
}
