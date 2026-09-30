package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sentinelexam/backend/internal/observability/logging"
	"github.com/sentinelexam/backend/internal/observability/telemetry"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ObservabilityMiddleware injects correlation IDs, logs requests, and records Prometheus metrics.
func ObservabilityMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract or generate Correlation ID
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				bytes := make([]byte, 8)
				_, _ = rand.Read(bytes)
				correlationID = "req-" + hex.EncodeToString(bytes)
			}

			w.Header().Set("X-Correlation-ID", correlationID)
			ctx := logging.WithCorrelationID(r.Context(), correlationID)

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r.WithContext(ctx))

			duration := time.Since(start)

			// Record Prometheus Metrics
			statusStr := strconv.Itoa(rw.statusCode)
			telemetry.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
			telemetry.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())

			// Log structured access line
			logger.InfoContext(ctx, "http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
				"correlation_id", correlationID,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// CORSMiddleware applies permissive CORS headers for local frontend integration.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
