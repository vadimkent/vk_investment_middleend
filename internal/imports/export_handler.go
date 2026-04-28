package imports

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	client *Client
}

func NewExportHandler(c *Client) *ExportHandler { return &ExportHandler{client: c} }

func (h *ExportHandler) Get(c *gin.Context) {
	resp, err := h.client.ExportStream(c.Request.Context(), resolveAuth(c))
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.Data(http.StatusBadGateway, "text/plain; charset=utf-8", []byte("Export failed."))
		return
	}
	defer resp.Body.Close()

	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		c.Header("Content-Disposition", cd)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/csv; charset=utf-8"
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", contentType)
	_, _ = io.Copy(c.Writer, resp.Body)
}
