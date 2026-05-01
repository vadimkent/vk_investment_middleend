package analysis

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newGinRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestHandleStreamError_Unauthorized(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, ErrUnauthorized)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unauthorized") {
		t.Fatalf("expected unauthorized envelope, got: %s", w.Body.String())
	}
}

func TestHandleStreamError_BackendError429EmitsRateLimited(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, &BackendError{HTTPStatus: http.StatusTooManyRequests, Code: "RATE_LIMITED", Message: "slow"})
	body := w.Body.String()
	if !strings.Contains(body, `event: error`) {
		t.Fatalf("expected SSE error event, got: %s", body)
	}
	if !strings.Contains(body, `"code":"RATE_LIMITED"`) {
		t.Fatalf("expected RATE_LIMITED code, got: %s", body)
	}
}

func TestHandleStreamError_BackendError5xxEmitsProviderUnavailable(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, &BackendError{HTTPStatus: http.StatusBadGateway, Code: "", Message: "bad"})
	body := w.Body.String()
	if !strings.Contains(body, `"code":"AI_PROVIDER_UNAVAILABLE"`) {
		t.Fatalf("expected AI_PROVIDER_UNAVAILABLE, got: %s", body)
	}
}

func TestHandleStreamError_BackendError4xxPassesThroughCode(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, &BackendError{HTTPStatus: http.StatusBadRequest, Code: "ANALYSIS_FOCUS_TOO_LONG", Message: "too long"})
	body := w.Body.String()
	if !strings.Contains(body, `"code":"ANALYSIS_FOCUS_TOO_LONG"`) {
		t.Fatalf("expected pass-through code, got: %s", body)
	}
}

func TestHandleStreamError_OtherErrorEmitsInternal(t *testing.T) {
	c, w := newGinRecorder()
	handleStreamError(c, errors.New("network ded"))
	body := w.Body.String()
	if !strings.Contains(body, `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("expected INTERNAL_ERROR, got: %s", body)
	}
}
