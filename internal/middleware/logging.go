package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	applogger "github.com/kirban/social-media/internal/logger"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack forwards the connection takeover a WebSocket upgrade needs.
//
// statusRecorder embeds the http.ResponseWriter interface, which does not
// include Hijack, so the wrapper would otherwise hide that the underlying
// writer supports it and every upgrade would fail with "http.ResponseWriter
// does not implement http.Hijacker".
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("logging middleware: underlying ResponseWriter is not an http.Hijacker")
	}
	return hj.Hijack()
}

// Flush forwards streaming flushes, hidden by the same embedding.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Logging(logger *applogger.AppLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(rec, r)

			logger.Info().
				Str("request_id", chimiddleware.GetReqID(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rec.status).
				Dur("latency", time.Since(start)).
				Msg("request")
		})
	}
}
