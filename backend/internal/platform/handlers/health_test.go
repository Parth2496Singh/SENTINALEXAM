package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sentinelexam/backend/internal/platform/handlers"
)

func TestHealthHandler_Live(t *testing.T) {
	h := handlers.NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "UP" {
		t.Errorf("expected status 'UP', got %v", body["status"])
	}
}

func TestHealthHandler_Startup(t *testing.T) {
	h := handlers.NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health/startup", nil)
	rec := httptest.NewRecorder()

	h.Startup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestHealthHandler_Ready_GracefulDegradation(t *testing.T) {
	// When DB and NATS are nil/unconnected, Ready should return 503 but still respond cleanly
	h := handlers.NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for unconfigured db/nats, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "NOT_READY" {
		t.Errorf("expected status 'NOT_READY', got %v", body["status"])
	}

	components, ok := body["components"].(map[string]any)
	if !ok {
		t.Fatalf("expected components map")
	}

	// AI service should be reported as advisory ready
	if components["ai_service"] != "ADVISORY_READY" {
		t.Errorf("expected ai_service to be ADVISORY_READY, got %v", components["ai_service"])
	}
}
