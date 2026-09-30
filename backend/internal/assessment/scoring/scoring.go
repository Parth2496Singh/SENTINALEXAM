// Package scoring implements deterministic server-side scoring for all
// approved question types. The client NEVER sends authoritative scores.
package scoring

import (
	"fmt"
	"math"
	"strings"

	"github.com/sentinelexam/backend/internal/assessment/domain"
	"github.com/sentinelexam/backend/internal/assessment/models"
)

// Result is the scoring outcome for a single question.
type Result struct {
	Earned  float64
	Maximum float64
	Correct bool
}

// ScoreResponse scores a candidate response against the question version's
// stored correct_answer and scoring configuration.
func ScoreResponse(qv models.QuestionVersion, response map[string]any) Result {
	if response == nil {
		return Result{Maximum: qv.Marks}
	}
	switch qv.Type {
	case domain.QuestionTypeSingleChoice:
		return scoreSingleChoice(qv, response)
	case domain.QuestionTypeMultipleChoice:
		return scoreMultipleChoice(qv, response)
	case domain.QuestionTypeTrueFalse:
		return scoreTrueFalse(qv, response)
	case domain.QuestionTypeText:
		return scoreText(qv, response)
	case domain.QuestionTypeNumeric:
		return scoreNumeric(qv, response)
	}
	return Result{Maximum: qv.Marks}
}

// ScoreAttempt calculates total and max score across all question versions.
func ScoreAttempt(questions []models.ExamQuestionDetail, responses []models.CandidateResponse) (score, maxScore float64) {
	responseMap := make(map[domain.ID]map[string]any, len(responses))
	for _, r := range responses {
		responseMap[r.QuestionVersionID] = r.CurrentValue
	}
	for _, eq := range questions {
		r := ScoreResponse(eq.QuestionVersion, responseMap[eq.QuestionVersionID])
		score += r.Earned
		maxScore += r.Maximum
	}
	// Round to 4dp
	score = math.Round(score*10000) / 10000
	maxScore = math.Round(maxScore*10000) / 10000
	return
}

// ---------------------------------------------------------------------------
// Per-type scorers
// ---------------------------------------------------------------------------

func scoreSingleChoice(qv models.QuestionVersion, response map[string]any) Result {
	// correct_answer: {"selected": "A"}
	correct := stringVal(qv.CorrectAnswer, "selected")
	given := stringVal(response, "selected")
	if correct == "" || given == "" {
		return Result{Maximum: qv.Marks}
	}
	if correct == given {
		return Result{Earned: qv.Marks, Maximum: qv.Marks, Correct: true}
	}
	return Result{Earned: -qv.NegativeMarks, Maximum: qv.Marks}
}

func scoreMultipleChoice(qv models.QuestionVersion, response map[string]any) Result {
	// correct_answer: {"selected": ["A","C"]}
	// response:       {"selected": ["A","C"]}
	correctSet := stringSliceVal(qv.CorrectAnswer, "selected")
	givenSet := stringSliceVal(response, "selected")
	if len(correctSet) == 0 {
		return Result{Maximum: qv.Marks}
	}
	correct := toSet(correctSet)
	given := toSet(givenSet)
	if setsEqual(correct, given) {
		return Result{Earned: qv.Marks, Maximum: qv.Marks, Correct: true}
	}
	return Result{Earned: -qv.NegativeMarks, Maximum: qv.Marks}
}

func scoreTrueFalse(qv models.QuestionVersion, response map[string]any) Result {
	// correct_answer: {"value": true}
	correctVal, ok := qv.CorrectAnswer["value"].(bool)
	if !ok {
		return Result{Maximum: qv.Marks}
	}
	givenVal, ok := response["value"].(bool)
	if !ok {
		return Result{Maximum: qv.Marks}
	}
	if correctVal == givenVal {
		return Result{Earned: qv.Marks, Maximum: qv.Marks, Correct: true}
	}
	return Result{Earned: -qv.NegativeMarks, Maximum: qv.Marks}
}

func scoreText(qv models.QuestionVersion, response map[string]any) Result {
	// Deterministic exact-match or case-insensitive match.
	// correct_answer: {"text": "Paris", "case_sensitive": false}
	correctText := stringVal(qv.CorrectAnswer, "text")
	givenText := stringVal(response, "text")
	if correctText == "" {
		return Result{Maximum: qv.Marks}
	}
	caseSensitive := false
	if v, ok := qv.CorrectAnswer["case_sensitive"].(bool); ok {
		caseSensitive = v
	}
	var matches bool
	if caseSensitive {
		matches = correctText == givenText
	} else {
		matches = strings.EqualFold(strings.TrimSpace(correctText), strings.TrimSpace(givenText))
	}
	if matches {
		return Result{Earned: qv.Marks, Maximum: qv.Marks, Correct: true}
	}
	return Result{Maximum: qv.Marks}
}

func scoreNumeric(qv models.QuestionVersion, response map[string]any) Result {
	// correct_answer: {"value": 42, "tolerance": 0.01}
	correctVal, err := toFloat64(qv.CorrectAnswer["value"])
	if err != nil {
		return Result{Maximum: qv.Marks}
	}
	givenVal, err := toFloat64(response["value"])
	if err != nil {
		return Result{Maximum: qv.Marks}
	}
	tolerance := 0.0
	if tv, ok := qv.CorrectAnswer["tolerance"]; ok {
		if fv, err := toFloat64(tv); err == nil {
			tolerance = fv
		}
	}
	if math.Abs(correctVal-givenVal) <= tolerance {
		return Result{Earned: qv.Marks, Maximum: qv.Marks, Correct: true}
	}
	return Result{Maximum: qv.Marks}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func stringVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		s, _ := v.(string)
		return s
	}
	return ""
}

func stringSliceVal(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch sv := v.(type) {
	case []string:
		return sv
	case []interface{}:
		var out []string
		for _, item := range sv {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func toSet(ss []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

func setsEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	}
	return 0, fmt.Errorf("not a number: %T", v)
}
