package portfolio

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
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

	q := EvolutionQuery{Points: 100, Currency: currency}
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

	var tree components.Component
	if mode == "pct" {
		// EvolutionPoint has no total_cost yet. Surface the no-cost empty state.
		tree = buildPctNoCostCard(state, currencies, lang)
	} else {
		tree = BuildValueOverTimeCard(points, state, currencies, lang)
	}

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

func distinctCurrencies(points []EvolutionPoint) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, p := range points {
		if _, ok := seen[p.Currency]; ok {
			continue
		}
		seen[p.Currency] = struct{}{}
		out = append(out, p.Currency)
	}
	return out
}

// buildPctNoCostCard reproduces the chart card shape with the no-cost empty
// message, preserving controls and their selected state.
func buildPctNoCostCard(state ChartState, currencies []string, lang string) components.Component {
	card := BuildValueOverTimeCard(nil, state, currencies, lang)
	chart := findDescendantByIDRef(&card, "chart-value-over-time")
	if chart != nil {
		chart.Props["empty_message"] = i18n.T(lang, "portfolio.chart.no_cost_data")
	}
	return card
}

// findDescendantByIDRef walks the component tree returning a pointer so callers
// can mutate Props.
func findDescendantByIDRef(c *components.Component, id string) *components.Component {
	if c.ID == id {
		return c
	}
	for i := range c.Children {
		if found := findDescendantByIDRef(&c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
