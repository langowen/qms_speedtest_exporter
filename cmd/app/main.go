package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/langowen/qms_speedtest_exporter/internal/adapter/qmsclient"
	"github.com/langowen/qms_speedtest_exporter/internal/config"
	"github.com/langowen/qms_speedtest_exporter/internal/port/http-server"
	"github.com/langowen/qms_speedtest_exporter/internal/service"
)

func main() {
	// Логгер slog
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Конфигурация
	cfg, _ := config.Load(logger)

	// Зависимости: адаптер и сервис
	adapter := qmsclient.NewQMCClient(logger, cfg)
	svc := service.NewService(adapter, cfg)

	// HTTP-сервер (внутренний порт)
	srv := http_server.NewServer(logger, cfg, svc)
	_ = srv.Start()

	// Graceful shutdown по сигналам
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
