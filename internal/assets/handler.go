package assets

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
)

type Handler struct {
	uc *GetUseCase
}

func NewHandler(uc *GetUseCase) *Handler {
	return &Handler{uc: uc}
}

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
		if errors.Is(err, ErrUnauthorized) {
			shared.RespondUnauthorized(c, "/login")
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load assets"}})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// parseListParams extracts and validates asset_type and offset from the query.
func parseListParams(c *gin.Context) (ListParams, error) {
	p := ListParams{}
	at := c.Query("asset_type")
	if at != "" {
		switch at {
		case "STOCK", "ETF", "CRYPTO", "BOND":
			p.AssetType = at
		default:
			return p, errors.New("invalid asset_type")
		}
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
