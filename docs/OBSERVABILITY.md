# Observability Specification

## 1. Observability Pillars

SentinelExam implements standard OpenTelemetry distributed tracing and Prometheus metrics.

### 1.1 Metrics (`/metrics`)
Standard Prometheus metrics exposed by the Go runtime:
* `http_requests_total{method, route, status}`: Cumulative request count.
* `http_request_duration_seconds{method, route, status}`: Request latency histogram.
* `active_assessment_sessions`: Current connected session count.
* `candidate_events_ingested_total{status}`: Counter for ingested candidate events (committed, duplicate, rejected).
* `database_pool_connections{state}`: PostgreSQL connection pool metrics.

### 1.2 Structured Logging
All logs output in JSON format with structured keys:
* `timestamp`: ISO-8601 UTC timestamp.
* `level`: `DEBUG`, `INFO`, `WARN`, `ERROR`.
* `correlation_id`: Context correlation ID.
* `module`: Originating Go package/subsystem.
* `msg`: Human-readable summary message.
* `error`: Error details / stack trace when applicable.

### 1.3 Distributed Tracing
OpenTelemetry tracer context propagation across HTTP boundaries and NATS JetStream event metadata.
