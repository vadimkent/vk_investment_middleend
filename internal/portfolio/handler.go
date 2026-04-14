package portfolio

import (
	"errors"
	"net/http"
	"strings"
	"time"

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
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	screen, err := h.uc.Execute(c.Request.Context(), auth, lang, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthorized):
			shared.RespondUnauthorized(c, "/screens/login")
		default:
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "BACKEND_ERROR", "message": "could not load portfolio"}})
		}
		return
	}
	c.JSON(http.StatusOK, screen)
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
