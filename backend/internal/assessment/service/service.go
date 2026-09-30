// Package service implements the assessment core business logic.
// Services coordinate domain validation, repository reads/writes, and state transitions.
// They have NO knowledge of HTTP or serialization formats.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
	"github.com/sentinelexam/backend/internal/assessment/repository"
	repopg "github.com/sentinelexam/backend/internal/assessment/repository/postgres"
	"github.com/sentinelexam/backend/internal/assessment/scoring"
	"github.com/sentinelexam/backend/internal/platform/api"
)

// ---------------------------------------------------------------------------
// Auth context helpers
// ---------------------------------------------------------------------------

type AuthClaims struct {
	Sub         string
	KeycloakSub domain.ID
	Email       string
	DisplayName string
	Roles       []string
}

func (a AuthClaims) HasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (a AuthClaims) IsAdmin() bool {
	return a.HasRole("exam_admin") || a.HasRole("system_admin")
}

func (a AuthClaims) IsCandidate() bool { return a.HasRole("candidate") }

// ---------------------------------------------------------------------------
// QuestionBankService
// ---------------------------------------------------------------------------

type QuestionBankService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewQuestionBankService(repos repository.Repositories, logger *slog.Logger) *QuestionBankService {
	return &QuestionBankService{repos: repos, logger: logger}
}

func (s *QuestionBankService) List(ctx context.Context, p domain.Pagination) ([]models.QuestionBank, int, error) {
	return s.repos.QuestionBanks.List(ctx, p)
}

func (s *QuestionBankService) Get(ctx context.Context, id domain.ID) (*models.QuestionBank, error) {
	return s.repos.QuestionBanks.FindByID(ctx, id)
}

type CreateQuestionBankInput struct {
	Name        string
	Description string
	Claims      AuthClaims
}

func (s *QuestionBankService) Create(ctx context.Context, in CreateQuestionBankInput) (*models.QuestionBank, error) {
	if in.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}
	userID, err := s.ensureUser(ctx, in.Claims)
	if err != nil {
		return nil, err
	}
	qb := &models.QuestionBank{
		Name:        in.Name,
		Description: in.Description,
		Status:      domain.QuestionBankStatusActive,
		CreatedBy:   &userID,
	}
	if err := s.repos.QuestionBanks.Create(ctx, qb); err != nil {
		return nil, fmt.Errorf("create question bank: %w", err)
	}
	return qb, nil
}

// ---------------------------------------------------------------------------
// QuestionService
// ---------------------------------------------------------------------------

type QuestionService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewQuestionService(repos repository.Repositories, logger *slog.Logger) *QuestionService {
	return &QuestionService{repos: repos, logger: logger}
}

type CreateQuestionInput struct {
	QuestionBankID domain.ID
	Type           domain.QuestionType
	Content        map[string]any
	CorrectAnswer  map[string]any
	Marks          float64
	NegativeMarks  float64
	Claims         AuthClaims
}

func (s *QuestionService) CreateWithVersion(ctx context.Context, in CreateQuestionInput) (*models.Question, *models.QuestionVersion, error) {
	if err := validateQuestionType(in.Type); err != nil {
		return nil, nil, err
	}
	if in.Marks <= 0 {
		return nil, nil, domain.NewValidationError("marks", "marks must be positive")
	}
	if _, err := s.repos.QuestionBanks.FindByID(ctx, in.QuestionBankID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: question bank", domain.ErrNotFound)
		}
		return nil, nil, err
	}
	userID, err := s.ensureUser(ctx, in.Claims)
	if err != nil {
		return nil, nil, err
	}
	q := &models.Question{
		QuestionBankID: in.QuestionBankID,
		Status:         domain.QuestionStatusDraft,
		CreatedBy:      &userID,
	}
	if err := s.repos.Questions.Create(ctx, q); err != nil {
		return nil, nil, fmt.Errorf("create question: %w", err)
	}
	qv := &models.QuestionVersion{
		QuestionID:    q.ID,
		Version:       1,
		Type:          in.Type,
		Content:       in.Content,
		CorrectAnswer: in.CorrectAnswer,
		Marks:         in.Marks,
		NegativeMarks: in.NegativeMarks,
		Status:        domain.QuestionStatusDraft,
		CreatedBy:     &userID,
	}
	if err := s.repos.QuestionVersions.Create(ctx, qv); err != nil {
		return nil, nil, fmt.Errorf("create question version: %w", err)
	}
	q.CurrentVersionID = &qv.ID
	if err := s.repos.Questions.Update(ctx, q); err != nil {
		return nil, nil, err
	}
	return q, qv, nil
}

func (s *QuestionService) Get(ctx context.Context, id domain.ID) (*models.Question, *models.QuestionVersion, error) {
	q, err := s.repos.Questions.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	var qv *models.QuestionVersion
	if q.CurrentVersionID != nil {
		qv, err = s.repos.QuestionVersions.FindByID(ctx, *q.CurrentVersionID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, nil, err
		}
	}
	return q, qv, nil
}

func (s *QuestionService) Publish(ctx context.Context, questionID domain.ID, _ AuthClaims) (*models.QuestionVersion, error) {
	q, err := s.repos.Questions.FindByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if q.CurrentVersionID == nil {
		return nil, fmt.Errorf("%w: no version to publish", domain.ErrValidation)
	}
	qv, err := s.repos.QuestionVersions.FindByID(ctx, *q.CurrentVersionID)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidQuestionTransition(qv.Status, domain.QuestionStatusActive); err != nil {
		return nil, err
	}
	if err := s.repos.QuestionVersions.UpdateStatus(ctx, qv.ID, domain.QuestionStatusActive); err != nil {
		return nil, err
	}
	q.Status = domain.QuestionStatusActive
	if err := s.repos.Questions.Update(ctx, q); err != nil {
		return nil, err
	}
	qv.Status = domain.QuestionStatusActive
	return qv, nil
}

