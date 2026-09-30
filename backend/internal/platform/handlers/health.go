package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sentinelexam/backend/internal/messaging/jetstream"
	"github.com/sentinelexam/backend/internal/persistence/postgres"
)

// HealthHandler manages live, ready, and startup probe endpoints.
type HealthHandler struct {
	db   *postgres.DB
	nats *jetstream.NATSClient
}

// NewHealthHandler creates a health check handler.
func NewHealthHandler(db *postgres.DB, nats *jetstream.NATSClient) *HealthHandler {
	return &HealthHandler{
		db:   db,
		nats: nats,
	}
}

// Live handles GET /health/live.
// Semantics: Must not fail merely because dependencies like Postgres or NATS are temporarily reconnecting.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "UP",
		"service":   "sentinel-backend",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready handles GET /health/ready.
// Semantics: Reflects whether the service can safely provide its core responsibility.
// AI availability must NOT make the assessment API unready.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	components := make(map[string]string)
	isReady := true

	// Check PostgreSQL (Authoritative)
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			components["database"] = "DOWN"
			isReady = false
		} else {
			components["database"] = "UP"
		}
	} else {
		components["database"] = "NOT_CONFIGURED"
		isReady = false
	}

	// Check NATS JetStream (Durable backbone)
	if h.nats != nil && h.nats.IsConnected() {
		components["nats"] = "UP"
	} else {
		components["nats"] = "DOWN"
		isReady = false
	}

	// Temporal & AI components are advisory/workflow-only for live ingestion readiness
	components["ai_service"] = "ADVISORY_READY"

	w.Header().Set("Content-Type", "application/json")
	if isReady {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "READY",
			"components": components,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     "NOT_READY",
			"components": components,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// Startup handles GET /health/startup.
func (h *HealthHandler) Startup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "STARTED",
		"initialized": true,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}
