package analysis

import (
	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	client *Client
}

func NewStreamHandler(c *Client) *StreamHandler { return &StreamHandler{client: c} }

func (h *StreamHandler) Get(c *gin.Context) {
	focus := c.Query("focus")
	resp, err := h.client.StreamSession(c.Request.Context(), resolveAuth(c), focus)
	if err != nil {
		handleStreamError(c, err)
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}
