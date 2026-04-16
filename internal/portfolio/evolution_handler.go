package portfolio

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/shared"
)

// evolutionFetcher is the narrow interface the handler needs; *Client satisfies it.
type evolutionFetcher interface {
	GetEvolution(ctx context.Context, authorization string, q EvolutionQuery) ([]EvolutionPoint, error)
}

type EvolutionHandler struct {
	client evolutionFetcher
	now    func() time.Time
}

func NewEvolutionHandler(client evolutionFetcher) *EvolutionHandler {
	return &EvolutionHandler{client: client, now: time.Now}
}

func (h *EvolutionHandler) Get(c *gin.Context) {
	timeframe := c.DefaultQuery("timeframe", "all")
	mode := c.DefaultQuery("mode", "abs")
	currency := c.Query("currency")

	if !isValidTimeframe(timeframe) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid timeframe"}})
		return
	}
	if mode != "abs" && mode != "pct" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid mode"}})
		return
	}

	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	// Don't pass currency to the BE — we need all points to compute the full
	// currency list for the control. The chart builder filters by state.Currency.
	q := EvolutionQuery{Points: 100}
	if from := timeframeFrom(timeframe, h.now()); from != nil {
		q.From = from
	}

	points, err := h.client.GetEvolution(c.Request.Context(), auth, q)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/screens/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load evolution"}})
		return
	}

	state := ChartState{Timeframe: timeframe, Mode: mode, Currency: currency}
	currencies := distinctCurrencies(points)
	tree := BuildValueOverTimeCard(points, state, currencies, lang)

	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: "chart-value-over-time-card",
		Tree:     &tree,
	})
}

func isValidTimeframe(tf string) bool {
	for _, v := range timeframes {
		if tf == v {
			return true
		}
	}
	return false
}

func timeframeFrom(tf string, now time.Time) *time.Time {
	switch tf {
	case "1m":
		t := now.AddDate(0, 0, -30)
		return &t
	case "3m":
		t := now.AddDate(0, 0, -90)
		return &t
	case "6m":
		t := now.AddDate(0, 0, -180)
		return &t
	case "ytd":
		t := time.Date(now.UTC().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return &t
	case "1y":
		t := now.AddDate(0, 0, -365)
		return &t
	default:
		return nil
	}
}

// distinctCurrencies returns currencies present in points, ordered by each
// currency's most-recent total_value descending. Matches the ordering used by
// the initial screen render (positions' current_value descending) as closely
// as possible when evolution data is available.
func distinctCurrencies(points []EvolutionPoint) []string {
	latest := map[string]EvolutionPoint{}
	for _, p := range points {
		existing, ok := latest[p.Currency]
		if !ok || p.RecordedAt.After(existing.RecordedAt) {
			latest[p.Currency] = p
		}
	}
	out := make([]string, 0, len(latest))
	for c := range latest {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		vi := latest[out[i]].TotalValue
		vj := latest[out[j]].TotalValue
		if vi == vj {
			return out[i] < out[j]
		}
		return vi > vj
	})
	return out
}

