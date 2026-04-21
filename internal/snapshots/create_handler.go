package snapshots

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// snapshotCreator is the narrow interface the Create handler depends on. *Client
// (see mutate_client.go) satisfies it.
type snapshotCreator interface {
	CreateSnapshot(ctx context.Context, authorization string, body map[string]any) (*Snapshot, error)
}

// CreateHandler serves POST /actions/snapshots/create: parses the wizard form
// submission, posts the snapshot to the backend, and either rebuilds the full
// snapshots screen on success or re-renders the Create wizard with an inline
// error on backend validation failure.
type CreateHandler struct {
	creator snapshotCreator
	uc      *GetUseCase
	catalog catalogFetcher
}

func NewCreateHandler(creator snapshotCreator, uc *GetUseCase, catalog catalogFetcher) *CreateHandler {
	return &CreateHandler{creator: creator, uc: uc, catalog: catalog}
}

// Post handles the create-snapshot wizard submission.
func (h *CreateHandler) Post(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	submitted, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}
	body := buildCreateBody(submitted)

	_, err = h.creator.CreateSnapshot(c.Request.Context(), auth, body)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		var be *BackendValidationError
		if errors.As(err, &be) {
			// Re-fetch the catalog so the replayed wizard still has asset options.
			cat, catErr := h.catalog.List(c.Request.Context(), auth)
			if respondCatalogFetchError(c, catErr, "could not load assets") {
				return
			}
			initialStepID := "summary"
			if be.Code == "FUTURE_DATED_SNAPSHOT" {
				initialStepID = "info"
			}
			wizard := BuildCreateWizard(cat, params, lang, be.Message, initialStepID)
			c.JSON(http.StatusOK, components.ActionResponse{
				Action:   "replace",
				TargetID: ModalSlotID,
				Tree:     &wizard,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not create snapshot"}})
		return
	}

	// Success — rebuild the screen tree and attach a success snackbar.
	tree, err := h.uc.Execute(c.Request.Context(), auth, params, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		if respondCatalogFetchError(c, err, "could not refresh snapshots") {
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not refresh snapshots"}})
		return
	}
	fb := components.Snackbar("feedback", i18n.T(lang, "snapshots.create.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}

// buildCreateBody transforms the raw wizard submission into the shape the
// backend expects.
//
// Rules:
//   - recorded_at is forwarded as-is (BE validates presence/format).
//   - notes is included only when non-empty.
//   - entries: for each parsed wizard entry, exactly one value field is sent:
//     - mode="price" + current_price non-empty  → {asset_id, current_price}
//     - mode="override" + current_value_override non-empty → {asset_id, current_value_override}
//     - mode="" + current_value_override non-empty (complex asset path) → {asset_id, current_value_override}
//     - all empty → entry dropped entirely (never sent as bare {asset_id}).
func buildCreateBody(submitted map[string]any) map[string]any {
	body := map[string]any{
		"recorded_at": asString(submitted, "recorded_at"),
	}

	if notes := asString(submitted, "notes"); notes != "" {
		body["notes"] = notes
	}

	entries := parseWizardEntries(submitted)
	beEntries := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		beEntry := buildBeEntry(e)
		if beEntry != nil {
			beEntries = append(beEntries, beEntry)
		}
	}
	body["entries"] = beEntries

	return body
}

// buildBeEntry converts one wizard entry into its backend representation.
// Returns nil when the entry carries no submitted value and must be dropped.
func buildBeEntry(e wizardEntry) map[string]any {
	switch {
	case e.Mode == "price" && e.CurrentPrice != "":
		return map[string]any{
			"asset_id":      e.AssetID,
			"current_price": e.CurrentPrice,
		}
	case e.CurrentValueOverride != "":
		// Covers mode="override" and the complex-asset path (mode="").
		return map[string]any{
			"asset_id":               e.AssetID,
			"current_value_override": e.CurrentValueOverride,
		}
	default:
		// No value submitted — drop this entry.
		return nil
	}
}
