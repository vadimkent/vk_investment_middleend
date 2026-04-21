package snapshots

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

// snapshotAutoCreator is the narrow interface the Auto handler depends on.
// *Client (see mutate_client.go) satisfies it.
type snapshotAutoCreator interface {
	AutoSnapshot(ctx context.Context, authorization, notes string) (*AutoResult, error)
}

// AutoHandler serves POST /actions/snapshots/auto: triggers an auto-snapshot
// on the backend and, on success, rebuilds the full screen tree with an open
// edit wizard pre-populated with the new snapshot. Terminal backend failures
// (NO_PRICE_PROVIDERS_CONFIGURED, ALL_PROVIDERS_FAILED) surface as feedback-
// only snackbars without opening the wizard.
type AutoHandler struct {
	client  snapshotAutoCreator
	uc      *GetUseCase
	catalog catalogFetcher
}

// NewAutoHandler constructs an AutoHandler.
func NewAutoHandler(client snapshotAutoCreator, uc *GetUseCase, catalog catalogFetcher) *AutoHandler {
	return &AutoHandler{client: client, uc: uc, catalog: catalog}
}

// Post handles the auto-snapshot action.
func (h *AutoHandler) Post(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	result, err := h.client.AutoSnapshot(c.Request.Context(), auth, "")
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			switch be.Code {
			case "NO_PRICE_PROVIDERS_CONFIGURED":
				fb := components.Snackbar("feedback", i18n.T(lang, "snapshots.auto.no_providers"), "warning")
				c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
				return
			case "ALL_PROVIDERS_FAILED":
				fb := components.Snackbar("feedback", i18n.T(lang, "snapshots.auto.all_failed"), "error")
				c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
				return
			default:
				c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "auto-snapshot failed"}})
				return
			}
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "auto-snapshot failed"}})
		return
	}

	// Fetch the refreshed list result and catalog for custom composition.
	res, cat, err := h.uc.ExecuteListResult(c.Request.Context(), auth, params)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh snapshots"}})
		return
	}

	// Compose the wizard banner.
	bannerMsg := i18n.T(lang, "snapshots.auto.banner")
	if len(result.Warnings) > 0 {
		tickers := make([]string, 0, len(result.Warnings))
		for _, w := range result.Warnings {
			tickers = append(tickers, w.Ticker)
		}
		bannerMsg += "\n\n" + i18n.T(lang, "snapshots.auto.warnings_title") + ": " + strings.Join(tickers, ", ")
	}
	banner := &components.WizardBanner{Variant: "info", Message: bannerMsg, Dismissible: true}

	// Build the edit wizard pre-populated with the new snapshot.
	wizard := BuildEditWizard(&result.Snapshot, cat, params, lang, "", "", banner)

	// Build the full screen tree with the wizard injected into the modal slot.
	tree := BuildScreenWithModal(res, cat, params, lang, wizard)

	fb := components.Snackbar("feedback", i18n.T(lang, "snapshots.auto.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}
