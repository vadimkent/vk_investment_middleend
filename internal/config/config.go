package config

import (
	"errors"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port               int           `envconfig:"PORT" default:"8080"`
	LogLevel           string        `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat          string        `envconfig:"LOG_FORMAT" default:"json"`
	BackendURL         string        `envconfig:"BACKEND_URL" default:"http://localhost:8080"`
	CORSAllowedOrigins string        `envconfig:"CORS_ALLOWED_ORIGINS" default:"http://localhost:3000"`
	RequestTimeout     time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	JWTSecret          string        `envconfig:"JWT_SECRET"`
	JWTLeewaySeconds   int           `envconfig:"JWT_LEEWAY_SECONDS" default:"30"`
	AnalysisFake       bool          `envconfig:"ANALYSIS_FAKE" default:"false"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	return &cfg, nil
}
