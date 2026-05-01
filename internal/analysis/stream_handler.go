package analysis

import (
	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	client *Client
	fake   bool
}

func NewStreamHandler(c *Client) *StreamHandler { return &StreamHandler{client: c} }

// NewStreamHandlerWithFake returns a handler that emits a synthetic SSE
// response (canned markdown chunked into deltas) instead of calling the
// backend. Used in dev to avoid burning AI provider tokens. Toggled via
// ANALYSIS_FAKE=true in the env.
func NewStreamHandlerWithFake(c *Client, fake bool) *StreamHandler {
	return &StreamHandler{client: c, fake: fake}
}

func (h *StreamHandler) Get(c *gin.Context) {
	if h.fake {
		streamFakeAnalysis(c)
		return
	}
	focus := c.Query("focus")
	resp, err := h.client.StreamSession(c.Request.Context(), resolveAuth(c), focus)
	if err != nil {
		handleStreamError(c, err)
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}
