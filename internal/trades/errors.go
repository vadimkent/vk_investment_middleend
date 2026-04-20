package trades

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// respondTradeFetchError maps an error returned by the trades Client to a gin
// response. Returns true if a response was written (caller should return);
// false when err == nil.
//
// Error precedence:
//   ErrUnauthorized          -> 401 redirect
//   ErrTradeNotFound         -> 404 NOT_FOUND
//   *BackendValidationError  -> NOT handled here (callers replay modals)
//   anything else            -> 502 BACKEND_ERROR with the given message
func respondTradeFetchError(c *gin.Context, err error, backendMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrUnauthorized) {
		shared.RespondUnauthorized(c, "/login")
		return true
	}
	if errors.Is(err, ErrTradeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND"}})
		return true
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": backendMessage}})
	return true
}

// respondCatalogFetchError maps an error returned by the assets catalog to a
// gin response. Same return semantics as respondTradeFetchError.
//
// Error precedence:
//   assetscatalog.ErrUnauthorized -> 401 redirect
//   anything else                 -> 502 BACKEND_ERROR
func respondCatalogFetchError(c *gin.Context, err error, backendMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, assetscatalog.ErrUnauthorized) {
		shared.RespondUnauthorized(c, "/login")
		return true
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": backendMessage}})
	return true
}

// respondBadRequest writes a 400 BAD_REQUEST response with the given message.
func respondBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": message}})
}
