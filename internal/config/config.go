package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config хранит все настройки приложения, читается из переменных окружения.
type Config struct {
	Port            string        `envconfig:"PORT" default:":8000"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
}

func New() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}
	return cfg, nil
}

func MustNew() Config {
	cfg, err := New()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}
	return cfg
}
