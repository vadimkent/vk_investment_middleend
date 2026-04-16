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

// liveFetcher is the narrow interface the live handler needs.
type liveFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
	GetEvolutionLast(ctx context.Context, authorization string, n int) ([]EvolutionPoint, error)
}

type LiveHandler struct {
	client liveFetcher
}

func NewLiveHandler(client liveFetcher) *LiveHandler {
	return &LiveHandler{client: client}
}

func (h *LiveHandler) Get(c *gin.Context) {
	live := c.Query("live") == "true"
	refresh := live && c.Query("refresh") == "true"
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)
	now := time.Now()

	resp, err := h.client.GetPositions(c.Request.Context(), auth, false, live, refresh)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		return
	}

	// Best-effort evolution for summary (Snapshot Change).
	evo, evoErr := h.client.GetEvolutionLast(c.Request.Context(), auth, 2)
	if evoErr != nil {
		if errors.Is(evoErr, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		evo = nil // tolerate
	}

	SortPositions(resp.Positions)
	metrics := ComputeMetrics(resp.Positions, evo)
	currencies := metrics.CurrencyOrder

	liveState := LiveState{Live: live, Refresh: refresh}
	tree := BuildLiveDataSection(resp, metrics, liveState, currencies, lang, now)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "live-data-section",
		Tree:     &tree,
	})
}
