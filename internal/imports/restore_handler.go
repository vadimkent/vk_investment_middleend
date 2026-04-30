package imports

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/project/vk-investment-middleend/internal/components"
)

const restoreMaxBytes = 10 * 1024 * 1024

type RestoreHandler struct {
	client *Client
}

func NewRestoreHandler(c *Client) *RestoreHandler { return &RestoreHandler{client: c} }

func (h *RestoreHandler) Post(c *gin.Context) {
	lang := resolveLang(c)

	if err := c.Request.ParseMultipartForm(restoreMaxBytes); err != nil {
		writeRestoreError(c, lang, "Could not read the upload.", "")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeRestoreError(c, lang, "Missing file.", "")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, restoreMaxBytes+1))
	if err != nil {
		writeRestoreError(c, lang, "Failed to read upload.", header.Filename)
		return
	}
	if len(content) == 0 {
		writeRestoreError(c, lang, "The uploaded file is empty.", header.Filename)
		return
	}
	if int64(len(content)) > restoreMaxBytes {
		writeRestoreError(c, lang, "File exceeds the 10 MB limit.", header.Filename)
		return
	}

	res, err := h.client.Restore(c.Request.Context(), resolveAuth(c), content)
	if err != nil {
		var be *BackendError
		if errors.As(err, &be) {
			writeRestoreError(c, lang, be.Message, header.Filename)
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
			return
		}
		writeRestoreError(c, lang, "Restore failed.", header.Filename)
		return
	}

	tree := BuildRestoreCardSuccess(lang, res)
	c.JSON(http.StatusOK, components.ReplaceResponse("restore-card", tree, nil))
}

func writeRestoreError(c *gin.Context, lang, message, prefillFilename string) {
	tree := BuildRestoreCardIdle(lang, message, prefillFilename)
	fb := components.Snackbar("feedback", message, "error")
	c.JSON(http.StatusOK, components.ReplaceResponse("restore-card", tree, &fb))
}

// RestoreIdleHandler emits the idle subtree of restore-card. Used by the
// "Restore another file" button on the success state.
type RestoreIdleHandler struct{}

func NewRestoreIdleHandler() *RestoreIdleHandler { return &RestoreIdleHandler{} }

func (h *RestoreIdleHandler) Get(c *gin.Context) {
	lang := resolveLang(c)
	tree := BuildRestoreCardIdle(lang, "", "")
	c.JSON(http.StatusOK, components.ReplaceResponse("restore-card", tree, nil))
}
