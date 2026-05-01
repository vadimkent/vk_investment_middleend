package analysis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setSSEHeaders writes the standard SSE response headers on the gin writer.
// Must be called before any body bytes are written.
func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

// writeSSEErrorEvent serializes {code, message} as a single SSE error event.
// Caller is expected to have already called setSSEHeaders + Status(200).
func writeSSEErrorEvent(c *gin.Context, code, message string) {
	payload, _ := json.Marshal(map[string]string{"code": code, "message": message})
	fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", payload)
}

// handleStreamError converts a pre-stream client error into either a 401 JSON
// envelope (auth — same shape as the rest of the project) or a single SSE
// error event (everything else). Mapping:
//   - ErrUnauthorized → 401 {"error":"unauthorized","redirect":"/login"}
//   - BackendError 429 → SSE error with code "RATE_LIMITED"
//   - BackendError 5xx → SSE error with code "AI_PROVIDER_UNAVAILABLE"
//     (or pass-through if the BE returned its own code)
//   - BackendError other → SSE error with the BE's code (or "INTERNAL_ERROR")
//   - any other error  → SSE error with code "INTERNAL_ERROR"
func handleStreamError(c *gin.Context, err error) {
	if errors.Is(err, ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "redirect": "/login"})
		return
	}

	setSSEHeaders(c)
	c.Status(http.StatusOK)

	var be *BackendError
	if errors.As(err, &be) {
		switch {
		case be.HTTPStatus == http.StatusTooManyRequests:
			writeSSEErrorEvent(c, ifEmptyCode(be.Code, "RATE_LIMITED"), be.Message)
		case be.HTTPStatus >= 500:
			writeSSEErrorEvent(c, ifEmptyCode(be.Code, "AI_PROVIDER_UNAVAILABLE"), be.Message)
		default:
			writeSSEErrorEvent(c, ifEmptyCode(be.Code, "INTERNAL_ERROR"), be.Message)
		}
	} else {
		writeSSEErrorEvent(c, "INTERNAL_ERROR", err.Error())
	}
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

// ifEmptyCode returns code unless empty, in which case fallback is returned.
func ifEmptyCode(code, fallback string) string {
	if code == "" {
		return fallback
	}
	return code
}

// proxySSE bypasses upstream's SSE response body to the gin context. Sets the
// SSE response headers, then streams body chunks with periodic flush. On
// mid-stream upstream error (network drop), emits a synthetic INTERNAL_ERROR
// event before closing.
func proxySSE(c *gin.Context, upstream *http.Response) {
	setSSEHeaders(c)
	c.Status(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// Defensive: fall back to a plain copy without flush. Should not
		// happen with gin's writer.
		_, _ = io.Copy(c.Writer, upstream.Body)
		return
	}
	buf := make([]byte, 4096)
	for {
		n, err := upstream.Body.Read(buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return // client gone
			}
			flusher.Flush()
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			writeSSEErrorEvent(c, "INTERNAL_ERROR", "connection lost")
			flusher.Flush()
			return
		}
	}
}