func (s *QuestionService) Retire(ctx context.Context, questionID domain.ID, _ AuthClaims) error {
	q, err := s.repos.Questions.FindByID(ctx, questionID)
	if err != nil {
		return err
	}
	if err := domain.ValidQuestionTransition(q.Status, domain.QuestionStatusRetired); err != nil {
		return err
	}
	q.Status = domain.QuestionStatusRetired
	return s.repos.Questions.Update(ctx, q)
}

// ---------------------------------------------------------------------------
// ExamService
// ---------------------------------------------------------------------------

type ExamService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewExamService(repos repository.Repositories, logger *slog.Logger) *ExamService {
	return &ExamService{repos: repos, logger: logger}
}

func (s *ExamService) List(ctx context.Context, p domain.Pagination) ([]models.Exam, int, error) {
	return s.repos.Exams.List(ctx, p)
}

func (s *ExamService) Get(ctx context.Context, id domain.ID) (*models.Exam, error) {
	return s.repos.Exams.FindByID(ctx, id)
}

type CreateExamInput struct {
	Title           string
	Description     string
	DurationSeconds int
	ScheduledStart  *time.Time
	ScheduledEnd    *time.Time
	MaxAttempts     int
	Navigation      map[string]any
	Scoring         map[string]any
	Claims          AuthClaims
}

func (s *ExamService) Create(ctx context.Context, in CreateExamInput) (*models.Exam, error) {
	if in.Title == "" {
		return nil, domain.NewValidationError("title", "title is required")
	}
	if in.DurationSeconds <= 0 {
		return nil, domain.NewValidationError("duration_seconds", "duration must be positive")
	}
	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	nav := in.Navigation
	if nav == nil {
		nav = map[string]any{"allow_back": true, "allow_skip": true}
	}
	sc := in.Scoring
	if sc == nil {
		sc = map[string]any{"negative_marking": false, "pass_percentage": nil}
	}
	userID, err := s.ensureUser(ctx, in.Claims)
	if err != nil {
		return nil, err
	}
	e := &models.Exam{
		Title:           in.Title,
		Description:     in.Description,
		Status:          domain.ExamStatusDraft,
		DurationSeconds: in.DurationSeconds,
		ScheduledStart:  in.ScheduledStart,
		ScheduledEnd:    in.ScheduledEnd,
		MaxAttempts:     maxAttempts,
		Navigation:      nav,
		Scoring:         sc,
		Version:         1,
		CreatedBy:       &userID,
	}
	if err := s.repos.Exams.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create exam: %w", err)
	}
	return e, nil
}

func (s *ExamService) Update(ctx context.Context, id domain.ID, in CreateExamInput) (*models.Exam, error) {
	e, err := s.repos.Exams.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.Status != domain.ExamStatusDraft {
		return nil, fmt.Errorf("%w: can only update DRAFT exams", domain.ErrInvalidTransition)
	}
	if in.Title != "" {
		e.Title = in.Title
	}
	if in.Description != "" {
		e.Description = in.Description
	}
	if in.DurationSeconds > 0 {
		e.DurationSeconds = in.DurationSeconds
	}
	if in.MaxAttempts > 0 {
		e.MaxAttempts = in.MaxAttempts
	}
	if in.ScheduledStart != nil {
		e.ScheduledStart = in.ScheduledStart
	}
	if in.ScheduledEnd != nil {
		e.ScheduledEnd = in.ScheduledEnd
	}
	if in.Navigation != nil {
		e.Navigation = in.Navigation
	}
	if in.Scoring != nil {
		e.Scoring = in.Scoring
	}
	if err := s.repos.Exams.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *ExamService) Publish(ctx context.Context, id domain.ID, _ AuthClaims) (*models.Exam, error) {
	e, err := s.repos.Exams.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidExamTransition(e.Status, domain.ExamStatusScheduled); err != nil {
		return nil, err
	}
	// Publish validation
	count, err := s.repos.ExamQuestions.CountByExam(ctx, id)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("%w: exam must have at least one question", domain.ErrExamPublishValidation)
	}
	if e.DurationSeconds <= 0 {
		return nil, fmt.Errorf("%w: invalid duration", domain.ErrExamPublishValidation)
	}
	if err := s.repos.Exams.UpdateStatus(ctx, id, domain.ExamStatusScheduled, e.Version); err != nil {
		return nil, err
	}
	e.Status = domain.ExamStatusScheduled
	return e, nil
}

func (s *ExamService) transitionExam(ctx context.Context, id domain.ID, to domain.ExamStatus) (*models.Exam, error) {
	e, err := s.repos.Exams.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidExamTransition(e.Status, to); err != nil {
		return nil, err
	}
	if err := s.repos.Exams.UpdateStatus(ctx, id, to, e.Version); err != nil {
		return nil, err
	}
	e.Status = to
	return e, nil
}

func (s *ExamService) Start(ctx context.Context, id domain.ID, c AuthClaims) (*models.Exam, error) {
	return s.transitionExam(ctx, id, domain.ExamStatusLive)
}
func (s *ExamService) Pause(ctx context.Context, id domain.ID, c AuthClaims) (*models.Exam, error) {
	return s.transitionExam(ctx, id, domain.ExamStatusPaused)
}
func (s *ExamService) Resume(ctx context.Context, id domain.ID, c AuthClaims) (*models.Exam, error) {
	return s.transitionExam(ctx, id, domain.ExamStatusLive)
}
func (s *ExamService) Complete(ctx context.Context, id domain.ID, c AuthClaims) (*models.Exam, error) {
	return s.transitionExam(ctx, id, domain.ExamStatusCompleted)
}
func (s *ExamService) Archive(ctx context.Context, id domain.ID, c AuthClaims) (*models.Exam, error) {
	return s.transitionExam(ctx, id, domain.ExamStatusArchived)
}

