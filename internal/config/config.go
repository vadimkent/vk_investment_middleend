package config

import (
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
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
