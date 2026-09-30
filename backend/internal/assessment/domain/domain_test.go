package domain_test

import (
	"testing"

	"github.com/sentinelexam/backend/internal/assessment/domain"
)

func TestCandidateEventTypeValidation(t *testing.T) {
	validTypes := []domain.CandidateEventType{
		domain.CandidateEventTypeOptionSelected,
		domain.CandidateEventTypeTextAnswered,
		domain.CandidateEventTypeClearAnswer,
		domain.CandidateEventTypeQuestionViewed,
		domain.CandidateEventTypeSectionChanged,
		domain.CandidateEventTypeHeartbeat,
		domain.CandidateEventTypeFocusLost,
		domain.CandidateEventTypeFocusRegained,
		domain.CandidateEventTypeSyncFlush,
	}

	for _, vt := range validTypes {
		if !domain.IsValidCandidateEventType(vt) {
			t.Errorf("expected valid candidate event type for %s", vt)
		}
	}

	invalidTypes := []domain.CandidateEventType{
		"invalid.type",
		"sentinel.candidate.hack",
		"",
	}

	for _, it := range invalidTypes {
		if domain.IsValidCandidateEventType(it) {
			t.Errorf("expected invalid candidate event type for %s", it)
		}
	}
}

func TestValidateEventSequence(t *testing.T) {
	tests := []struct {
		name             string
		incomingSeq      int64
		lastCommittedSeq int64
		wantErr          bool
	}{
		{
			name:             "Next monotonic sequence from 0",
			incomingSeq:      1,
			lastCommittedSeq: 0,
			wantErr:          false,
		},
		{
			name:             "Next monotonic sequence from 42",
			incomingSeq:      43,
			lastCommittedSeq: 42,
			wantErr:          false,
		},
		{
			name:             "Duplicate or prior sequence (idempotent replay)",
			incomingSeq:      42,
			lastCommittedSeq: 42,
			wantErr:          false,
		},
		{
			name:             "Prior sequence (idempotent overlap)",
			incomingSeq:      30,
			lastCommittedSeq: 42,
			wantErr:          false,
		},
		{
			name:             "Sequence gap",
			incomingSeq:      45,
			lastCommittedSeq: 42,
			wantErr:          true,
		},
		{
			name:             "Zero sequence invalid",
			incomingSeq:      0,
			lastCommittedSeq: 0,
			wantErr:          true,
		},
		{
			name:             "Negative sequence invalid",
			incomingSeq:      -1,
			lastCommittedSeq: 0,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateEventSequence(tt.incomingSeq, tt.lastCommittedSeq)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateEventSequence(%d, %d) error = %v, wantErr = %v",
					tt.incomingSeq, tt.lastCommittedSeq, err, tt.wantErr)
			}
		})
	}
}
