package telemetry

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	// HTTPRequestsTotal counts total incoming HTTP requests.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total count of HTTP requests processed by Sentinel API.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration measures HTTP request latency distribution.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests processed by Sentinel API.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
}

// Telemetry encapsulates OpenTelemetry tracer and Prometheus handler.
type Telemetry struct {
	Tracer trace.Tracer
	logger *slog.Logger
}

// InitTelemetry initializes OpenTelemetry configuration and metrics.
func InitTelemetry(ctx context.Context, serviceName, otlpEndpoint string, logger *slog.Logger) (*Telemetry, func(context.Context) error, error) {
	logger.Info("initializing telemetry", "service", serviceName, "otlp_endpoint", otlpEndpoint)

	// In initial foundation, we initialize a standard tracer provider
	tp := noop.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer(serviceName)

	cleanup := func(c context.Context) error {
		logger.Info("shutting down telemetry")
		return nil
	}

	return &Telemetry{
		Tracer: tracer,
		logger: logger,
	}, cleanup, nil
}

// MetricsHandler returns the standard Prometheus HTTP handler.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
