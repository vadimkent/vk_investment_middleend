package trades

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestRespondTradeFetchError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantWrote  bool
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{name: "nil", err: nil, wantWrote: false},
		{name: "unauthorized", err: ErrUnauthorized, wantWrote: true, wantStatus: http.StatusUnauthorized},
		{name: "not_found", err: ErrTradeNotFound, wantWrote: true, wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND"},
		{name: "backend_wrapped", err: fmt.Errorf("%w: status 500", ErrBackend), wantWrote: true, wantStatus: http.StatusBadGateway, wantCode: "BACKEND_ERROR", wantMsg: "could not load trade"},
		{name: "other", err: errors.New("boom"), wantWrote: true, wantStatus: http.StatusBadGateway, wantCode: "BACKEND_ERROR", wantMsg: "could not load trade"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newTestContext()
			wrote := respondTradeFetchError(c, tc.err, "could not load trade")
			assert.Equal(t, tc.wantWrote, wrote)
			if !tc.wantWrote {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Empty(t, w.Body.String())
				return
			}
			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantStatus == http.StatusUnauthorized {
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				assert.Equal(t, "unauthorized", body["error"])
				assert.Equal(t, "/login", body["redirect"])
				return
			}
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			errObj, ok := body["error"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tc.wantCode, errObj["code"])
			if tc.wantMsg != "" {
				assert.Equal(t, tc.wantMsg, errObj["message"])
			}
		})
	}
}

func TestRespondCatalogFetchError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantWrote  bool
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{name: "nil", err: nil, wantWrote: false},
		{name: "unauthorized", err: assetscatalog.ErrUnauthorized, wantWrote: true, wantStatus: http.StatusUnauthorized},
		{name: "backend_wrapped", err: fmt.Errorf("%w: status 500", assetscatalog.ErrBackend), wantWrote: true, wantStatus: http.StatusBadGateway, wantCode: "BACKEND_ERROR", wantMsg: "could not load assets"},
		{name: "other", err: errors.New("kaboom"), wantWrote: true, wantStatus: http.StatusBadGateway, wantCode: "BACKEND_ERROR", wantMsg: "could not load assets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newTestContext()
			wrote := respondCatalogFetchError(c, tc.err, "could not load assets")
			assert.Equal(t, tc.wantWrote, wrote)
			if !tc.wantWrote {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Empty(t, w.Body.String())
				return
			}
			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantStatus == http.StatusUnauthorized {
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				assert.Equal(t, "unauthorized", body["error"])
				assert.Equal(t, "/login", body["redirect"])
				return
			}
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			errObj, ok := body["error"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tc.wantCode, errObj["code"])
			assert.Equal(t, tc.wantMsg, errObj["message"])
		})
	}
}

func TestRespondBadRequest(t *testing.T) {
	c, w := newTestContext()
	respondBadRequest(c, "missing id")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BAD_REQUEST", errObj["code"])
	assert.Equal(t, "missing id", errObj["message"])
}
