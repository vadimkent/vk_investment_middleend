package portfolio

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// positionsFetcherWithInclude is the narrow interface the include-closed
// handler needs. Satisfied by *Client.
type positionsFetcherWithInclude interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
}

type IncludeClosedHandler struct {
	client positionsFetcherWithInclude
}

func NewIncludeClosedHandler(client positionsFetcherWithInclude) *IncludeClosedHandler {
	return &IncludeClosedHandler{client: client}
}

type includeClosedRequest struct {
	IncludeClosed *bool `json:"include_closed"`
}

// Post handles POST /actions/portfolio/include_closed.
func (h *IncludeClosedHandler) Post(c *gin.Context) {
	var req includeClosedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid request body"}})
		return
	}
	if req.IncludeClosed == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "include_closed is required"}})
		return
	}

	auth := c.GetHeader("Authorization")
	resp, err := h.client.GetPositions(c.Request.Context(), auth, *req.IncludeClosed, false, false)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		return
	}
	positions := resp.Positions
	SortPositions(positions)

	lang := parseLang(c)
	tree := BuildPositionsTable(positions, lang, time.Now())
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "positions-table-card",
		Tree:     &tree,
	})
}
