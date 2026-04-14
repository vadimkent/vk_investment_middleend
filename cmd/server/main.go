package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/project/vk-investment-middleend/internal/config"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/observability"
	"github.com/project/vk-investment-middleend/internal/server"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if err := i18n.Load("locales"); err != nil {
		log.Fatal().Err(err).Msg("failed to load translations")
	}

	tp, err := observability.InitTracer("vk-investment-middleend")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init tracer")
	}
	defer observability.Shutdown(tp)

	srv := server.New(cfg)
	if err := srv.Run(); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
