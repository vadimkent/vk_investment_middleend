package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/project/vk-investment-middleend/internal/config"
	"github.com/project/vk-investment-middleend/internal/home"
	"github.com/project/vk-investment-middleend/internal/shell"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
}

func New(cfg *config.Config) *Server {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("vk-investment-middleend"))

	s := &Server{
		cfg:    cfg,
		router: router,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthHandler)

	// Shell — app frame with navigation
	shellUC := shell.NewGetUseCase()
	shellHandler := shell.NewHandler(shellUC)
	s.router.GET("/shell", shellHandler.Get)

	// Screen handlers: client → use case → handler
	homeClient := home.NewClient(s.cfg.BackendURL)
	homeUC := home.NewGetUseCase(homeClient)
	homeHandler := home.NewHandler(homeUC)
	s.router.GET("/screens/home", homeHandler.Get)
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
