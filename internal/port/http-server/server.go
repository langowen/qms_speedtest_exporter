package http_server

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"time"

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
func (s *Server) Start() {
	s.httpServer.Handler = s.registerRouter()

	s.log.Info("starting http server", slog.String("addr", s.cfg.HTTPPort))
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http server stopped", slog.Any("err", err))
		}
	}()
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
	r.Get("/", s.homeHandler)
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

type PageData struct {
	Title   string
	Message string
	Time    string
	Links   []Link
}

type Link struct {
	URL         string
	Title       string
	Description string
}

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Speedtest Service",
		Message: "Сервис для тестирования скорости интернета, время выполнения теста скорости порядка 40 секунд",
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Links: []Link{
			{URL: "/speedtest", Title: "/speedtest", Description: "Запуск теста скорости"},
			{URL: "/server_list", Title: "/server_list", Description: "Список серверов"},
			{URL: "/health", Title: "/health", Description: "Статус сервиса"},
		},
	}

	tmpl := template.Must(template.New("home").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Message}}</p>
    
    <div>
        {{range .Links}}
        <div class="link-item">
            <a href="{{.URL}}">{{.Title}}</a>
            <span class="link-desc">{{.Description}}</span>
        </div>
        {{end}}
    </div>
    
    <p>Время: {{.Time}}</p>
</body>
</html>`))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}
