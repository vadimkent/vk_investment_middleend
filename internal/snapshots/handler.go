package snapshots

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/shared"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

type Handler struct{ uc *GetUseCase }

func NewHandler(uc *GetUseCase) *Handler { return &Handler{uc: uc} }

// Get serves GET /screens/snapshots: validates query params, calls the use
// case, and returns the full screen tree. Snapshot or catalog ErrUnauthorized
// → 401 with redirect; any other backend error → 502 BACKEND_ERROR.
func (h *Handler) Get(c *gin.Context) {
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
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
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load snapshots"}})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// parseListParams parses the snapshots list query params: is_full_snapshot
// ("true"/"false"/absent) and offset (non-negative integer). Empty values mean
// "no filter".
func parseListParams(c *gin.Context) (ListParams, error) {
	p := ListParams{}
	if v := c.Query("is_full_snapshot"); v != "" {
		switch v {
		case "true":
			t := true
			p.IsFullSnapshot = &t
		case "false":
			f := false
			p.IsFullSnapshot = &f
		default:
			return p, errors.New("invalid is_full_snapshot")
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

// respondBadRequest writes a 400 BAD_REQUEST response with the given message.
func respondBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": message}})
}
