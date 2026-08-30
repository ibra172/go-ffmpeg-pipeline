package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config хранит все настройки приложения, читается из переменных окружения.
type Config struct {
	Port            string        `envconfig:"HTTP_PORT" default:":8000"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
	SwaggerDir      string        `envconfig:"SWAGGER_DIR" default:"./docs"`

	AMQPURL       string `envconfig:"AMQP_URL" default:"amqp://guest:guest@127.0.0.1:5672/"`
	TaskQueueName string `envconfig:"TASK_QUEUE_NAME" default:"tasks"`

	DataDir         string `envconfig:"DATA_DIR" default:"/data"`
	MaxUploadSizeMB int64  `envconfig:"MAX_UPLOAD_SIZE_MB" default:"500"`
	WatermarkPath   string `envconfig:"WATERMARK_PATH" default:"resources/watermark.png"`

	GRPCPort   string `envconfig:"GRPC_PORT" default:":9090"`
	GRPCTarget string `envconfig:"GRPC_TARGET" default:"localhost:9090"`
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
