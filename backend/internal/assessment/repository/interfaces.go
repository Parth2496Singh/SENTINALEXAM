// Package repository defines the data access interface and PostgreSQL
// implementation for all assessment core entities.
package repository

import (
	"context"

	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
)

// ---------------------------------------------------------------------------
// Repository interfaces — depend on abstractions, not pgx directly
// ---------------------------------------------------------------------------

type UserRepository interface {
	FindByKeycloakSub(ctx context.Context, sub domain.ID) (*models.User, error)
	Upsert(ctx context.Context, user *models.User) error
}

type CandidateRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.Candidate, error)
	FindByUserID(ctx context.Context, userID domain.ID) (*models.Candidate, error)
	List(ctx context.Context, pagination domain.Pagination) ([]models.Candidate, int, error)
	Create(ctx context.Context, c *models.Candidate) error
	Update(ctx context.Context, c *models.Candidate) error
}

type QuestionBankRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.QuestionBank, error)
	List(ctx context.Context, pagination domain.Pagination) ([]models.QuestionBank, int, error)
	Create(ctx context.Context, qb *models.QuestionBank) error
}

type QuestionRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.Question, error)
	ListByBank(ctx context.Context, bankID domain.ID, pagination domain.Pagination) ([]models.Question, int, error)
	Create(ctx context.Context, q *models.Question) error
	Update(ctx context.Context, q *models.Question) error
}

type QuestionVersionRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.QuestionVersion, error)
	FindLatestByQuestion(ctx context.Context, questionID domain.ID) (*models.QuestionVersion, error)
	ListByQuestion(ctx context.Context, questionID domain.ID) ([]models.QuestionVersion, error)
	Create(ctx context.Context, qv *models.QuestionVersion) error
	UpdateStatus(ctx context.Context, id domain.ID, status domain.QuestionStatus) error
}

type ExamRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.Exam, error)
	List(ctx context.Context, pagination domain.Pagination) ([]models.Exam, int, error)
	Create(ctx context.Context, e *models.Exam) error
	Update(ctx context.Context, e *models.Exam) error
	UpdateStatus(ctx context.Context, id domain.ID, status domain.ExamStatus, version int) error
}

type ExamQuestionRepository interface {
	FindByExam(ctx context.Context, examID domain.ID) ([]models.ExamQuestionDetail, error)
	Add(ctx context.Context, eq *models.ExamQuestion) error
	Remove(ctx context.Context, examID, questionVersionID domain.ID) error
	CountByExam(ctx context.Context, examID domain.ID) (int, error)
}

type AssignmentRepository interface {
	FindByExamAndCandidate(ctx context.Context, examID, candidateID domain.ID) (*models.ExamAssignment, error)
	FindByID(ctx context.Context, id domain.ID) (*models.ExamAssignment, error)
	ListByExam(ctx context.Context, examID domain.ID, pagination domain.Pagination) ([]models.ExamAssignment, int, error)
	ListByCandidate(ctx context.Context, candidateID domain.ID, pagination domain.Pagination) ([]models.ExamAssignment, int, error)
	Create(ctx context.Context, a *models.ExamAssignment) error
	UpdateStatus(ctx context.Context, id domain.ID, status domain.AssignmentStatus, revokedAt interface{}) error
}

type AttemptRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.ExamAttempt, error)
	FindByIDForUpdate(ctx context.Context, id domain.ID) (*models.ExamAttempt, error)
	FindByExamAndCandidate(ctx context.Context, examID, candidateID domain.ID) ([]models.ExamAttempt, error)
	CountByAssignment(ctx context.Context, assignmentID domain.ID) (int, error)
	List(ctx context.Context, pagination domain.Pagination) ([]models.ExamAttempt, int, error)
	Create(ctx context.Context, a *models.ExamAttempt) error
	Update(ctx context.Context, a *models.ExamAttempt) error
}

type SessionRepository interface {
	FindByID(ctx context.Context, id domain.ID) (*models.AssessmentSession, error)
	FindActiveByAttempt(ctx context.Context, attemptID domain.ID) (*models.AssessmentSession, error)
	Create(ctx context.Context, s *models.AssessmentSession) error
	Update(ctx context.Context, s *models.AssessmentSession) error
}

type ResponseRepository interface {
	FindByAttemptAndQuestion(ctx context.Context, attemptID, questionVersionID domain.ID) (*models.CandidateResponse, error)
	ListByAttempt(ctx context.Context, attemptID domain.ID) ([]models.CandidateResponse, error)
	Upsert(ctx context.Context, r *models.CandidateResponse) error
	UpdateStatus(ctx context.Context, attemptID domain.ID, status domain.ResponseStatus) error
}

type SubmissionRepository interface {
	FindByAttempt(ctx context.Context, attemptID domain.ID) (*models.Submission, error)
	Create(ctx context.Context, s *models.Submission) error
	Update(ctx context.Context, s *models.Submission) error
}

type CandidateEventRepository interface {
	Insert(ctx context.Context, event *models.CandidateEvent) error
	FindByEventID(ctx context.Context, eventID domain.ID) (*models.CandidateEvent, error)
	FindByAttemptAndSequence(ctx context.Context, attemptID domain.ID, seq int64) (*models.CandidateEvent, error)
	GetMaxSequence(ctx context.Context, attemptID domain.ID) (int64, error)
	FindLatestByAttempt(ctx context.Context, attemptID domain.ID) (*models.CandidateEvent, error)
	ListByAttemptRange(ctx context.Context, attemptID domain.ID, fromSeq, toSeq int64) ([]models.CandidateEvent, error)
	ListByAttempt(ctx context.Context, attemptID domain.ID) ([]models.CandidateEvent, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, event *models.OutboxEvent) error
	ListPending(ctx context.Context, limit int) ([]models.OutboxEvent, error)
	MarkPublished(ctx context.Context, id domain.ID) error
}

// Repositories bundles all repositories for injection.
type Repositories struct {
	Users             UserRepository
	Candidates        CandidateRepository
	QuestionBanks     QuestionBankRepository
	Questions         QuestionRepository
	QuestionVersions  QuestionVersionRepository
	Exams             ExamRepository
	ExamQuestions     ExamQuestionRepository
	Assignments       AssignmentRepository
	Attempts          AttemptRepository
	Sessions          SessionRepository
	Responses         ResponseRepository
	Submissions       SubmissionRepository
	CandidateEvents   CandidateEventRepository
	OutboxEvents      OutboxRepository
}
