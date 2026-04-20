package trades

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type Handler struct{ uc *GetUseCase }

func NewHandler(uc *GetUseCase) *Handler { return &Handler{uc: uc} }

// Get serves GET /screens/trades: validates query params, calls the use case,
// and returns the full screen tree. Trades or catalog ErrUnauthorized → 401
// with redirect; any other backend error → 502 BACKEND_ERROR.
func (h *Handler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	tree, err := h.uc.Execute(c.Request.Context(), auth, params, lang)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) || errors.Is(err, assetscatalog.ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load trades"}})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// parseListParams parses the shared trades list query params: asset_id (UUID),
// trade_type ("BUY"/"SELL"), offset (non-negative integer). Empty values mean
// "no filter".
func parseListParams(c *gin.Context) (ListParams, error) {
	p := ListParams{}
	if v := c.Query("asset_id"); v != "" {
		if _, err := uuid.Parse(v); err != nil {
			return p, errors.New("invalid asset_id")
		}
		p.AssetID = v
	}
	if v := c.Query("trade_type"); v != "" {
		if v != "BUY" && v != "SELL" {
			return p, errors.New("invalid trade_type")
		}
		p.TradeType = v
	}
	if raw := c.Query("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return p, errors.New("invalid offset")
		}
		p.Offset = n
	}
	return p, nil
}

// parseLang extracts the base language tag from Accept-Language, falling back
// to "en" when the header is missing or blank.
func parseLang(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return "en"
	}
	parts := strings.SplitN(header, ",", 2)
	lang := strings.SplitN(parts[0], "-", 2)[0]
	lang = strings.SplitN(lang, ";", 2)[0]
	return strings.TrimSpace(lang)
}
