package snapshots

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// snapshotUpdater is the narrow interface the Update handler depends on.
// *Client (see mutate_client.go) satisfies it.
type snapshotUpdater interface {
	UpdateSnapshot(ctx context.Context, authorization, id string, body map[string]any) (*Snapshot, error)
}

// UpdateHandler serves PATCH /actions/snapshots/:id: re-fetches the original
// snapshot, diffs it against the submitted wizard form to build a minimal PATCH
// body, calls the backend, and either rebuilds the snapshots screen on success
// or re-renders the Edit wizard with an inline error on backend validation failure.
type UpdateHandler struct {
	updater snapshotUpdater
	getter  snapshotGetter
	uc      *GetUseCase
	catalog catalogFetcher
}

func NewUpdateHandler(updater snapshotUpdater, getter snapshotGetter, uc *GetUseCase, catalog catalogFetcher) *UpdateHandler {
	return &UpdateHandler{updater: updater, getter: getter, uc: uc, catalog: catalog}
}

// Patch handles the edit-snapshot wizard submission.
func (h *UpdateHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondBadRequest(c, "missing id")
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		respondBadRequest(c, "invalid id")
		return
	}

	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	// Re-fetch the original snapshot as the baseline for diff.
	original, err := h.getter.GetSnapshot(c.Request.Context(), auth, id)
	if respondSnapshotFetchError(c, err, "could not load snapshot") {
		return
	}

	submitted, err := parseJSONBody(c)
	if err != nil {
		respondBadRequest(c, "invalid JSON body")
		return
	}

	diffBody := buildSnapshotUpdateDiff(submitted, original)

	if len(diffBody) > 0 {
		_, err = h.updater.UpdateSnapshot(c.Request.Context(), auth, id, diffBody)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				shared.RespondUnauthorized(c, "/login")
				return
			}
			if errors.Is(err, ErrSnapshotNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND"}})
				return
			}
			var be *BackendValidationError
			if errors.As(err, &be) {
				cat, catErr := h.catalog.List(c.Request.Context(), auth)
				if respondCatalogFetchError(c, catErr, "could not load assets") {
					return
				}
				// In edit mode all validation errors go to "summary" —
				// the info step is read-only so FUTURE_DATED_SNAPSHOT is
				// not actionable there; "summary" is a safe universal default.
				initialStepID := "summary"
				wizard := BuildEditWizard(original, cat, params, lang, be.Message, initialStepID, nil)
				c.JSON(http.StatusOK, components.ActionResponse{
					Action:   "replace",
					TargetID: ModalSlotID,
					Tree:     &wizard,
				})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not update snapshot"}})
			return
		}
	}

	// Success (or no-op): rebuild the screen tree and attach a success snackbar.
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
	fb := components.Snackbar("feedback", i18n.T(lang, "snapshots.edit.success"), "success")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ScreenID,
		Tree:     &tree,
		Feedback: &fb,
	})
}

// buildSnapshotUpdateDiff compares the submitted wizard body against the
// original snapshot and returns a PATCH body with only changed fields.
//
// Rules:
//   - notes: included if the submitted value differs from original.Notes.
//     Empty string is a valid "clear" value (BE accepts it to clear notes).
//   - entries: computed via diffEntries; only new or changed entries are
//     included. The BE does not support removing entries via PATCH, so entries
//     present in the original but absent in the submission are ignored.
func buildSnapshotUpdateDiff(submitted map[string]any, original *Snapshot) map[string]any {
	body := map[string]any{}

	// Notes diff: include whenever the submitted value differs from original.
	// If both are empty, skip. If user cleared notes (submitted ""), include it.
	if v, ok := submitted["notes"]; ok {
		s, _ := v.(string)
		if s != original.Notes {
			body["notes"] = s
		}
	}

	// Entries diff.
	submittedEntries := parseWizardEntries(submitted)
	diffed := diffEntries(original.Entries, submittedEntries)
	if len(diffed) > 0 {
		body["entries"] = diffed
	}

	return body
}

// diffEntries compares submitted wizard entries against original snapshot
// entries and returns only entries that are new or whose values changed.
//
// Original entries absent from the submission are left untouched (not
// included in the diff) — the BE does not allow entry removal via PATCH.
func diffEntries(original []Entry, submitted []wizardEntry) []map[string]any {
	// Index original entries by AssetID for O(1) lookup.
	origByID := make(map[string]Entry, len(original))
	for _, e := range original {
		origByID[e.AssetID] = e
	}

	var result []map[string]any
	for _, e := range submitted {
		beEntry := buildBeEntry(e)
		if beEntry == nil {
			// No value submitted — skip.
			continue
		}

		orig, exists := origByID[e.AssetID]
		if !exists {
			// New asset not in the original snapshot — always include.
			result = append(result, beEntry)
			continue
		}

		// Existing asset — compare values. BE null is stored as empty string in
		// domain Entry; submitted empty string means "no value". Treat both as "".
		bePrice, _ := beEntry["current_price"].(string)
		beOverride, _ := beEntry["current_value_override"].(string)

		changed := bePrice != orig.CurrentPrice || beOverride != orig.CurrentValueOverride
		if changed {
			result = append(result, beEntry)
		}
	}

	return result
}
