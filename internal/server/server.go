package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/project/vk-investment-middleend/internal/assets"
	"github.com/project/vk-investment-middleend/internal/auth"
	"github.com/project/vk-investment-middleend/internal/config"
	"github.com/project/vk-investment-middleend/internal/home"
	"github.com/project/vk-investment-middleend/internal/login"
	"github.com/project/vk-investment-middleend/internal/portfolio"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
	"github.com/project/vk-investment-middleend/internal/shell"
	"github.com/project/vk-investment-middleend/internal/trades"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
}

func New(cfg *config.Config) *Server {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("vk-investment-middleend"))

	s := &Server{cfg: cfg, router: router}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Public routes (no auth).
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/screens/login", login.NewHandler().Get)

	authClient := auth.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	s.router.POST("/actions/login", auth.NewLoginHandler(authClient).Post)
	s.router.POST("/actions/register", auth.NewRegisterHandler(authClient).Post)

	// Protected routes.
	leeway := time.Duration(s.cfg.JWTLeewaySeconds) * time.Second
	protected := s.router.Group("")
	protected.Use(auth.RequireAuth(s.cfg.JWTSecret, leeway, "/login"))

	shellUC := shell.NewGetUseCase()
	shellHandler := shell.NewHandler(shellUC)
	protected.GET("/shell", shellHandler.Get)

	homeClient := home.NewClient(s.cfg.BackendURL)
	homeUC := home.NewGetUseCase(homeClient)
	homeHandler := home.NewHandler(homeUC)
	protected.GET("/screens/home", homeHandler.Get)

	portfolioClient := portfolio.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	portfolioHandler := portfolio.NewHandler(portfolio.NewGetUseCase(portfolioClient))
	protected.GET("/screens/portfolio", portfolioHandler.Get)
	protected.POST("/actions/portfolio/include_closed", portfolio.NewIncludeClosedHandler(portfolioClient).Post)
	protected.GET("/actions/portfolio/evolution", portfolio.NewEvolutionHandler(portfolioClient).Get)
	protected.GET("/actions/portfolio/allocation", portfolio.NewAllocationHandler(portfolioClient).Get)
	protected.GET("/actions/portfolio/live_data", portfolio.NewLiveHandler(portfolioClient).Get)

	assetsClient := assets.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	assetsUC := assets.NewGetUseCase(assetsClient)
	protected.GET("/screens/assets", assets.NewHandler(assetsUC).Get)
	protected.GET("/actions/assets/list", assets.NewListHandler(assetsUC).Get)
	protected.GET("/actions/assets/create_modal", assets.NewCreateModalHandler().Get)
	protected.GET("/actions/assets/edit_modal", assets.NewEditModalHandler(assetsClient).Get)
	protected.GET("/actions/assets/delete_modal", assets.NewDeleteModalHandler(assetsClient).Get)
	protected.POST("/actions/assets/create", assets.NewCreateHandler(assetsClient).Post)
	protected.PATCH("/actions/assets/:id", assets.NewUpdateHandler(assetsClient).Patch)
	protected.DELETE("/actions/assets/:id", assets.NewDeleteHandler(assetsClient).Delete)

	// --- trades ---
	tradesClient := trades.NewClient(s.cfg.BackendURL, s.cfg.RequestTimeout)
	catalog := assetscatalog.NewCatalog(s.cfg.BackendURL, s.cfg.RequestTimeout)
	tradesUC := trades.NewGetUseCase(tradesClient, catalog)
	protected.GET("/screens/trades", trades.NewHandler(tradesUC).Get)
	protected.GET("/actions/trades/list", trades.NewListHandler(tradesUC).Get)
	protected.GET("/actions/trades/create_modal", trades.NewCreateModalHandler(catalog).Get)
	protected.GET("/actions/trades/edit_modal", trades.NewEditModalHandler(tradesClient, catalog).Get)
	protected.GET("/actions/trades/delete_modal", trades.NewDeleteModalHandler(tradesClient, catalog).Get)
	protected.POST("/actions/trades/create", trades.NewCreateHandler(tradesClient, tradesUC, catalog).Post)
	protected.PATCH("/actions/trades/:id", trades.NewUpdateHandler(tradesClient, tradesUC, catalog).Patch)
	protected.DELETE("/actions/trades/:id", trades.NewDeleteHandler(tradesClient, tradesUC).Delete)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "vk-investment-middleend",
	})
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Info().Str("addr", addr).Msg("starting server")
	return s.router.Run(addr)
}
