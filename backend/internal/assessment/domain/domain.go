// Package domain defines the core domain types, constants, and state
// machine transition rules for SentinelExam Assessment Core.
// This package is infrastructure-free: no database or HTTP dependencies.
package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Sentinel domain errors
// ---------------------------------------------------------------------------

var (
	ErrInvalidTransition    = errors.New("invalid state transition")
	ErrAttemptExpired       = errors.New("attempt has expired")
	ErrAttemptNotActive     = errors.New("attempt is not active")
	ErrResponseFrozen       = errors.New("response is frozen; attempt is finalized")
	ErrNotFound             = errors.New("not found")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrConflict             = errors.New("conflict")
	ErrValidation           = errors.New("validation error")
	ErrAlreadyExists        = errors.New("already exists")
	ErrQuestionVersionImmutable = errors.New("published question version is immutable")
	ErrExamPublishValidation    = errors.New("exam publish validation failed")
	ErrDuplicateSubmission      = errors.New("duplicate submission")
	ErrAssignmentRevoked        = errors.New("assignment has been revoked")
	ErrDuplicateEvent           = errors.New("duplicate event")
	ErrSequenceConflict         = errors.New("sequence conflict")
	ErrStaleEpoch               = errors.New("stale session epoch")
	ErrSequenceGap              = errors.New("sequence gap")
)

// ValidationError carries field-level details.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %v", e.Fields)
}

func NewValidationError(field, msg string) *ValidationError {
	return &ValidationError{Fields: map[string]string{field: msg}}
}

// ---------------------------------------------------------------------------
// Exam
// ---------------------------------------------------------------------------

type ExamStatus string

const (
	ExamStatusDraft     ExamStatus = "DRAFT"
	ExamStatusScheduled ExamStatus = "SCHEDULED"
	ExamStatusLive      ExamStatus = "LIVE"
	ExamStatusPaused    ExamStatus = "PAUSED"
	ExamStatusCompleted ExamStatus = "COMPLETED"
	ExamStatusArchived  ExamStatus = "ARCHIVED"
)

