package scoring_test

import (
	"testing"
	"time"

	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
	"github.com/sentinelexam/backend/internal/assessment/scoring"
)

func makeQV(t domain.QuestionType, correct map[string]any, marks, negMarks float64) models.QuestionVersion {
	return models.QuestionVersion{
		Type:          t,
		CorrectAnswer: correct,
		Marks:         marks,
		NegativeMarks: negMarks,
	}
}

func TestSingleChoice(t *testing.T) {
	qv := makeQV(domain.QuestionTypeSingleChoice, map[string]any{"selected": "B"}, 4, 1)

	t.Run("correct", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"selected": "B"})
		assertEqual(t, 4.0, r.Earned)
		assertEqual(t, true, r.Correct)
	})

	t.Run("wrong answer gets negative marks", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"selected": "A"})
		assertEqual(t, -1.0, r.Earned)
		assertEqual(t, false, r.Correct)
	})

	t.Run("unanswered gets zero", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, nil)
		assertEqual(t, 0.0, r.Earned)
	})
}

func TestMultipleChoice(t *testing.T) {
	qv := makeQV(domain.QuestionTypeMultipleChoice,
		map[string]any{"selected": []interface{}{"A", "C"}}, 4, 1)

	t.Run("all correct", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"selected": []interface{}{"A", "C"}})
		assertEqual(t, 4.0, r.Earned)
		assertEqual(t, true, r.Correct)
	})

	t.Run("partial incorrect", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"selected": []interface{}{"A"}})
		assertEqual(t, -1.0, r.Earned)
		assertEqual(t, false, r.Correct)
	})
}

func TestTrueFalse(t *testing.T) {
	qv := makeQV(domain.QuestionTypeTrueFalse, map[string]any{"value": true}, 2, 0.5)

	t.Run("correct", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"value": true})
		assertEqual(t, 2.0, r.Earned)
	})

	t.Run("wrong", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"value": false})
		assertEqual(t, -0.5, r.Earned)
	})
}

func TestText(t *testing.T) {
	qvCaseSensitive := makeQV(domain.QuestionTypeText,
		map[string]any{"text": "Paris", "case_sensitive": true}, 3, 0)
	qvCI := makeQV(domain.QuestionTypeText,
		map[string]any{"text": "Paris", "case_sensitive": false}, 3, 0)

	t.Run("exact match", func(t *testing.T) {
		r := scoring.ScoreResponse(qvCaseSensitive, map[string]any{"text": "Paris"})
		assertEqual(t, 3.0, r.Earned)
	})

	t.Run("wrong case with case_sensitive=true fails", func(t *testing.T) {
		r := scoring.ScoreResponse(qvCaseSensitive, map[string]any{"text": "paris"})
		assertEqual(t, 0.0, r.Earned)
	})

	t.Run("case insensitive match", func(t *testing.T) {
		r := scoring.ScoreResponse(qvCI, map[string]any{"text": "  paris  "})
		assertEqual(t, 3.0, r.Earned)
	})
}

func TestNumeric(t *testing.T) {
	qv := makeQV(domain.QuestionTypeNumeric,
		map[string]any{"value": float64(42), "tolerance": float64(0.5)}, 5, 0)

	t.Run("exact", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"value": float64(42)})
		assertEqual(t, 5.0, r.Earned)
	})

	t.Run("within tolerance", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"value": float64(42.4)})
		assertEqual(t, 5.0, r.Earned)
	})

	t.Run("outside tolerance", func(t *testing.T) {
		r := scoring.ScoreResponse(qv, map[string]any{"value": float64(43)})
		assertEqual(t, 0.0, r.Earned)
	})
}

func TestScoreAttempt(t *testing.T) {
	q1ID := domain.NewID()
	q2ID := domain.NewID()

	questions := []models.ExamQuestionDetail{
		{
			ExamQuestion: models.ExamQuestion{QuestionVersionID: q1ID},
			QuestionVersion: makeQV(domain.QuestionTypeSingleChoice,
				map[string]any{"selected": "A"}, 4, 1),
		},
		{
			ExamQuestion: models.ExamQuestion{QuestionVersionID: q2ID},
			QuestionVersion: makeQV(domain.QuestionTypeNumeric,
				map[string]any{"value": float64(10), "tolerance": float64(0)}, 2, 0),
		},
	}
	responses := []models.CandidateResponse{
		{QuestionVersionID: q1ID, CurrentValue: map[string]any{"selected": "A"}},
		{QuestionVersionID: q2ID, CurrentValue: map[string]any{"value": float64(10)}},
	}
	score, max := scoring.ScoreAttempt(questions, responses)
	assertEqual(t, 6.0, score)
	assertEqual(t, 6.0, max)
}

func TestStateMachineExam(t *testing.T) {
	cases := []struct {
		from, to domain.ExamStatus
		wantErr  bool
	}{
		{domain.ExamStatusDraft, domain.ExamStatusScheduled, false},
		{domain.ExamStatusScheduled, domain.ExamStatusLive, false},
		{domain.ExamStatusLive, domain.ExamStatusPaused, false},
		{domain.ExamStatusPaused, domain.ExamStatusLive, false},
		{domain.ExamStatusLive, domain.ExamStatusCompleted, false},
		{domain.ExamStatusCompleted, domain.ExamStatusArchived, false},
		// Invalid transitions
		{domain.ExamStatusDraft, domain.ExamStatusLive, true},
		{domain.ExamStatusArchived, domain.ExamStatusDraft, true},
	}
	for _, tc := range cases {
		err := domain.ValidExamTransition(tc.from, tc.to)
		if tc.wantErr && err == nil {
			t.Errorf("expected error for %s→%s, got nil", tc.from, tc.to)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("unexpected error for %s→%s: %v", tc.from, tc.to, err)
		}
	}
}

func TestStateMachineAttempt(t *testing.T) {
	cases := []struct {
		from, to domain.AttemptStatus
		wantErr  bool
	}{
		{domain.AttemptStatusNotStarted, domain.AttemptStatusActive, false},
		{domain.AttemptStatusActive, domain.AttemptStatusSubmitted, false},
		{domain.AttemptStatusActive, domain.AttemptStatusExpired, false},
		{domain.AttemptStatusSubmitted, domain.AttemptStatusFinalized, false},
		{domain.AttemptStatusExpired, domain.AttemptStatusFinalized, false},
		// Invalid
		{domain.AttemptStatusFinalized, domain.AttemptStatusActive, true},
		{domain.AttemptStatusSubmitted, domain.AttemptStatusActive, true},
	}
	for _, tc := range cases {
		err := domain.ValidAttemptTransition(tc.from, tc.to)
		if tc.wantErr && err == nil {
			t.Errorf("expected error for %s→%s, got nil", tc.from, tc.to)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("unexpected error for %s→%s: %v", tc.from, tc.to, err)
		}
	}
}

func TestTimerHelpers(t *testing.T) {
	past := time.Now().Add(-1 * time.Second)
	future := time.Now().Add(1 * time.Hour)

	if !domain.IsExpired(&past) {
		t.Error("expected past time to be expired")
	}
	if domain.IsExpired(&future) {
		t.Error("expected future time to not be expired")
	}
	if domain.IsExpired(nil) {
		t.Error("nil expires_at should not be expired")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertEqual[T comparable](t *testing.T, expected, got T) {
	t.Helper()
	if expected != got {
		t.Errorf("expected %v, got %v", expected, got)
	}
}