func (s *ExamService) AddQuestion(ctx context.Context, examID, questionVersionID domain.ID, position int, _ AuthClaims) (*models.ExamQuestion, error) {
	e, err := s.repos.Exams.FindByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	if e.Status != domain.ExamStatusDraft {
		return nil, fmt.Errorf("%w: can only add questions to DRAFT exams", domain.ErrInvalidTransition)
	}
	qv, err := s.repos.QuestionVersions.FindByID(ctx, questionVersionID)
	if err != nil {
		return nil, err
	}
	if qv.Status != domain.QuestionStatusActive {
		return nil, fmt.Errorf("%w: only ACTIVE question versions can be added", domain.ErrValidation)
	}
	eq := &models.ExamQuestion{
		ExamID:            examID,
		QuestionVersionID: questionVersionID,
		Position:          position,
	}
	if err := s.repos.ExamQuestions.Add(ctx, eq); err != nil {
		return nil, err
	}
	return eq, nil
}

func (s *ExamService) GetQuestions(ctx context.Context, examID domain.ID) ([]models.ExamQuestionDetail, error) {
	return s.repos.ExamQuestions.FindByExam(ctx, examID)
}

// ---------------------------------------------------------------------------
// CandidateService
// ---------------------------------------------------------------------------

type CandidateService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewCandidateService(repos repository.Repositories, logger *slog.Logger) *CandidateService {
	return &CandidateService{repos: repos, logger: logger}
}

func (s *CandidateService) List(ctx context.Context, p domain.Pagination) ([]models.Candidate, int, error) {
	return s.repos.Candidates.List(ctx, p)
}

func (s *CandidateService) Get(ctx context.Context, id domain.ID) (*models.Candidate, error) {
	return s.repos.Candidates.FindByID(ctx, id)
}

func (s *CandidateService) GetByUserID(ctx context.Context, keycloakSub domain.ID) (*models.Candidate, error) {
	user, err := s.repos.Users.FindByKeycloakSub(ctx, keycloakSub)
	if err != nil {
		return nil, err
	}
	return s.repos.Candidates.FindByUserID(ctx, user.ID)
}

type CreateCandidateInput struct {
	KeycloakSub domain.ID
	Email       string
	FullName    string
	Claims      AuthClaims
}

