package portfolio

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// allocationFetcher is the narrow interface the handler needs; *Client satisfies it.
type allocationFetcher interface {
	GetPositions(ctx context.Context, authorization string, includeClosed, live, refresh bool) (*PortfolioResponse, error)
}

type AllocationHandler struct {
	client allocationFetcher
}

func NewAllocationHandler(client allocationFetcher) *AllocationHandler {
	return &AllocationHandler{client: client}
}

// Get handles GET /actions/portfolio/allocation.
func (h *AllocationHandler) Get(c *gin.Context) {
	groupBy := c.DefaultQuery("group_by", "asset")
	currency := c.Query("currency")

	if groupBy != "asset" && groupBy != "type" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid group_by"}})
		return
	}

	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	portfolioResp, err := h.client.GetPositions(c.Request.Context(), auth, false, false, false)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load allocation"}})
		return
	}
	positions := portfolioResp.Positions

	state := AllocationState{GroupBy: groupBy, Currency: currency}
	currencies := distinctPositionCurrencies(positions)
	tree := BuildAllocationSection(positions, state, currencies, lang)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "allocation-section",
		Tree:     &tree,
	})
}

// distinctPositionCurrencies returns currencies present in positions (with
// non-null current_value), ordered by each currency's total value DESC. Matches
// the ordering used on the initial screen render so reloads preserve it.
func distinctPositionCurrencies(positions []Position) []string {
	totals := map[string]float64{}
	for _, p := range positions {
		if p.CurrentValue == nil {
			continue
		}
		totals[p.Currency] += *p.CurrentValue
	}
	out := make([]string, 0, len(totals))
	for c := range totals {
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		vi, vj := totals[out[i]], totals[out[j]]
		if vi == vj {
			return out[i] < out[j]
		}
		return vi > vj
	})
	return out
}
