package config

import (
	"log"
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config хранит настраиваемые параметры приложения.
type Config struct {
	HTTPPort       string        `env:"HTTP_PORT" env-default:"8080"`
	BinaryPath     string        `env:"BINARY_PATH" env-default:"bin/qms_lib"`
	ServerDataPath string        `env:"SERVER_DATA_PATH" env-default:"server_data"`
	TestResultPath string        `env:"TEST_RESULT_PATH" env-default:"data/test.json"`
	ExecTimeoutSec time.Duration `env:"EXEC_TIMEOUT_SEC" env-default:"120s"`
	ServerID       int           `env:"SERVER_ID" env-default:"0"`
}

func Load(logger *slog.Logger) (*Config, error) {
	// Загружаем .env, если есть
	_ = godotenv.Load()

	cfg := &Config{}

	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		log.Fatalf("Error reading env %v", err)
	}

	if logger != nil {
		logger.Info("config loaded",
			slog.String("http_addr", cfg.HTTPPort),
			slog.String("binary_path", cfg.BinaryPath),
			slog.String("server_data", cfg.ServerDataPath),
			slog.String("test_result", cfg.TestResultPath),
			slog.Float64("exec_timeout_sec", cfg.ExecTimeoutSec.Seconds()),
		)
	}

	return cfg, nil
}
