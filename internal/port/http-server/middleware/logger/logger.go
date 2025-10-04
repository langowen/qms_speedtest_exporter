package logger

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func NewLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logger.Info("logger middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			log := logger.With(
				slog.String("component", "middleware/logger"),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("protocol", r.URL.Scheme),
				slog.String("host", r.Host),
				slog.String("URL", decodeURI(r.RequestURI)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				status := ww.Status()
				bytes := ww.BytesWritten()
				duration := time.Since(start)

				responseFields := []any{
					"status", status,
					"bytes", bytes,
					"duration", duration.String(),
					"duration_ms", duration.Seconds() * 1000,
				}

				var level slog.Level
				var msg string

				switch {
				case status >= 500:
					level = slog.LevelError
					msg = "server error"
				case status >= 400:
					level = slog.LevelWarn
					msg = "client error"
				case status == 304:
					level = slog.LevelInfo
					msg = "not modified (cached)"
				case status >= 300:
					level = slog.LevelInfo
					msg = "redirection"
				default:
					level = slog.LevelInfo
					msg = "request completed"
				}

				log.Log(r.Context(), level, msg, responseFields...)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}

func decodeURI(uri string) string {
	decoded, err := url.PathUnescape(uri)
	if err != nil {
		return uri
	}
	return decoded
}