func ValidExamTransition(from, to ExamStatus) error {
	allowed := map[ExamStatus][]ExamStatus{
		ExamStatusDraft:     {ExamStatusScheduled},
		ExamStatusScheduled: {ExamStatusLive},
		ExamStatusLive:      {ExamStatusPaused, ExamStatusCompleted},
		ExamStatusPaused:    {ExamStatusLive, ExamStatusCompleted},
		ExamStatusCompleted: {ExamStatusArchived},
		ExamStatusArchived:  {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("%w: exam %s → %s", ErrInvalidTransition, from, to)
}

// ---------------------------------------------------------------------------
// Question / QuestionVersion
// ---------------------------------------------------------------------------

type QuestionStatus string

const (
	QuestionStatusDraft      QuestionStatus = "DRAFT"
	QuestionStatusActive     QuestionStatus = "ACTIVE"
	QuestionStatusSuperseded QuestionStatus = "SUPERSEDED"
	QuestionStatusRetired    QuestionStatus = "RETIRED"
)

func ValidQuestionTransition(from, to QuestionStatus) error {
	allowed := map[QuestionStatus][]QuestionStatus{
		QuestionStatusDraft:      {QuestionStatusActive},
		QuestionStatusActive:     {QuestionStatusSuperseded, QuestionStatusRetired},
		QuestionStatusSuperseded: {QuestionStatusRetired},
		QuestionStatusRetired:    {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("%w: question %s → %s", ErrInvalidTransition, from, to)
}

func IsQuestionVersionImmutable(status QuestionStatus) bool {
	return status == QuestionStatusActive ||
		status == QuestionStatusSuperseded ||
		status == QuestionStatusRetired
}

type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "SINGLE_CHOICE"
	QuestionTypeMultipleChoice QuestionType = "MULTIPLE_CHOICE"
	QuestionTypeTrueFalse      QuestionType = "TRUE_FALSE"
	QuestionTypeText           QuestionType = "TEXT"
	QuestionTypeNumeric        QuestionType = "NUMERIC"
)

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

type AssignmentStatus string

const (
	AssignmentStatusAssigned AssignmentStatus = "ASSIGNED"
	AssignmentStatusRevoked  AssignmentStatus = "REVOKED"
)

// ---------------------------------------------------------------------------
// Attempt
// ---------------------------------------------------------------------------

type AttemptStatus string

const (
	AttemptStatusNotStarted AttemptStatus = "NOT_STARTED"
	AttemptStatusActive     AttemptStatus = "ACTIVE"
	AttemptStatusSubmitted  AttemptStatus = "SUBMITTED"
	AttemptStatusExpired    AttemptStatus = "EXPIRED"
	AttemptStatusTerminated AttemptStatus = "TERMINATED"
	AttemptStatusFinalized  AttemptStatus = "FINALIZED"
)

func ValidAttemptTransition(from, to AttemptStatus) error {
	allowed := map[AttemptStatus][]AttemptStatus{
		AttemptStatusNotStarted: {AttemptStatusActive},
		AttemptStatusActive:     {AttemptStatusSubmitted, AttemptStatusExpired, AttemptStatusTerminated},
		AttemptStatusSubmitted:  {AttemptStatusFinalized},
		AttemptStatusExpired:    {AttemptStatusFinalized},
		AttemptStatusTerminated: {AttemptStatusFinalized},
		AttemptStatusFinalized:  {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("%w: attempt %s → %s", ErrInvalidTransition, from, to)
}

func IsAttemptTerminal(s AttemptStatus) bool {
	return s == AttemptStatusSubmitted || s == AttemptStatusExpired ||
		s == AttemptStatusTerminated || s == AttemptStatusFinalized
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

type SessionStatus string

const (
	SessionStatusCreated      SessionStatus = "CREATED"
	SessionStatusConnected    SessionStatus = "CONNECTED"
	SessionStatusDisconnected SessionStatus = "DISCONNECTED"
	SessionStatusReconnecting SessionStatus = "RECONNECTING"
	SessionStatusTerminated   SessionStatus = "TERMINATED"
)

func ValidSessionTransition(from, to SessionStatus) error {
	allowed := map[SessionStatus][]SessionStatus{
		SessionStatusCreated:      {SessionStatusConnected},
		SessionStatusConnected:    {SessionStatusDisconnected, SessionStatusTerminated},
		SessionStatusDisconnected: {SessionStatusReconnecting, SessionStatusTerminated},
		SessionStatusReconnecting: {SessionStatusConnected, SessionStatusTerminated},
		SessionStatusTerminated:   {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("%w: session %s → %s", ErrInvalidTransition, from, to)
}

// ---------------------------------------------------------------------------
// Response
// ---------------------------------------------------------------------------

type ResponseStatus string

const (
	ResponseStatusUnanswered ResponseStatus = "UNANSWERED"
	ResponseStatusAnswered   ResponseStatus = "ANSWERED"
	ResponseStatusSubmitted  ResponseStatus = "SUBMITTED"
)

func ValidResponseTransition(from, to ResponseStatus) error {
	// UNANSWERED ↔ ANSWERED and ANSWERED → SUBMITTED
	allowed := map[ResponseStatus][]ResponseStatus{
		ResponseStatusUnanswered: {ResponseStatusAnswered},
		ResponseStatusAnswered:   {ResponseStatusUnanswered, ResponseStatusSubmitted},
		ResponseStatusSubmitted:  {},
	}
	for _, s := range allowed[from] {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("%w: response %s → %s", ErrInvalidTransition, from, to)
}

// ---------------------------------------------------------------------------
// Other statuses
// ---------------------------------------------------------------------------

type CandidateStatus string

const (
	CandidateStatusActive    CandidateStatus = "ACTIVE"
	CandidateStatusSuspended CandidateStatus = "SUSPENDED"
)

type QuestionBankStatus string

const (
	QuestionBankStatusActive   QuestionBankStatus = "ACTIVE"
	QuestionBankStatusArchived QuestionBankStatus = "ARCHIVED"
)

type SubmissionStatus string

const (
	SubmissionStatusPending   SubmissionStatus = "PENDING"
	SubmissionStatusSubmitted SubmissionStatus = "SUBMITTED"
	SubmissionStatusFinalized SubmissionStatus = "FINALIZED"
)

type EventProcessingStatus string

const (
	EventProcessingStatusCommitted EventProcessingStatus = "COMMITTED"
	EventProcessingStatusRejected  EventProcessingStatus = "REJECTED"
	EventProcessingStatusIgnored   EventProcessingStatus = "IGNORED"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusPublished OutboxStatus = "PUBLISHED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

type CandidateEventType string

const (
	CandidateEventTypeOptionSelected  CandidateEventType = "sentinel.candidate.option_selected"
	CandidateEventTypeTextAnswered    CandidateEventType = "sentinel.candidate.text_answered"
	CandidateEventTypeNumericEntered  CandidateEventType = "sentinel.candidate.numeric_entered"
	CandidateEventTypeClearAnswer     CandidateEventType = "sentinel.candidate.clear_answer"
	CandidateEventTypeQuestionViewed  CandidateEventType = "sentinel.candidate.question_viewed"
	CandidateEventTypeSectionChanged  CandidateEventType = "sentinel.candidate.section_changed"
	CandidateEventTypeReviewFlagged   CandidateEventType = "sentinel.candidate.review_flagged"
	CandidateEventTypeQuestionNavigated CandidateEventType = "sentinel.candidate.question_navigated"
	CandidateEventTypeHeartbeat       CandidateEventType = "sentinel.candidate.heartbeat"
	CandidateEventTypeFocusLost       CandidateEventType = "sentinel.candidate.focus_lost"
	CandidateEventTypeFocusRegained   CandidateEventType = "sentinel.candidate.focus_regained"
	CandidateEventTypeSyncFlush       CandidateEventType = "sentinel.candidate.sync_flush"
)

// ---------------------------------------------------------------------------
// Domain entities (lightweight value objects used across layers)
// ---------------------------------------------------------------------------

type ID = uuid.UUID

func NewID() ID { return uuid.New() }
func ParseID(s string) (ID, error) { return uuid.Parse(s) }

func MustParseID(s string) ID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic("invalid uuid: " + s)
	}
	return id
}

type Pagination struct {
	Page    int
	PerPage int
}

func (p Pagination) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func (p Pagination) Limit() int {
	if p.PerPage < 1 || p.PerPage > 200 {
		return 50
	}
	return p.PerPage
}

// ---------------------------------------------------------------------------
// Timer helpers
// ---------------------------------------------------------------------------

// IsExpired returns true if the server-authoritative expires_at has passed.
func IsExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*expiresAt)
}

// ComputeExpiresAt returns the server-authoritative deadline for an attempt.
func ComputeExpiresAt(startedAt time.Time, durationSeconds int) time.Time {
	return startedAt.Add(time.Duration(durationSeconds) * time.Second)
}

// ---------------------------------------------------------------------------
// Candidate Event Validation Helpers
// ---------------------------------------------------------------------------

func NormalizeCandidateEventType(t string) (CandidateEventType, bool) {
	switch t {
	case "sentinel.candidate.option_selected", "candidate.option_selected":
		return CandidateEventTypeOptionSelected, true
	case "sentinel.candidate.text_answered", "candidate.text_answered", "sentinel.candidate.text_entered", "candidate.text_entered":
		return CandidateEventTypeTextAnswered, true
	case "sentinel.candidate.numeric_entered", "candidate.numeric_entered":
		return CandidateEventTypeNumericEntered, true
	case "sentinel.candidate.clear_answer", "candidate.clear_answer", "sentinel.candidate.answer_cleared", "candidate.answer_cleared":
		return CandidateEventTypeClearAnswer, true
	case "sentinel.candidate.question_viewed", "candidate.question_viewed":
		return CandidateEventTypeQuestionViewed, true
	case "sentinel.candidate.section_changed", "candidate.section_changed":
		return CandidateEventTypeSectionChanged, true
	case "sentinel.candidate.review_flagged", "candidate.review_flagged":
		return CandidateEventTypeReviewFlagged, true
	case "sentinel.candidate.question_navigated", "candidate.question_navigated":
		return CandidateEventTypeQuestionNavigated, true
	case "sentinel.candidate.heartbeat", "candidate.heartbeat":
		return CandidateEventTypeHeartbeat, true
	case "sentinel.candidate.focus_lost", "candidate.focus_lost":
		return CandidateEventTypeFocusLost, true
	case "sentinel.candidate.focus_regained", "candidate.focus_regained":
		return CandidateEventTypeFocusRegained, true
	case "sentinel.candidate.sync_flush", "candidate.sync_flush":
		return CandidateEventTypeSyncFlush, true
	default:
		return "", false
	}
}

func IsValidCandidateEventType(t CandidateEventType) bool {
	_, ok := NormalizeCandidateEventType(string(t))
	return ok
}

func ValidateEventSequence(incomingSeq, lastCommittedSeq int64) error {
	if incomingSeq <= 0 {
		return fmt.Errorf("%w: sequence number must be positive (got %d)", ErrValidation, incomingSeq)
	}
	if incomingSeq <= lastCommittedSeq {
		// Existing or duplicate sequence range
		return nil
	}
	if incomingSeq == lastCommittedSeq+1 {
		// Strictly next monotonic sequence
		return nil
	}
	// Sequence gap
	return fmt.Errorf("%w: expected %d, got %d", ErrSequenceGap, lastCommittedSeq+1, incomingSeq)
}

