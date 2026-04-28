package imports

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

type ResolveGapsHandler struct {
	client *Client
}

func NewResolveGapsHandler(c *Client) *ResolveGapsHandler { return &ResolveGapsHandler{client: c} }

func (h *ResolveGapsHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	id := c.Param("id")

	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	resolutions := parseGapResolutions(c.Request.PostForm)

	sess, err := h.client.ResolveGaps(c.Request.Context(), resolveAuth(c), id, resolutions)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			// We don't have the previous session here — re-fetch by no-op
			// resolve with empty resolutions is fragile. Instead, return a
			// minimal modal carrying the error banner; the user can retry.
			fb := components.Snackbar("feedback", be.Message, "error")
			c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
			return
		}
		if errors.Is(err, ErrSessionNotFound) {
			tree := BuildRoot(lang)
			fb := components.Snackbar("feedback", i18n.T(lang, "import.session_expired"), "warning")
			c.JSON(http.StatusOK, components.ReplaceResponse("import-root", tree, &fb))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		fb := components.Snackbar("feedback", i18n.T(lang, "import.failure_generic"), "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildReviewModal(lang, sess, "")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-modal-slot", tree, nil))
}

// parseGapResolutions extracts resolutions[<gap_id>]=value pairs from the form.
func parseGapResolutions(form map[string][]string) []GapResolution {
	out := make([]GapResolution, 0)
	const prefix = "resolutions["
	const suffix = "]"
	for key, vals := range form {
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
			continue
		}
		gapID := key[len(prefix) : len(key)-len(suffix)]
		if gapID == "" {
			continue
		}
		val := ""
		if len(vals) > 0 {
			val = strings.TrimSpace(vals[0])
		}
		if val == "" {
			continue
		}
		out = append(out, GapResolution{GapID: gapID, Value: val})
	}
	return out
}
