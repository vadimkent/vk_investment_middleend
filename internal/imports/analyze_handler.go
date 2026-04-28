package imports

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
)

const analyzeMaxBytes = 5 * 1024 * 1024

type AnalyzeHandler struct {
	client *Client
}

func NewAnalyzeHandler(c *Client) *AnalyzeHandler { return &AnalyzeHandler{client: c} }

func (h *AnalyzeHandler) Post(c *gin.Context) {
	lang := resolveLang(c)

	if err := c.Request.ParseMultipartForm(analyzeMaxBytes); err != nil {
		writeAIImportError(c, lang, "Upload exceeds the size limit.")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeAIImportError(c, lang, "Missing file.")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, analyzeMaxBytes+1))
	if err != nil {
		writeAIImportError(c, lang, "Failed to read upload.")
		return
	}
	if int64(len(content)) > analyzeMaxBytes {
		writeAIImportError(c, lang, "File exceeds the 5 MB limit.")
		return
	}

	mediaType := header.Header.Get("Content-Type")
	hint := c.Request.FormValue("hint")
	prefillFilename := header.Filename

	sess, err := h.client.StartSession(c.Request.Context(), resolveAuth(c), content, mediaType, hint)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			tree := BuildAIImportCardIdle(lang, be.Message, prefillFilename, hint)
			c.JSON(http.StatusOK, components.ReplaceResponse("ai-import-card", tree, nil))
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		// network / 5xx → snackbar error, leave the card as-is (no replace).
		fb := components.Snackbar("feedback", "Import failed. Please try again.", "error")
		c.JSON(http.StatusOK, components.ActionResponse{Action: "none", Feedback: &fb})
		return
	}

	tree := BuildReviewModal(lang, sess, "")
	c.JSON(http.StatusOK, components.ReplaceResponse("import-modal-slot", tree, nil))
}

func writeAIImportError(c *gin.Context, lang, message string) {
	hint := c.Request.FormValue("hint")
	tree := BuildAIImportCardIdle(lang, message, "", hint)
	c.JSON(http.StatusOK, components.ReplaceResponse("ai-import-card", tree, nil))
}
