// Package handlers implements all /api/v1 HTTP handlers for the assessment core.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
	"github.com/sentinelexam/backend/internal/assessment/service"
	authmw "github.com/sentinelexam/backend/internal/auth/middleware"
	"github.com/sentinelexam/backend/internal/platform/api"
)

// ---------------------------------------------------------------------------
// QuestionBankHandler
// ---------------------------------------------------------------------------

type QuestionBankHandler struct {
	svc    *service.QuestionBankService
	qSvc   *service.QuestionService
	logger *slog.Logger
}

func NewQuestionBankHandler(svc *service.QuestionBankService, qSvc *service.QuestionService, logger *slog.Logger) *QuestionBankHandler {
	return &QuestionBankHandler{svc: svc, qSvc: qSvc, logger: logger}
}

func (h *QuestionBankHandler) List(w http.ResponseWriter, r *http.Request) {
	p := api.ParsePagination(r)
	banks, total, err := h.svc.List(r.Context(), p)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, api.PaginatedResponse{
		Data: banks,
		Meta: api.PaginationMeta{Page: p.Page, PerPage: p.PerPage, Total: total},
	})
}

func (h *QuestionBankHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	bank, err := h.svc.Create(r.Context(), service.CreateQuestionBankInput{
		Name:        req.Name,
		Description: req.Description,
		Claims:      *claims,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": bank})
}

func (h *QuestionBankHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "bankId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	bank, err := h.svc.Get(r.Context(), id)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": bank})
}

