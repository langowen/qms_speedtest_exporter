package http_server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/langowen/qms_speedtest_exporter/internal/entities"
	"github.com/langowen/qms_speedtest_exporter/internal/port/http-server/middleware/logger"

	"github.com/langowen/qms_speedtest_exporter/internal/config"
	"github.com/langowen/qms_speedtest_exporter/internal/service"
)

// Server инкапсулирует HTTP-сервер и маршруты
type Server struct {
	log        *slog.Logger
	cfg        *config.Config
	svc        Service
	httpServer *http.Server
}

type Service interface {
	GetServers(ctx context.Context) ([]entities.Server, error)
	RunSpeedtest(ctx context.Context) (string, error)
}

func NewServer(log *slog.Logger, cfg *config.Config, svc *service.Service) *Server {
	return &Server{
		log: log,
		cfg: cfg,
		svc: svc,
		httpServer: &http.Server{
			Addr: ":" + cfg.HTTPPort,
		},
	}
}

// Start запускает сервер (не блокирует). Возвращает функцию для ожидания завершения.
func (s *Server) Start() error {
	s.httpServer.Handler = s.registerRouter()

	s.log.Info("starting http server", slog.String("addr", s.cfg.HTTPPort))
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http server stopped", slog.Any("err", err))
		}
	}()
	return nil
}

// Shutdown останавливает сервер с graceful shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("shutting down http server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) registerRouter() http.Handler {
	r := chi.NewRouter()
	// Middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.NewLogger(s.log))
	// Таймаут обработки запроса
	r.Use(middleware.Timeout(s.cfg.ExecTimeoutSec))

	// Хэндлеры
	r.Get("/health", s.healthCheck)
	r.Get("/server_list", s.serverList)
	r.Get("/speedtest", s.speedtest)

	return r
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) serverList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	servers, err := s.svc.GetServers(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, entities.ErrorResponse{Code: "get_servers_failed", Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (s *Server) speedtest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	res, err := s.svc.RunSpeedtest(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, entities.ErrorResponse{Code: "speedtest_failed", Message: err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(res))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
