package snapshots

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// snapshotGetter is the narrow interface the edit/delete handlers use to fetch
// a single snapshot by id. *Client satisfies it.
type snapshotGetter interface {
	GetSnapshot(ctx context.Context, authorization, id string) (*Snapshot, error)
}

// respondSnapshotFetchError maps an error returned by the snapshots Client to a
// gin response. Returns true if a response was written (caller should return);
// false when err == nil.
//
// Error precedence:
//
//	ErrUnauthorized      → 401 redirect
//	ErrSnapshotNotFound  → 404 NOT_FOUND
//	anything else        → 502 BACKEND_ERROR with the given message
func respondSnapshotFetchError(c *gin.Context, err error, backendMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrUnauthorized) {
		shared.RespondUnauthorized(c, "/login")
		return true
	}
	if errors.Is(err, ErrSnapshotNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND"}})
		return true
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": backendMessage}})
	return true
}

// respondCatalogFetchError maps an error returned by the assets catalog to a
// gin response. Same return semantics as respondSnapshotFetchError.
//
// Error precedence:
//
//	assetscatalog.ErrUnauthorized → 401 redirect
//	anything else                 → 502 BACKEND_ERROR
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
