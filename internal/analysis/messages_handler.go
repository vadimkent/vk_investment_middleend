package analysis

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MessagesHandler struct {
	client *Client
}

func NewMessagesHandler(c *Client) *MessagesHandler { return &MessagesHandler{client: c} }

type messageRequest struct {
	Content string `json:"content"`
}

func (h *MessagesHandler) Post(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "missing session id"}})
		return
	}
	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "missing required field: content"}})
		return
	}

	resp, err := h.client.AddMessage(c.Request.Context(), resolveAuth(c), id, req.Content)
	if err != nil {
		// Map ErrSessionNotFound through handleStreamError as a normal
		// pre-stream error: it'll synthesize an SSE error event with code
		// ANALYSIS_SESSION_NOT_FOUND-like content. To get the right code we
		// translate ErrSessionNotFound into a BackendError carrying the
		// standard code so downstream handlers don't need to know.
		if errors.Is(err, ErrSessionNotFound) {
			err = &BackendError{HTTPStatus: http.StatusNotFound, Code: "ANALYSIS_SESSION_NOT_FOUND", Message: "session not found"}
		}
		handleStreamError(c, err)
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}