func (h *QuestionBankHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	bankID, err := api.ParseIDParam(r, "bankId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req struct {
		Type          string         `json:"type"`
		Content       map[string]any `json:"content"`
		CorrectAnswer map[string]any `json:"correct_answer"`
		Marks         float64        `json:"marks"`
		NegativeMarks float64        `json:"negative_marks"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	q, qv, err := h.qSvc.CreateWithVersion(r.Context(), service.CreateQuestionInput{
		QuestionBankID: bankID,
		Type:           domain.QuestionType(req.Type),
		Content:        req.Content,
		CorrectAnswer:  req.CorrectAnswer,
		Marks:          req.Marks,
		NegativeMarks:  req.NegativeMarks,
		Claims:         *claims,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{
		"question": q,
		"version":  qv,
	}})
}

// ---------------------------------------------------------------------------
// QuestionHandler
// ---------------------------------------------------------------------------

type QuestionHandler struct {
	svc    *service.QuestionService
	logger *slog.Logger
}

func NewQuestionHandler(svc *service.QuestionService, logger *slog.Logger) *QuestionHandler {
	return &QuestionHandler{svc: svc, logger: logger}
}

func (h *QuestionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "questionId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	q, qv, err := h.svc.Get(r.Context(), id)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"question": q,
		"version":  qv,
	}})
}

func (h *QuestionHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "questionId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	qv, err := h.svc.Publish(r.Context(), id, *claims)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": qv})
}

func (h *QuestionHandler) Retire(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "questionId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	if err := h.svc.Retire(r.Context(), id, *claims); err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"status": "RETIRED"}})
}

// ---------------------------------------------------------------------------
// ExamHandler
// ---------------------------------------------------------------------------

type ExamHandler struct {
	svc    *service.ExamService
	logger *slog.Logger
}

func NewExamHandler(svc *service.ExamService, logger *slog.Logger) *ExamHandler {
	return &ExamHandler{svc: svc, logger: logger}
}

func (h *ExamHandler) List(w http.ResponseWriter, r *http.Request) {
	p := api.ParsePagination(r)
	exams, total, err := h.svc.List(r.Context(), p)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, api.PaginatedResponse{
		Data: exams,
		Meta: api.PaginationMeta{Page: p.Page, PerPage: p.PerPage, Total: total},
	})
}

func (h *ExamHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	exam, err := h.svc.Get(r.Context(), id)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": exam})
}

func (h *ExamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title           string         `json:"title"`
		Description     string         `json:"description"`
		DurationSeconds int            `json:"duration_seconds"`
		ScheduledStart  *time.Time     `json:"scheduled_start"`
		ScheduledEnd    *time.Time     `json:"scheduled_end"`
		MaxAttempts     int            `json:"max_attempts"`
		Navigation      map[string]any `json:"navigation"`
		Scoring         map[string]any `json:"scoring"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	exam, err := h.svc.Create(r.Context(), service.CreateExamInput{
		Title:           req.Title,
		Description:     req.Description,
		DurationSeconds: req.DurationSeconds,
		ScheduledStart:  req.ScheduledStart,
		ScheduledEnd:    req.ScheduledEnd,
		MaxAttempts:     req.MaxAttempts,
		Navigation:      req.Navigation,
		Scoring:         req.Scoring,
		Claims:          *claims,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": exam})
}

func (h *ExamHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req service.CreateExamInput
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	exam, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": exam})
}

func (h *ExamHandler) Publish(w http.ResponseWriter, r *http.Request) {
	h.examTransition(w, r, func(ctx context.Context, id domain.ID, c service.AuthClaims) (*models.Exam, error) {
		return h.svc.Publish(ctx, id, c)
	})
}
func (h *ExamHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.examTransition(w, r, h.svc.Start)
}
func (h *ExamHandler) Pause(w http.ResponseWriter, r *http.Request) {
	h.examTransition(w, r, h.svc.Pause)
}
func (h *ExamHandler) Resume(w http.ResponseWriter, r *http.Request) {
	h.examTransition(w, r, h.svc.Resume)
}
func (h *ExamHandler) Complete(w http.ResponseWriter, r *http.Request) {
	h.examTransition(w, r, h.svc.Complete)
}
func (h *ExamHandler) Archive(w http.ResponseWriter, r *http.Request) {
	h.examTransition(w, r, h.svc.Archive)
}

func (h *ExamHandler) examTransition(w http.ResponseWriter, r *http.Request, fn func(context.Context, domain.ID, service.AuthClaims) (*models.Exam, error)) {
	id, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	exam, err := fn(r.Context(), id, *claims)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": exam})
}

func (h *ExamHandler) AddQuestion(w http.ResponseWriter, r *http.Request) {
	examID, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req struct {
		QuestionVersionID string `json:"question_version_id"`
		Position          int    `json:"position"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	qvID, err := domain.ParseID(req.QuestionVersionID)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid question_version_id")
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	eq, err := h.svc.AddQuestion(r.Context(), examID, qvID, req.Position, *claims)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": eq})
}

func (h *ExamHandler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	examID, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	questions, err := h.svc.GetQuestions(r.Context(), examID)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": questions})
}

// ---------------------------------------------------------------------------
// CandidateHandler
// ---------------------------------------------------------------------------

type CandidateHandler struct {
	svc    *service.CandidateService
	logger *slog.Logger
}

func NewCandidateHandler(svc *service.CandidateService, logger *slog.Logger) *CandidateHandler {
	return &CandidateHandler{svc: svc, logger: logger}
}

func (h *CandidateHandler) List(w http.ResponseWriter, r *http.Request) {
	p := api.ParsePagination(r)
	candidates, total, err := h.svc.List(r.Context(), p)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, api.PaginatedResponse{
		Data: candidates,
		Meta: api.PaginationMeta{Page: p.Page, PerPage: p.PerPage, Total: total},
	})
}

func (h *CandidateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "candidateId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": c})
}

func (h *CandidateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		KeycloakSub string `json:"keycloak_sub"`
		Email       string `json:"email"`
		FullName    string `json:"full_name"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	var subID domain.ID
	if req.KeycloakSub != "" {
		subID, _ = domain.ParseID(req.KeycloakSub)
	}
	c, err := h.svc.Create(r.Context(), service.CreateCandidateInput{
		KeycloakSub: subID,
		Email:       req.Email,
		FullName:    req.FullName,
		Claims:      *claims,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": c})
}

// ---------------------------------------------------------------------------
// AssignmentHandler
// ---------------------------------------------------------------------------

type AssignmentHandler struct {
	svc     *service.AssignmentService
	candSvc *service.CandidateService
	logger  *slog.Logger
}

func NewAssignmentHandler(svc *service.AssignmentService, candSvc *service.CandidateService, logger *slog.Logger) *AssignmentHandler {
	return &AssignmentHandler{svc: svc, candSvc: candSvc, logger: logger}
}

func (h *AssignmentHandler) ListMyAssignments(w http.ResponseWriter, r *http.Request) {
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.candSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}

	p := api.ParsePagination(r)
	assignments, total, err := h.svc.ListForCandidate(r.Context(), candidate.ID, p)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, api.PaginatedResponse{
		Data: assignments,
		Meta: api.PaginationMeta{Page: p.Page, PerPage: p.PerPage, Total: total},
	})
}

func (h *AssignmentHandler) List(w http.ResponseWriter, r *http.Request) {
	examID, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	p := api.ParsePagination(r)
	assignments, total, err := h.svc.List(r.Context(), examID, p)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, api.PaginatedResponse{
		Data: assignments,
		Meta: api.PaginationMeta{Page: p.Page, PerPage: p.PerPage, Total: total},
	})
}

func (h *AssignmentHandler) Assign(w http.ResponseWriter, r *http.Request) {
	examID, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req struct {
		CandidateID string `json:"candidate_id"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	candidateID, err := domain.ParseID(req.CandidateID)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid candidate_id")
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	a, err := h.svc.Assign(r.Context(), service.AssignCandidateInput{
		ExamID:      examID,
		CandidateID: candidateID,
		Claims:      *claims,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": a})
}

func (h *AssignmentHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	examID, err := api.ParseIDParam(r, "examId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	candidateID, err := api.ParseIDParam(r, "candidateId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	if err := h.svc.Revoke(r.Context(), examID, candidateID, *claims); err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"status": "REVOKED"}})
}

// ---------------------------------------------------------------------------
// AttemptHandler
// ---------------------------------------------------------------------------

type AttemptHandler struct {
	svc       *service.AttemptService
	cSvc      *service.CandidateService
	rSvc      *service.ResponseService
	subSvc    *service.SubmissionService
	ingestSvc *service.EventIngestionService
	logger    *slog.Logger
}

func NewAttemptHandler(
	svc *service.AttemptService,
	cSvc *service.CandidateService,
	rSvc *service.ResponseService,
	subSvc *service.SubmissionService,
	ingestSvc *service.EventIngestionService,
	logger *slog.Logger,
) *AttemptHandler {
	return &AttemptHandler{
		svc:       svc,
		cSvc:      cSvc,
		rSvc:      rSvc,
		subSvc:    subSvc,
		ingestSvc: ingestSvc,
		logger:    logger,
	}
}

// POST /api/v1/exams/{examId}/attempts — candidate starts an attempt
func (h *AttemptHandler) Start(w http.ResponseWriter, r *http.Request) {
	examID, err := api.ParseIDParam(r, "examId")
	if err != nil {
		h.logger.Error("AttemptHandler.Start parse examId error", "error", err)
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	h.logger.Info("AttemptHandler.Start candidate lookup start", "keycloak_sub", claims.KeycloakSub, "exam_id", examID)
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		h.logger.Error("AttemptHandler.Start candidate lookup failed", "keycloak_sub", claims.KeycloakSub, "error", err)
		api.MapError(w, err)
		return
	}
	h.logger.Info("AttemptHandler.Start candidate lookup succeeded", "candidate_id", candidate.ID)
	attempt, err := h.svc.Start(r.Context(), service.StartAttemptInput{
		ExamID:      examID,
		CandidateID: candidate.ID,
		Claims:      *claims,
	})
	if err != nil {
		h.logger.Error("AttemptHandler.Start AttemptService.Start failed", "error", err)
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": attempt})
}

// GET /api/v1/attempts/{attemptId}/runtime
func (h *AttemptHandler) Runtime(w http.ResponseWriter, r *http.Request) {
	attemptID, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	runtime, err := h.svc.GetRuntime(r.Context(), attemptID, candidate.ID)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": runtime})
}

// GET /api/v1/attempts/{attemptId}/responses
func (h *AttemptHandler) ListResponses(w http.ResponseWriter, r *http.Request) {
	attemptID, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	responses, err := h.rSvc.List(r.Context(), attemptID, candidate.ID)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": responses})
}

// POST /api/v1/attempts/{attemptId}/responses/{questionVersionId}
func (h *AttemptHandler) SaveResponse(w http.ResponseWriter, r *http.Request) {
	attemptID, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	qvID, err := api.ParseIDParam(r, "questionVersionId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req struct {
		Value map[string]any `json:"value"` // null = clear answer
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	resp, err := h.rSvc.Save(r.Context(), service.SaveResponseInput{
		AttemptID:         attemptID,
		CandidateID:       candidate.ID,
		QuestionVersionID: qvID,
		Value:             req.Value,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": resp})
}

// POST /api/v1/attempts/{attemptId}/submit
func (h *AttemptHandler) Submit(w http.ResponseWriter, r *http.Request) {
	attemptID, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	sub, err := h.subSvc.Submit(r.Context(), attemptID, candidate.ID)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": sub})
}

// POST /api/v1/attempts/{attemptId}/events:sync
func (h *AttemptHandler) SyncEvents(w http.ResponseWriter, r *http.Request) {
	attemptID, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req struct {
		Events []service.SyncEventItemInput `json:"events"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	res, err := h.ingestSvc.SyncEvents(r.Context(), service.SyncEventsInput{
		AttemptID: attemptID,
		Events:    req.Events,
		Claims:    *claims,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": res})
}

// Admin: GET /api/v1/attempts
func (h *AttemptHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	p := api.ParsePagination(r)
	attempts, total, err := h.svc.ListForAdmin(r.Context(), p)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, api.PaginatedResponse{
		Data: attempts,
		Meta: api.PaginationMeta{Page: p.Page, PerPage: p.PerPage, Total: total},
	})
}

// Admin: GET /api/v1/attempts/{attemptId}
func (h *AttemptHandler) AdminGet(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	attempt, err := h.svc.GetForAdmin(r.Context(), id)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": attempt})
}

// ---------------------------------------------------------------------------
// SessionHandler
// ---------------------------------------------------------------------------

type SessionHandler struct {
	svc    *service.SessionService
	cSvc   *service.CandidateService
	logger *slog.Logger
}

func NewSessionHandler(svc *service.SessionService, cSvc *service.CandidateService, logger *slog.Logger) *SessionHandler {
	return &SessionHandler{svc: svc, cSvc: cSvc, logger: logger}
}

// POST /api/v1/attempts/{attemptId}/sessions
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	attemptID, err := api.ParseIDParam(r, "attemptId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var req struct {
		ClientInstanceID string `json:"client_instance_id"`
	}
	if err := api.DecodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if req.ClientInstanceID == "" {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "client_instance_id is required")
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	sess, err := h.svc.Create(r.Context(), service.CreateSessionInput{
		AttemptID:        attemptID,
		CandidateID:      candidate.ID,
		ClientInstanceID: req.ClientInstanceID,
	})
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"data": sess})
}

// POST /api/v1/sessions/{sessionId}/heartbeat
func (h *SessionHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	sessionID, err := api.ParseIDParam(r, "sessionId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	sess, err := h.svc.Heartbeat(r.Context(), sessionID, candidate.ID)
	if err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"session_id":       sess.ID,
		"status":           sess.Status,
		"last_heartbeat_at": sess.LastHeartbeatAt,
	}})
}

// POST /api/v1/sessions/{sessionId}/terminate
func (h *SessionHandler) Terminate(w http.ResponseWriter, r *http.Request) {
	sessionID, err := api.ParseIDParam(r, "sessionId")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	claims := authmw.ClaimsFromCtx(r.Context())
	candidate, err := h.cSvc.GetByUserID(r.Context(), claims.KeycloakSub)
	if err != nil {
		api.MapError(w, err)
		return
	}
	if err := h.svc.Terminate(r.Context(), sessionID, candidate.ID); err != nil {
		api.MapError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"status": "TERMINATED"}})
}
