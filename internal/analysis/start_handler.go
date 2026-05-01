package analysis

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

const maxFocusLen = 500

type StartHandler struct{}

func NewStartHandler() *StartHandler { return &StartHandler{} }

func (h *StartHandler) Post(c *gin.Context) {
	lang := resolveLang(c)
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	focus := strings.TrimSpace(c.Request.PostForm.Get("focus"))

	if len([]rune(focus)) > maxFocusLen {
		// Re-emit the form with the (truncated for display) focus + inline error.
		formTree := BuildStartState(lang, focus, i18n.T(lang, "analysis.error.focus_too_long"))
		// The replace target is the form id, so we replace just the form, not
		// the whole content area. The tree we send is the card+form returned
		// by BuildStartState — but the FE expects to find the form as the
		// root of what it replaces. Strategy: send only the form subtree.
		// BuildStartState returns the card; we want only the form inside.
		formChild := extractStartForm(formTree)
		c.JSON(http.StatusOK, components.ReplaceResponse("analysis-start-form", formChild, nil))
		return
	}

	tree := BuildContentChat(lang, focus)
	c.JSON(http.StatusOK, components.ReplaceResponse("analysis-content", tree, nil))
}

// extractStartForm digs into the BuildStartState card to return the inner
// Form component (id="analysis-start-form"). Used to re-emit only the form
// when the replace target is the form id.
func extractStartForm(card components.Component) components.Component {
	if card.Type == "form" && card.ID == "analysis-start-form" {
		return card
	}
	for _, ch := range card.Children {
		if found := extractStartForm(ch); found.Type == "form" {
			return found
		}
	}
	return card // fallback: send whatever we have
}
