// Package api contains shared HTTP utilities for the assessment API.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sentinelexam/backend/internal/assessment/domain"
)

// ---------------------------------------------------------------------------
// Standard API error envelope
// ---------------------------------------------------------------------------

type ErrorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: "validation failed", Fields: fields},
	})
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type SequenceConflictError struct {
	ExpectedSequence      int64 `json:"expected_sequence"`
	LastCommittedSequence int64 `json:"last_committed_sequence"`
}

func (e *SequenceConflictError) Error() string {
	return fmt.Sprintf("sequence conflict: expected %d, last committed %d", e.ExpectedSequence, e.LastCommittedSequence)
}

func WriteSequenceConflictError(w http.ResponseWriter, expectedSeq, lastCommittedSeq int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "SEQUENCE_CONFLICT",
			"message": "Sequence gap or conflict detected",
			"details": map[string]any{
				"expected_sequence":       expectedSeq,
				"last_committed_sequence": lastCommittedSeq,
			},
		},
	})
}

// MapError translates domain errors to appropriate HTTP responses.
func MapError(w http.ResponseWriter, err error) {
	var ve *domain.ValidationError
	var sce *SequenceConflictError
	switch {
	case errors.As(err, &ve):
		WriteValidationError(w, ve.Fields)
	case errors.As(err, &sce):
		WriteSequenceConflictError(w, sce.ExpectedSequence, sce.LastCommittedSequence)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, domain.ErrUnauthorized):
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, domain.ErrStaleEpoch):
		WriteError(w, http.StatusConflict, "STALE_SESSION_EPOCH", "stale session epoch")
	case errors.Is(err, domain.ErrSequenceConflict):
		WriteError(w, http.StatusConflict, "SEQUENCE_CONFLICT", "sequence conflict")
	case errors.Is(err, domain.ErrSequenceGap):
		WriteError(w, http.StatusConflict, "SEQUENCE_CONFLICT", err.Error())
	case errors.Is(err, domain.ErrInvalidTransition):
		WriteError(w, http.StatusConflict, "INVALID_STATE_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrConflict):
		WriteError(w, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, domain.ErrAttemptExpired):
		WriteError(w, http.StatusGone, "ATTEMPT_EXPIRED", "attempt has expired")
	case errors.Is(err, domain.ErrAttemptNotActive):
		WriteError(w, http.StatusConflict, "ATTEMPT_NOT_ACTIVE", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		WriteError(w, http.StatusConflict, "ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrAssignmentRevoked):
		WriteError(w, http.StatusForbidden, "ASSIGNMENT_REVOKED", "assignment has been revoked")
	case errors.Is(err, domain.ErrExamPublishValidation):
		WriteError(w, http.StatusUnprocessableEntity, "PUBLISH_VALIDATION_FAILED", err.Error())
	case errors.Is(err, domain.ErrQuestionVersionImmutable):
		WriteError(w, http.StatusConflict, "QUESTION_VERSION_IMMUTABLE", "cannot modify a published question version")
	case errors.Is(err, domain.ErrValidation):
		WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
	}
}

// ---------------------------------------------------------------------------
// Pagination helpers
// ---------------------------------------------------------------------------

type PaginatedResponse struct {
	Data  any        `json:"data"`
	Meta  PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func ParsePagination(r *http.Request) domain.Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 50
	}
	return domain.Pagination{Page: page, PerPage: perPage}
}

// ---------------------------------------------------------------------------
// URL path ID extraction helper
// ---------------------------------------------------------------------------

func ParseIDParam(r *http.Request, name string) (domain.ID, error) {
	val := r.PathValue(name)
	if val == "" {
		return domain.ID{}, fmt.Errorf("missing path parameter: %s", name)
	}
	id, err := domain.ParseID(val)
	if err != nil {
		return domain.ID{}, fmt.Errorf("invalid id: %s", val)
	}
	return id, nil
}

// ---------------------------------------------------------------------------
// Request body decoding
// ---------------------------------------------------------------------------

func DecodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("malformed request body: %w", err)
	}
	return nil
}
