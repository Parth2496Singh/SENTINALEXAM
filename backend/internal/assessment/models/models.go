// Package models defines the data models (structs) that map to database rows.
// These are NOT domain aggregates with behaviour; they are persistence representations.
package models

import (
	"time"

	"github.com/sentinelexam/backend/internal/assessment/domain"
)

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

type User struct {
	ID          domain.ID
	KeycloakSub domain.ID
	Email       string
	DisplayName string
	Roles       []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ---------------------------------------------------------------------------
// Candidate
// ---------------------------------------------------------------------------

type Candidate struct {
	ID        domain.ID
	UserID    domain.ID
	Status    domain.CandidateStatus
	FullName  string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ---------------------------------------------------------------------------
// QuestionBank
// ---------------------------------------------------------------------------

type QuestionBank struct {
	ID          domain.ID
	Name        string
	Description string
	Status      domain.QuestionBankStatus
	CreatedBy   *domain.ID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ---------------------------------------------------------------------------
// Question
// ---------------------------------------------------------------------------

type Question struct {
	ID               domain.ID
	QuestionBankID   domain.ID
	Status           domain.QuestionStatus
	CurrentVersionID *domain.ID
	CreatedBy        *domain.ID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ---------------------------------------------------------------------------
// QuestionVersion
// ---------------------------------------------------------------------------

type QuestionVersion struct {
	ID            domain.ID
	QuestionID    domain.ID
	Version       int
	Type          domain.QuestionType
	Content       map[string]any // stimulus, options, etc.
	CorrectAnswer map[string]any // scoring key
	Marks         float64
	NegativeMarks float64
	Status        domain.QuestionStatus
	CreatedBy     *domain.ID
	CreatedAt     time.Time
}

// ---------------------------------------------------------------------------
// Exam
// ---------------------------------------------------------------------------

type Exam struct {
	ID              domain.ID
	Title           string
	Description     string
	Status          domain.ExamStatus
	DurationSeconds int
	ScheduledStart  *time.Time
	ScheduledEnd    *time.Time
	MaxAttempts     int
	Navigation      map[string]any
	Scoring         map[string]any
	Version         int
	CreatedBy       *domain.ID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ---------------------------------------------------------------------------
// ExamQuestion
// ---------------------------------------------------------------------------

type ExamQuestion struct {
	ID                domain.ID
	ExamID            domain.ID
	QuestionVersionID domain.ID
	Position          int
	CreatedAt         time.Time
}

// ExamQuestionDetail enriches ExamQuestion with version info.
type ExamQuestionDetail struct {
	ExamQuestion
	QuestionVersion QuestionVersion
}

// ---------------------------------------------------------------------------
// ExamAssignment
// ---------------------------------------------------------------------------

type ExamAssignment struct {
	ID          domain.ID
	ExamID      domain.ID
	CandidateID domain.ID
	Status      domain.AssignmentStatus
	AssignedBy  *domain.ID
	AssignedAt  time.Time
	RevokedAt   *time.Time
}

// ---------------------------------------------------------------------------
// ExamAttempt
// ---------------------------------------------------------------------------

type ExamAttempt struct {
	ID                domain.ID
	ExamID            domain.ID
	CandidateID       domain.ID
	AssignmentID      domain.ID
	Status            domain.AttemptStatus
	StartedAt         *time.Time
	ExpiresAt         *time.Time
	SubmittedAt       *time.Time
	FinalizedAt       *time.Time
	SessionEpoch      int
	CurrentQuestionID *domain.ID
	Version           int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ---------------------------------------------------------------------------
// AssessmentSession
// ---------------------------------------------------------------------------

type AssessmentSession struct {
	ID                 domain.ID
	AttemptID          domain.ID
	ClientInstanceID   string
	SessionEpoch       int
	Status             domain.SessionStatus
	ConnectedAt        *time.Time
	DisconnectedAt     *time.Time
	TerminatedAt       *time.Time
	LastHeartbeatAt    *time.Time
	LastClientSequence int64
	LastServerSequence int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ---------------------------------------------------------------------------
// CandidateResponse
// ---------------------------------------------------------------------------

type CandidateResponse struct {
	ID                domain.ID
	AttemptID         domain.ID
	QuestionVersionID domain.ID
	CurrentValue      map[string]any
	Status            domain.ResponseStatus
	Version           int
	UpdatedAt         time.Time
}

// ---------------------------------------------------------------------------
// Submission
// ---------------------------------------------------------------------------

type Submission struct {
	ID          domain.ID
	AttemptID   domain.ID
	Status      domain.SubmissionStatus
	RequestedAt time.Time
	SubmittedAt *time.Time
	FinalizedAt *time.Time
	Score       *float64
	MaxScore    *float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ---------------------------------------------------------------------------
// Runtime bootstrap payload
// ---------------------------------------------------------------------------

type AttemptRuntime struct {
	Attempt   ExamAttempt
	Exam      Exam
	Questions []ExamQuestionDetail
	Responses []CandidateResponse
	Session   *AssessmentSession
}

// ---------------------------------------------------------------------------
// CandidateEvent (Sequenced append-only WAL entity)
// ---------------------------------------------------------------------------

type CandidateEvent struct {
	ID               domain.ID
	EventID          domain.ID
	AttemptID        domain.ID
	SessionID        *domain.ID
	SessionEpoch     int
	SequenceNumber   int64
	EventType        domain.CandidateEventType
	SchemaVersion    string
	ClientInstanceID string
	Payload          map[string]any
	ClientTimestamp  time.Time
	ServerReceivedAt time.Time
	PrevHash         *string
	EventHash        *string
	ProcessingStatus domain.EventProcessingStatus
}

// ---------------------------------------------------------------------------
// OutboxEvent (Transactional outbox for JetStream relay)
// ---------------------------------------------------------------------------

type OutboxEvent struct {
	ID          domain.ID
	EventType   string
	StreamName  string
	Subject     string
	Payload     map[string]any
	Status      domain.OutboxStatus
	CreatedAt   time.Time
	PublishedAt *time.Time
}