func (s *CandidateService) Create(ctx context.Context, in CreateCandidateInput) (*models.Candidate, error) {
	if in.FullName == "" {
		return nil, domain.NewValidationError("full_name", "full_name is required")
	}

	claims := in.Claims
	if in.KeycloakSub != (domain.ID{}) {
		claims.KeycloakSub = in.KeycloakSub
	}
	if in.Email != "" {
		claims.Email = in.Email
	}

	userID, err := s.ensureUser(ctx, claims)
	if err != nil {
		return nil, err
	}

	existing, err := s.repos.Candidates.FindByUserID(ctx, userID)
	if err == nil && existing != nil {
		return existing, nil
	}

	c := &models.Candidate{
		ID:       domain.NewID(),
		UserID:   userID,
		Status:   domain.CandidateStatusActive,
		FullName: in.FullName,
		Metadata: map[string]any{},
	}
	if err := s.repos.Candidates.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ---------------------------------------------------------------------------
// AssignmentService
// ---------------------------------------------------------------------------

type AssignmentService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewAssignmentService(repos repository.Repositories, logger *slog.Logger) *AssignmentService {
	return &AssignmentService{repos: repos, logger: logger}
}

func (s *AssignmentService) List(ctx context.Context, examID domain.ID, p domain.Pagination) ([]models.ExamAssignment, int, error) {
	return s.repos.Assignments.ListByExam(ctx, examID, p)
}

func (s *AssignmentService) ListForCandidate(ctx context.Context, candidateID domain.ID, p domain.Pagination) ([]models.ExamAssignment, int, error) {
	return s.repos.Assignments.ListByCandidate(ctx, candidateID, p)
}

type AssignCandidateInput struct {
	ExamID      domain.ID
	CandidateID domain.ID
	Claims      AuthClaims
}

func (s *AssignmentService) Assign(ctx context.Context, in AssignCandidateInput) (*models.ExamAssignment, error) {
	e, err := s.repos.Exams.FindByID(ctx, in.ExamID)
	if err != nil {
		return nil, err
	}
	if e.Status != domain.ExamStatusLive && e.Status != domain.ExamStatusScheduled {
		return nil, fmt.Errorf("%w: can only assign to SCHEDULED or LIVE exams", domain.ErrValidation)
	}
	if _, err := s.repos.Candidates.FindByID(ctx, in.CandidateID); err != nil {
		return nil, fmt.Errorf("%w: candidate", domain.ErrNotFound)
	}
	// Check existing assignment
	existing, err := s.repos.Assignments.FindByExamAndCandidate(ctx, in.ExamID, in.CandidateID)
	if err == nil {
		if existing.Status == domain.AssignmentStatusAssigned {
			return existing, nil // already assigned — idempotent
		}
		// Was revoked, re-assign
		if err := s.repos.Assignments.UpdateStatus(ctx, existing.ID, domain.AssignmentStatusAssigned, nil); err != nil {
			return nil, err
		}
		existing.Status = domain.AssignmentStatusAssigned
		existing.RevokedAt = nil
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	userID, err := s.ensureUser(ctx, in.Claims)
	if err != nil {
		return nil, err
	}
	a := &models.ExamAssignment{
		ExamID:      in.ExamID,
		CandidateID: in.CandidateID,
		Status:      domain.AssignmentStatusAssigned,
		AssignedBy:  &userID,
	}
	if err := s.repos.Assignments.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AssignmentService) Revoke(ctx context.Context, examID, candidateID domain.ID, _ AuthClaims) error {
	a, err := s.repos.Assignments.FindByExamAndCandidate(ctx, examID, candidateID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.repos.Assignments.UpdateStatus(ctx, a.ID, domain.AssignmentStatusRevoked, now)
}

// ---------------------------------------------------------------------------
// AttemptService
// ---------------------------------------------------------------------------

type AttemptService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewAttemptService(repos repository.Repositories, logger *slog.Logger) *AttemptService {
	return &AttemptService{repos: repos, logger: logger}
}

func (s *AttemptService) GetForAdmin(ctx context.Context, id domain.ID) (*models.ExamAttempt, error) {
	return s.repos.Attempts.FindByID(ctx, id)
}

func (s *AttemptService) ListForAdmin(ctx context.Context, p domain.Pagination) ([]models.ExamAttempt, int, error) {
	return s.repos.Attempts.List(ctx, p)
}

type StartAttemptInput struct {
	ExamID      domain.ID
	CandidateID domain.ID // derived from JWT — NOT from client payload
	Claims      AuthClaims
}

func (s *AttemptService) Start(ctx context.Context, in StartAttemptInput) (*models.ExamAttempt, error) {
	// Verify exam is LIVE
	exam, err := s.repos.Exams.FindByID(ctx, in.ExamID)
	if err != nil {
		s.logger.Error("AttemptService.Start exam lookup failed", "exam_id", in.ExamID, "error", err)
		return nil, err
	}
	s.logger.Info("AttemptService.Start exam lookup succeeded", "exam_id", exam.ID, "status", exam.Status)
	if exam.Status != domain.ExamStatusLive {
		s.logger.Error("AttemptService.Start exam not LIVE", "status", exam.Status)
		return nil, fmt.Errorf("%w: exam is not LIVE", domain.ErrValidation)
	}

	// Verify assignment
	assignment, err := s.repos.Assignments.FindByExamAndCandidate(ctx, in.ExamID, in.CandidateID)
	if err != nil {
		s.logger.Error("AttemptService.Start assignment lookup failed", "exam_id", in.ExamID, "candidate_id", in.CandidateID, "error", err)
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, err
	}
	s.logger.Info("AttemptService.Start assignment lookup succeeded", "assignment_id", assignment.ID, "status", assignment.Status)
	if assignment.Status == domain.AssignmentStatusRevoked {
		return nil, domain.ErrAssignmentRevoked
	}

	// Check max attempts
	count, err := s.repos.Attempts.CountByAssignment(ctx, assignment.ID)
	if err != nil {
		return nil, err
	}
	if count >= exam.MaxAttempts {
		return nil, fmt.Errorf("%w: max attempts reached", domain.ErrValidation)
	}

	now := time.Now().UTC()
	expiresAt := domain.ComputeExpiresAt(now, exam.DurationSeconds)

	a := &models.ExamAttempt{
		ExamID:       in.ExamID,
		CandidateID:  in.CandidateID,
		AssignmentID: assignment.ID,
		Status:       domain.AttemptStatusActive,
		StartedAt:    &now,
		ExpiresAt:    &expiresAt,
		SessionEpoch: 0,
		Version:      1,
	}
	if err := s.repos.Attempts.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create attempt: %w", err)
	}

	// Pre-create UNANSWERED responses for all questions
	questions, err := s.repos.ExamQuestions.FindByExam(ctx, in.ExamID)
	if err != nil {
		return nil, err
	}
	for _, eq := range questions {
		cr := &models.CandidateResponse{
			AttemptID:         a.ID,
			QuestionVersionID: eq.QuestionVersionID,
			Status:            domain.ResponseStatusUnanswered,
			Version:           1,
		}
		if err := s.repos.Responses.Upsert(ctx, cr); err != nil {
			s.logger.Error("failed to create initial response", "question_version_id", eq.QuestionVersionID, "error", err)
		}
	}
	return a, nil
}

func (s *AttemptService) GetRuntime(ctx context.Context, attemptID, candidateID domain.ID) (*models.AttemptRuntime, error) {
	attempt, err := s.repos.Attempts.FindByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	// Ownership check — never trust client-supplied candidate_id
	if attempt.CandidateID != candidateID {
		return nil, domain.ErrNotFound // 404 to avoid enumeration
	}
	// Check expiry — auto-expire if needed
	if attempt.Status == domain.AttemptStatusActive && domain.IsExpired(attempt.ExpiresAt) {
		_ = domain.ValidAttemptTransition(domain.AttemptStatusActive, domain.AttemptStatusExpired) // known valid
		attempt.Status = domain.AttemptStatusExpired
		if err := s.repos.Attempts.Update(ctx, attempt); err != nil {
			s.logger.Error("failed to mark attempt expired", "attempt_id", attemptID, "error", err)
		}
	}
	exam, err := s.repos.Exams.FindByID(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	questions, err := s.repos.ExamQuestions.FindByExam(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	// Strip correct_answer from question versions (never expose to candidate)
	for i := range questions {
		questions[i].QuestionVersion.CorrectAnswer = nil
	}
	responses, err := s.repos.Responses.ListByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	session, _ := s.repos.Sessions.FindActiveByAttempt(ctx, attemptID)

	return &models.AttemptRuntime{
		Attempt:   *attempt,
		Exam:      *exam,
		Questions: questions,
		Responses: responses,
		Session:   session,
	}, nil
}

// ---------------------------------------------------------------------------
// SessionService
// ---------------------------------------------------------------------------

type SessionService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewSessionService(repos repository.Repositories, logger *slog.Logger) *SessionService {
	return &SessionService{repos: repos, logger: logger}
}

type CreateSessionInput struct {
	AttemptID        domain.ID
	CandidateID      domain.ID
	ClientInstanceID string
}

func (s *SessionService) Create(ctx context.Context, in CreateSessionInput) (*models.AssessmentSession, error) {
	attempt, err := s.repos.Attempts.FindByID(ctx, in.AttemptID)
	if err != nil {
		return nil, err
	}
	if attempt.CandidateID != in.CandidateID {
		return nil, domain.ErrNotFound
	}
	if attempt.Status != domain.AttemptStatusActive {
		return nil, fmt.Errorf("%w: attempt is not active", domain.ErrAttemptNotActive)
	}
	// Terminate any existing active sessions for this attempt
	if existing, err := s.repos.Sessions.FindActiveByAttempt(ctx, in.AttemptID); err == nil {
		existing.Status = domain.SessionStatusTerminated
		now := time.Now().UTC()
		existing.TerminatedAt = &now
		_ = s.repos.Sessions.Update(ctx, existing)
	}

	attempt.SessionEpoch++
	now := time.Now().UTC()
	sess := &models.AssessmentSession{
		AttemptID:        in.AttemptID,
		ClientInstanceID: in.ClientInstanceID,
		SessionEpoch:     attempt.SessionEpoch,
		Status:           domain.SessionStatusConnected,
		ConnectedAt:      &now,
		LastHeartbeatAt:  &now,
	}
	if err := s.repos.Sessions.Create(ctx, sess); err != nil {
		return nil, err
	}
	if err := s.repos.Attempts.Update(ctx, attempt); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *SessionService) Heartbeat(ctx context.Context, sessionID, candidateID domain.ID) (*models.AssessmentSession, error) {
	sess, err := s.repos.Sessions.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	attempt, err := s.repos.Attempts.FindByID(ctx, sess.AttemptID)
	if err != nil {
		return nil, err
	}
	if attempt.CandidateID != candidateID {
		return nil, domain.ErrNotFound
	}
	if domain.IsExpired(attempt.ExpiresAt) {
		return nil, domain.ErrAttemptExpired
	}
	now := time.Now().UTC()
	sess.LastHeartbeatAt = &now
	if sess.Status == domain.SessionStatusDisconnected || sess.Status == domain.SessionStatusCreated {
		sess.Status = domain.SessionStatusConnected
	}
	if err := s.repos.Sessions.Update(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *SessionService) Terminate(ctx context.Context, sessionID, candidateID domain.ID) error {
	sess, err := s.repos.Sessions.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}
	attempt, err := s.repos.Attempts.FindByID(ctx, sess.AttemptID)
	if err != nil {
		return err
	}
	if attempt.CandidateID != candidateID {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	sess.Status = domain.SessionStatusTerminated
	sess.TerminatedAt = &now
	return s.repos.Sessions.Update(ctx, sess)
}

// ---------------------------------------------------------------------------
// ResponseService
// ---------------------------------------------------------------------------

type ResponseService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewResponseService(repos repository.Repositories, logger *slog.Logger) *ResponseService {
	return &ResponseService{repos: repos, logger: logger}
}

type SaveResponseInput struct {
	AttemptID         domain.ID
	CandidateID       domain.ID
	QuestionVersionID domain.ID
	Value             map[string]any // nil = clear answer
}

func (s *ResponseService) Save(ctx context.Context, in SaveResponseInput) (*models.CandidateResponse, error) {
	attempt, err := s.repos.Attempts.FindByID(ctx, in.AttemptID)
	if err != nil {
		return nil, err
	}
	if attempt.CandidateID != in.CandidateID {
		return nil, domain.ErrNotFound
	}
	if attempt.Status != domain.AttemptStatusActive {
		return nil, domain.ErrAttemptNotActive
	}
	if domain.IsExpired(attempt.ExpiresAt) {
		return nil, domain.ErrAttemptExpired
	}
	// Determine new status
	status := domain.ResponseStatusUnanswered
	if in.Value != nil {
		status = domain.ResponseStatusAnswered
	}
	cr := &models.CandidateResponse{
		AttemptID:         in.AttemptID,
		QuestionVersionID: in.QuestionVersionID,
		CurrentValue:      in.Value,
		Status:            status,
		Version:           1,
	}
	if err := s.repos.Responses.Upsert(ctx, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (s *ResponseService) List(ctx context.Context, attemptID, candidateID domain.ID) ([]models.CandidateResponse, error) {
	attempt, err := s.repos.Attempts.FindByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.CandidateID != candidateID {
		return nil, domain.ErrNotFound
	}
	return s.repos.Responses.ListByAttempt(ctx, attemptID)
}

// ---------------------------------------------------------------------------
// SubmissionService
// ---------------------------------------------------------------------------

type SubmissionService struct {
	repos  repository.Repositories
	logger *slog.Logger
}

func NewSubmissionService(repos repository.Repositories, logger *slog.Logger) *SubmissionService {
	return &SubmissionService{repos: repos, logger: logger}
}

func (s *SubmissionService) Submit(ctx context.Context, attemptID, candidateID domain.ID) (*models.Submission, error) {
	// Idempotency: check if submission already exists
	if existing, err := s.repos.Submissions.FindByAttempt(ctx, attemptID); err == nil {
		return existing, nil // duplicate submission — return existing
	}

	attempt, err := s.repos.Attempts.FindByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.CandidateID != candidateID {
		return nil, domain.ErrNotFound
	}

	// Check expiry and auto-expire
	if attempt.Status == domain.AttemptStatusActive && domain.IsExpired(attempt.ExpiresAt) {
		attempt.Status = domain.AttemptStatusExpired
		_ = s.repos.Attempts.Update(ctx, attempt)
	}

	if attempt.Status != domain.AttemptStatusActive {
		return nil, fmt.Errorf("%w: attempt status is %s", domain.ErrAttemptNotActive, attempt.Status)
	}

	exam, err := s.repos.Exams.FindByID(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	questions, err := s.repos.ExamQuestions.FindByExam(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	responses, err := s.repos.Responses.ListByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	// Deterministic server-side scoring
	totalScore, maxScore := scoring.ScoreAttempt(questions, responses)

	now := time.Now().UTC()
	sub := &models.Submission{
		AttemptID:   attemptID,
		Status:      domain.SubmissionStatusFinalized,
		RequestedAt: now,
		SubmittedAt: &now,
		FinalizedAt: &now,
		Score:       &totalScore,
		MaxScore:    &maxScore,
	}
	if err := s.repos.Submissions.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("create submission: %w", err)
	}

	// Freeze responses
	if err := s.repos.Responses.UpdateStatus(ctx, attemptID, domain.ResponseStatusSubmitted); err != nil {
		s.logger.Error("failed to freeze responses", "attempt_id", attemptID, "error", err)
	}

	// Transition attempt
	_ = domain.ValidAttemptTransition(domain.AttemptStatusActive, domain.AttemptStatusFinalized)
	attempt.Status = domain.AttemptStatusFinalized
	attempt.SubmittedAt = &now
	attempt.FinalizedAt = &now
	if err := s.repos.Attempts.Update(ctx, attempt); err != nil {
		s.logger.Error("failed to finalize attempt", "attempt_id", attemptID, "error", err)
	}

	_ = exam // suppress unused (used via scoring above)
	return sub, nil
}

// ---------------------------------------------------------------------------
// Shared user-resolution helper
// ---------------------------------------------------------------------------

type userResolver interface {
	ensureUser(ctx context.Context, claims AuthClaims) (domain.ID, error)
}

func ensureUserFn(repos repository.Repositories) func(ctx context.Context, claims AuthClaims) (domain.ID, error) {
	return func(ctx context.Context, claims AuthClaims) (domain.ID, error) {
		user := &models.User{
			KeycloakSub: claims.KeycloakSub,
			Email:       claims.Email,
			DisplayName: claims.DisplayName,
			Roles:       claims.Roles,
		}
		if err := repos.Users.Upsert(ctx, user); err != nil {
			return domain.ID{}, fmt.Errorf("upsert user: %w", err)
		}
		found, err := repos.Users.FindByKeycloakSub(ctx, claims.KeycloakSub)
		if err != nil {
			return domain.ID{}, err
		}
		return found.ID, nil
	}
}

// attach ensureUser to each service via embedding
func (s *QuestionBankService) ensureUser(ctx context.Context, c AuthClaims) (domain.ID, error) {
	return ensureUserFn(s.repos)(ctx, c)
}
func (s *QuestionService) ensureUser(ctx context.Context, c AuthClaims) (domain.ID, error) {
	return ensureUserFn(s.repos)(ctx, c)
}
func (s *ExamService) ensureUser(ctx context.Context, c AuthClaims) (domain.ID, error) {
	return ensureUserFn(s.repos)(ctx, c)
}
func (s *CandidateService) ensureUser(ctx context.Context, c AuthClaims) (domain.ID, error) {
	return ensureUserFn(s.repos)(ctx, c)
}
func (s *AssignmentService) ensureUser(ctx context.Context, c AuthClaims) (domain.ID, error) {
	return ensureUserFn(s.repos)(ctx, c)
}

func validateQuestionType(t domain.QuestionType) error {
	switch t {
	case domain.QuestionTypeSingleChoice,
		domain.QuestionTypeMultipleChoice,
		domain.QuestionTypeTrueFalse,
		domain.QuestionTypeText,
		domain.QuestionTypeNumeric:
		return nil
	}
	return domain.NewValidationError("type", fmt.Sprintf("unsupported question type: %s", t))
}

// ---------------------------------------------------------------------------
// EventIngestionService
// ---------------------------------------------------------------------------

type EventIngestionService struct {
	pool   *pgxpool.Pool
	repos  repository.Repositories
	logger *slog.Logger
}

func NewEventIngestionService(pool *pgxpool.Pool, repos repository.Repositories, logger *slog.Logger) *EventIngestionService {
	return &EventIngestionService{
		pool:   pool,
		repos:  repos,
		logger: logger,
	}
}

type SyncEventItemInput struct {
	EventID          domain.ID      `json:"event_id"`
	SessionID        *domain.ID     `json:"session_id"`
	SessionEpoch     int            `json:"session_epoch"`
	SequenceNumber   int64          `json:"sequence_number"`
	EventType        string         `json:"event_type"`
	SchemaVersion    string         `json:"schema_version"`
	ClientInstanceID string         `json:"client_instance_id"`
	ClientTimestamp  time.Time      `json:"client_timestamp"`
	Payload          map[string]any `json:"payload"`
}

type SyncEventsInput struct {
	AttemptID domain.ID
	Events    []SyncEventItemInput
	Claims    AuthClaims
}

type SyncEventsOutput struct {
	AcceptedCount         int       `json:"accepted_count"`
	LastCommittedSequence int64     `json:"last_committed_sequence"`
	ServerTimestamp       time.Time `json:"server_timestamp"`
}

func (s *EventIngestionService) SyncEvents(ctx context.Context, in SyncEventsInput) (*SyncEventsOutput, error) {
	if in.AttemptID == (domain.ID{}) {
		return nil, domain.NewValidationError("attempt_id", "attempt_id is required")
	}

	// 1. Authenticate / authorize ownership
	attempt, err := s.repos.Attempts.FindByID(ctx, in.AttemptID)
	if err != nil {
		return nil, err
	}

	if in.Claims.IsCandidate() {
		user, err := s.repos.Users.FindByKeycloakSub(ctx, in.Claims.KeycloakSub)
		if err != nil {
			return nil, domain.ErrForbidden
		}
		candidate, err := s.repos.Candidates.FindByUserID(ctx, user.ID)
		if err != nil {
			return nil, domain.ErrForbidden
		}
		if attempt.CandidateID != candidate.ID {
			return nil, domain.ErrNotFound // Return 404 to prevent enumeration
		}
	} else if !in.Claims.IsAdmin() {
		return nil, domain.ErrForbidden
	}

	// If empty batch, return current committed state immediately
	if len(in.Events) == 0 {
		maxSeq, err := s.repos.CandidateEvents.GetMaxSequence(ctx, in.AttemptID)
		if err != nil {
			return nil, err
		}
		return &SyncEventsOutput{
			AcceptedCount:         0,
			LastCommittedSequence: maxSeq,
			ServerTimestamp:       time.Now().UTC(),
		}, nil
	}

	// 2. Validate input batch fields upfront
	for i, ev := range in.Events {
		if ev.EventID == (domain.ID{}) {
			return nil, domain.NewValidationError(fmt.Sprintf("events[%d].event_id", i), "event_id is required")
		}
		normType, ok := domain.NormalizeCandidateEventType(ev.EventType)
		if !ok {
			return nil, domain.NewValidationError(fmt.Sprintf("events[%d].event_type", i), fmt.Sprintf("unsupported event_type: %s", ev.EventType))
		}
		in.Events[i].EventType = string(normType)
		if ev.ClientInstanceID == "" {
			return nil, domain.NewValidationError(fmt.Sprintf("events[%d].client_instance_id", i), "client_instance_id is required")
		}
		if ev.ClientTimestamp.IsZero() {
			return nil, domain.NewValidationError(fmt.Sprintf("events[%d].client_timestamp", i), "client_timestamp is required")
		}
		if ev.SequenceNumber <= 0 {
			return nil, domain.NewValidationError(fmt.Sprintf("events[%d].sequence_number", i), "sequence_number must be > 0")
		}
		if ev.SchemaVersion != "" && ev.SchemaVersion != "1.0" {
			return nil, domain.NewValidationError(fmt.Sprintf("events[%d].schema_version", i), fmt.Sprintf("unsupported schema_version: %s", ev.SchemaVersion))
		}
	}

	// 3. PostgreSQL Transaction with Row-Level Lock
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txRepos := repopg.RepositoriesTx(tx)

	// Acquire row lock on exam_attempts
	lockedAttempt, err := txRepos.Attempts.FindByIDForUpdate(ctx, in.AttemptID)
	if err != nil {
		return nil, err
	}

	// Check attempt status & expiry
	if lockedAttempt.Status == domain.AttemptStatusSubmitted || lockedAttempt.Status == domain.AttemptStatusFinalized {
		return nil, fmt.Errorf("%w: attempt is %s", domain.ErrAttemptNotActive, lockedAttempt.Status)
	}
	if lockedAttempt.Status != domain.AttemptStatusActive {
		return nil, fmt.Errorf("%w: attempt is not active (%s)", domain.ErrAttemptNotActive, lockedAttempt.Status)
	}
	if domain.IsExpired(lockedAttempt.ExpiresAt) {
		// Auto-expire
		lockedAttempt.Status = domain.AttemptStatusExpired
		_ = txRepos.Attempts.Update(ctx, lockedAttempt)
		return nil, domain.ErrAttemptExpired
	}

	// Fetch current committed sequence and latest event for hash chain
	currentMaxSeq, err := txRepos.CandidateEvents.GetMaxSequence(ctx, in.AttemptID)
	if err != nil {
		return nil, err
	}

	latestEvent, err := txRepos.CandidateEvents.FindLatestByAttempt(ctx, in.AttemptID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			latestEvent = nil
		} else {
			return nil, err
		}
	}

	acceptedCount := 0
	serverNow := time.Now().UTC()

	// 4. Process each event sequentially in the batch
	for _, ev := range in.Events {
		// Validate session epoch
		if ev.SessionEpoch < lockedAttempt.SessionEpoch {
			return nil, domain.ErrStaleEpoch
		}

		// Idempotency / Duplicate Check
		existingByEventID, err := txRepos.CandidateEvents.FindByEventID(ctx, ev.EventID)
		if err == nil && existingByEventID != nil {
			// Exact duplicate retry
			if existingByEventID.AttemptID != in.AttemptID || existingByEventID.SequenceNumber != ev.SequenceNumber {
				return nil, domain.ErrConflict
			}
			continue
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}

		// Check sequence monotonicity vs currentMaxSeq
		if ev.SequenceNumber <= currentMaxSeq {
			// Sequence already committed with different event_id -> Conflict
			return nil, &api.SequenceConflictError{
				ExpectedSequence:      currentMaxSeq + 1,
				LastCommittedSequence: currentMaxSeq,
			}
		}

		if ev.SequenceNumber > currentMaxSeq+1 {
			// Sequence gap
			return nil, &api.SequenceConflictError{
				ExpectedSequence:      currentMaxSeq + 1,
				LastCommittedSequence: currentMaxSeq,
			}
		}

		// Valid next sequence: ev.SequenceNumber == currentMaxSeq + 1
		var prevHash string
		if latestEvent == nil {
			// Initial event hash chain anchor: H_0 = SHA256(AttemptID + StartTimestamp)
			var startStr string
			if lockedAttempt.StartedAt != nil {
				startStr = fmt.Sprintf("%d", lockedAttempt.StartedAt.UnixMilli())
			}
			initBytes := sha256.Sum256([]byte(lockedAttempt.ID.String() + startStr))
			prevHash = hex.EncodeToString(initBytes[:])
		} else {
			if latestEvent.EventHash != nil {
				prevHash = *latestEvent.EventHash
			}
		}

		payloadBytes, err := json.Marshal(ev.Payload)
		if err != nil {
			payloadBytes = []byte("{}")
		}

		// H_n = SHA256(H_(n-1) + SeqID + EventType + PayloadJSON + ClientTimestamp)
		rawChain := fmt.Sprintf("%s%d%s%s%s",
			prevHash,
			ev.SequenceNumber,
			ev.EventType,
			string(payloadBytes),
			ev.ClientTimestamp.UTC().Format(time.RFC3339Nano),
		)
		hashSum := sha256.Sum256([]byte(rawChain))
		eventHash := hex.EncodeToString(hashSum[:])

		canonicalEvent := &models.CandidateEvent{
			ID:               domain.NewID(),
			EventID:          ev.EventID,
			AttemptID:        in.AttemptID,
			SessionID:        ev.SessionID,
			SessionEpoch:     ev.SessionEpoch,
			SequenceNumber:   ev.SequenceNumber,
			EventType:        domain.CandidateEventType(ev.EventType),
			SchemaVersion:    "1.0",
			ClientInstanceID: ev.ClientInstanceID,
			Payload:          ev.Payload,
			ClientTimestamp:  ev.ClientTimestamp.UTC(),
			ServerReceivedAt: serverNow,
			PrevHash:         &prevHash,
			EventHash:        &eventHash,
			ProcessingStatus: domain.EventProcessingStatusCommitted,
		}

		if err := txRepos.CandidateEvents.Insert(ctx, canonicalEvent); err != nil {
			return nil, fmt.Errorf("failed to persist candidate event: %w", err)
		}

		// Project candidate response if applicable (e.g. option_selected, text_answered, clear_answer)
		if qvIDStr, ok := ev.Payload["question_version_id"].(string); ok && qvIDStr != "" {
			if qvID, parseErr := domain.ParseID(qvIDStr); parseErr == nil {
				switch canonicalEvent.EventType {
				case domain.CandidateEventTypeOptionSelected, domain.CandidateEventTypeTextAnswered, domain.CandidateEventTypeNumericEntered:
					val, hasVal := ev.Payload["value"]
					var curVal map[string]any
					if hasVal {
						if m, isMap := val.(map[string]any); isMap {
							curVal = m
						} else {
							curVal = map[string]any{"selected": val}
						}
					}
					_ = txRepos.Responses.Upsert(ctx, &models.CandidateResponse{
						AttemptID:         in.AttemptID,
						QuestionVersionID: qvID,
						CurrentValue:      curVal,
						Status:            domain.ResponseStatusAnswered,
					})
				case domain.CandidateEventTypeClearAnswer:
					_ = txRepos.Responses.Upsert(ctx, &models.CandidateResponse{
						AttemptID:         in.AttemptID,
						QuestionVersionID: qvID,
						CurrentValue:      nil,
						Status:            domain.ResponseStatusUnanswered,
					})
				}
			}
		}

		// Transactionally create OutboxEvent for JetStream relay
		outboxPayload := map[string]any{
			"specversion":     "1.0",
			"id":              canonicalEvent.EventID.String(),
			"source":          "sentinel.backend/ingest",
			"type":            string(canonicalEvent.EventType),
			"datacontenttype": "application/json",
			"time":            canonicalEvent.ServerReceivedAt.Format(time.RFC3339Nano),
			"data": map[string]any{
				"attempt_id":         canonicalEvent.AttemptID.String(),
				"session_id":         canonicalEvent.SessionID,
				"session_epoch":      canonicalEvent.SessionEpoch,
				"sequence_number":    canonicalEvent.SequenceNumber,
				"client_instance_id": canonicalEvent.ClientInstanceID,
				"client_timestamp":   canonicalEvent.ClientTimestamp.Format(time.RFC3339Nano),
				"payload":            canonicalEvent.Payload,
				"prev_hash":          canonicalEvent.PrevHash,
				"event_hash":         canonicalEvent.EventHash,
			},
		}

		outboxEvent := &models.OutboxEvent{
			ID:          domain.NewID(),
			EventType:   string(canonicalEvent.EventType),
			StreamName:  "SENTINEL_CANDIDATE_EVENTS",
			Subject:     string(canonicalEvent.EventType),
			Payload:     outboxPayload,
			Status:      domain.OutboxStatusPending,
			CreatedAt:   serverNow,
		}

		if err := txRepos.OutboxEvents.Create(ctx, outboxEvent); err != nil {
			return nil, fmt.Errorf("failed to persist outbox event: %w", err)
		}

		// Advance sequence & state
		currentMaxSeq = ev.SequenceNumber
		latestEvent = canonicalEvent
		acceptedCount++
	}

	// Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit event sync transaction: %w", err)
	}

	return &SyncEventsOutput{
		AcceptedCount:         acceptedCount,
		LastCommittedSequence: currentMaxSeq,
		ServerTimestamp:       serverNow,
	}, nil
}

