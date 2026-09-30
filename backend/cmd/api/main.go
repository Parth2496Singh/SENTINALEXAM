package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentinelexam/backend/internal/assessment/repository"
	repopg "github.com/sentinelexam/backend/internal/assessment/repository/postgres"
	"github.com/sentinelexam/backend/internal/assessment/service"
	"github.com/sentinelexam/backend/internal/auth/keycloak"
	authmw "github.com/sentinelexam/backend/internal/auth/middleware"
	"github.com/sentinelexam/backend/internal/messaging/jetstream"
	"github.com/sentinelexam/backend/internal/observability/logging"
	"github.com/sentinelexam/backend/internal/observability/telemetry"
	"github.com/sentinelexam/backend/internal/persistence/postgres"
	"github.com/sentinelexam/backend/internal/platform/config"
	"github.com/sentinelexam/backend/internal/platform/handlers"
	"github.com/sentinelexam/backend/internal/platform/middleware"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadFromEnv()

	// 2. Structured Logging
	logger := logging.NewLogger(cfg.LogLevel)
	logger.Info("starting sentinel-backend api service",
		"environment", cfg.Environment,
		"port", cfg.HTTPPort,
	)

	// 3. Telemetry & Metrics Initialization
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, cleanupTelemetry, err := telemetry.InitTelemetry(ctx, cfg.ServiceName, cfg.OTLPEndpoint, logger)
	if err != nil {
		logger.Error("failed to initialize telemetry", "error", err)
	}
	defer func() {
		_ = cleanupTelemetry(context.Background())
	}()

	// 4. PostgreSQL Initialization
	db, err := postgres.NewClient(ctx, cfg, logger)
	if err != nil {
		logger.Warn("postgresql initial connection failed (will retry in background)", "error", err)
	} else {
		defer db.Close()
		if err := db.Ping(ctx); err != nil {
			logger.Warn("postgresql ping failed", "error", err)
		} else {
			logger.Info("postgresql connection verified successfully")
		}
	}

	// 5. NATS JetStream Initialization
	natsClient, err := jetstream.NewClient(cfg, logger)
	if err != nil {
		logger.Warn("nats jetstream connection failed (will retry in background)", "error", err)
	} else {
		defer natsClient.Close()
		logger.Info("nats jetstream connected successfully")
	}

	// 6. Assessment Repository Layer
	var repos repository.Repositories
	if db != nil {
		repoDB := repopg.New(db.Pool)
		repos = repoDB.Repositories()
	}

	// 7. Auth
	authenticator := keycloak.NewAuthenticator(cfg, logger)
	authMW := authmw.AuthMiddleware(authenticator)
	adminOnly := authmw.RequireAnyRole("exam_admin", "system_admin")
	candidateOnly := authmw.RequireAnyRole("candidate")
	candidateOrAdmin := authmw.RequireAnyRole("candidate", "exam_admin", "system_admin")

	// 8. Services
	qbSvc := service.NewQuestionBankService(repos, logger)
	qSvc := service.NewQuestionService(repos, logger)
	examSvc := service.NewExamService(repos, logger)
	candidateSvc := service.NewCandidateService(repos, logger)
	assignSvc := service.NewAssignmentService(repos, logger)
	attemptSvc := service.NewAttemptService(repos, logger)
	sessionSvc := service.NewSessionService(repos, logger)
	responseSvc := service.NewResponseService(repos, logger)
	submissionSvc := service.NewSubmissionService(repos, logger)
	var pool *pgxpool.Pool
	if db != nil {
		pool = db.Pool
	}
	ingestSvc := service.NewEventIngestionService(pool, repos, logger)

	// 9. Handlers
	qbHandler := handlers.NewQuestionBankHandler(qbSvc, qSvc, logger)
	qHandler := handlers.NewQuestionHandler(qSvc, logger)
	examHandler := handlers.NewExamHandler(examSvc, logger)
	candidateHandler := handlers.NewCandidateHandler(candidateSvc, logger)
	assignHandler := handlers.NewAssignmentHandler(assignSvc, candidateSvc, logger)
	attemptHandler := handlers.NewAttemptHandler(attemptSvc, candidateSvc, responseSvc, submissionSvc, ingestSvc, logger)
	sessionHandler := handlers.NewSessionHandler(sessionSvc, candidateSvc, logger)

	// 10. Router
	mux := http.NewServeMux()

	// --- Health & Platform ---
	healthHandler := handlers.NewHealthHandler(db, natsClient)
	mux.HandleFunc("GET /health/live", healthHandler.Live)
	mux.HandleFunc("GET /health/ready", healthHandler.Ready)
	mux.HandleFunc("GET /health/startup", healthHandler.Startup)
	mux.Handle("GET /metrics", telemetry.MetricsHandler())
	mux.HandleFunc("GET /api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"platform":    "SentinelExam",
			"description": "Resilient & Trustworthy Online Assessment Platform",
			"version":     "1.0.0",
			"status":      "assessment_core_ready",
		})
	})

	// --- Question Banks (exam_admin+) ---
	mux.Handle("GET /api/v1/question-banks", chain(authMW, adminOnly)(http.HandlerFunc(qbHandler.List)))
	mux.Handle("POST /api/v1/question-banks", chain(authMW, adminOnly)(http.HandlerFunc(qbHandler.Create)))
	mux.Handle("GET /api/v1/question-banks/{bankId}", chain(authMW, adminOnly)(http.HandlerFunc(qbHandler.Get)))
	mux.Handle("POST /api/v1/question-banks/{bankId}/questions", chain(authMW, adminOnly)(http.HandlerFunc(qbHandler.CreateQuestion)))

	// --- Questions (exam_admin+) ---
	mux.Handle("GET /api/v1/questions/{questionId}", chain(authMW, adminOnly)(http.HandlerFunc(qHandler.Get)))
	mux.Handle("POST /api/v1/questions/{questionId}/publish", chain(authMW, adminOnly)(http.HandlerFunc(qHandler.Publish)))
	mux.Handle("POST /api/v1/questions/{questionId}/retire", chain(authMW, adminOnly)(http.HandlerFunc(qHandler.Retire)))

	// --- Exams ---
	mux.Handle("GET /api/v1/exams", chain(authMW, candidateOrAdmin)(http.HandlerFunc(examHandler.List)))
	mux.Handle("POST /api/v1/exams", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Create)))
	mux.Handle("GET /api/v1/exams/{examId}", chain(authMW, candidateOrAdmin)(http.HandlerFunc(examHandler.Get)))
	mux.Handle("PUT /api/v1/exams/{examId}", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Update)))
	mux.Handle("GET /api/v1/exams/{examId}/questions", chain(authMW, candidateOrAdmin)(http.HandlerFunc(examHandler.GetQuestions)))
	mux.Handle("POST /api/v1/exams/{examId}/questions", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.AddQuestion)))
	mux.Handle("POST /api/v1/exams/{examId}/publish", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Publish)))
	mux.Handle("POST /api/v1/exams/{examId}/start", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Start)))
	mux.Handle("POST /api/v1/exams/{examId}/pause", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Pause)))
	mux.Handle("POST /api/v1/exams/{examId}/resume", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Resume)))
	mux.Handle("POST /api/v1/exams/{examId}/complete", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Complete)))
	mux.Handle("POST /api/v1/exams/{examId}/archive", chain(authMW, adminOnly)(http.HandlerFunc(examHandler.Archive)))

	// --- Candidates ---
	mux.Handle("GET /api/v1/candidates", chain(authMW, adminOnly)(http.HandlerFunc(candidateHandler.List)))
	mux.Handle("POST /api/v1/candidates", chain(authMW, candidateOrAdmin)(http.HandlerFunc(candidateHandler.Create)))
	mux.Handle("GET /api/v1/candidates/{candidateId}", chain(authMW, adminOnly)(http.HandlerFunc(candidateHandler.Get)))

	// --- Assignments ---
	mux.Handle("GET /api/v1/candidate/assignments", chain(authMW, candidateOnly)(http.HandlerFunc(assignHandler.ListMyAssignments)))
	mux.Handle("GET /api/v1/exams/{examId}/assignments", chain(authMW, adminOnly)(http.HandlerFunc(assignHandler.List)))
	mux.Handle("POST /api/v1/exams/{examId}/assignments", chain(authMW, adminOnly)(http.HandlerFunc(assignHandler.Assign)))
	mux.Handle("DELETE /api/v1/exams/{examId}/assignments/{candidateId}", chain(authMW, adminOnly)(http.HandlerFunc(assignHandler.Revoke)))

	// --- Candidate Attempt Runtime ---
	mux.Handle("POST /api/v1/exams/{examId}/attempts", chain(authMW, candidateOrAdmin)(http.HandlerFunc(attemptHandler.Start)))
	mux.Handle("GET /api/v1/attempts/{attemptId}/runtime", chain(authMW, candidateOrAdmin)(http.HandlerFunc(attemptHandler.Runtime)))
	mux.Handle("GET /api/v1/attempts/{attemptId}/responses", chain(authMW, candidateOrAdmin)(http.HandlerFunc(attemptHandler.ListResponses)))
	mux.Handle("PUT /api/v1/attempts/{attemptId}/responses/{questionVersionId}", chain(authMW, candidateOrAdmin)(http.HandlerFunc(attemptHandler.SaveResponse)))
	mux.Handle("POST /api/v1/attempts/{attemptId}/events:sync", chain(authMW, candidateOrAdmin)(http.HandlerFunc(attemptHandler.SyncEvents)))
	mux.Handle("POST /api/v1/attempts/{attemptId}/submit", chain(authMW, candidateOrAdmin)(http.HandlerFunc(attemptHandler.Submit)))

	// --- Sessions ---
	mux.Handle("POST /api/v1/attempts/{attemptId}/sessions", chain(authMW, candidateOrAdmin)(http.HandlerFunc(sessionHandler.Create)))
	mux.Handle("POST /api/v1/sessions/{sessionId}/heartbeat", chain(authMW, candidateOrAdmin)(http.HandlerFunc(sessionHandler.Heartbeat)))
	mux.Handle("POST /api/v1/sessions/{sessionId}/terminate", chain(authMW, candidateOrAdmin)(http.HandlerFunc(sessionHandler.Terminate)))

	// --- Admin: Attempt management ---
	mux.Handle("GET /api/v1/admin/attempts", chain(authMW, adminOnly)(http.HandlerFunc(attemptHandler.AdminList)))
	mux.Handle("GET /api/v1/admin/attempts/{attemptId}", chain(authMW, adminOnly)(http.HandlerFunc(attemptHandler.AdminGet)))

	// Wrap global middleware
	handler := middleware.CORSMiddleware(middleware.ObservabilityMiddleware(logger)(mux))

	// 11. HTTP Server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	serverErrChan := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- err
		}
	}()

	// 12. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-quit:
		logger.Info("received termination signal", "signal", sig.String())
	case err := <-serverErrChan:
		logger.Error("server listener error", "error", err)
	}

	logger.Info("shutting down http server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	} else {
		logger.Info("server exited cleanly")
	}
}

// chain applies middleware in left-to-right order, outermost first.
func chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}


