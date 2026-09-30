package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	baseURL      = "http://localhost:8080/api/v1"
	healthLive   = "http://localhost:8080/health/live"
	healthReady  = "http://localhost:8080/health/ready"
	adminToken   = "Bearer dev:exam_admin:00000000-0000-0000-0000-000000000001"
	sysAdminToken = "Bearer dev:system_admin:00000000-0000-0000-0000-000000000099"
)

type TestResult struct {
	Category string
	Feature  string
	Endpoint string
	Expected string
	Actual   string
	Passed   bool
	Details  string
}

var results []TestResult

func record(category, feature, endpoint, expected, actual string, passed bool, details string) {
	results = append(results, TestResult{
		Category: category,
		Feature:  feature,
		Endpoint: endpoint,
		Expected: expected,
		Actual:   actual,
		Passed:   passed,
		Details:  details,
	})
	status := "[PASS]"
	if !passed {
		status = "[FAIL]"
	}
	fmt.Printf("%s [%s] %s (%s) -> Expected: %s | Actual: %s\n", status, category, feature, endpoint, expected, actual)
	if !passed && details != "" {
		fmt.Printf("   Details: %s\n", details)
	}
}

func doReq(method, urlStr, token string, payload any) (int, map[string]any, string) {
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return 0, nil, err.Error()
	}
	if token != "" {
		r.Header.Set("Authorization", token)
	}
	r.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return 0, nil, err.Error()
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var res map[string]any
	_ = json.Unmarshal(bodyBytes, &res)
	return resp.StatusCode, res, string(bodyBytes)
}

func apiReq(method, path, token string, payload any) (int, map[string]any, string) {
	return doReq(method, baseURL+path, token, payload)
}

func extractID(m map[string]any) string {
	if val, ok := m["id"].(string); ok {
		return val
	}
	if val, ok := m["ID"].(string); ok {
		return val
	}
	if data, ok := m["data"].(map[string]any); ok {
		if val, ok := data["id"].(string); ok {
			return val
		}
		if val, ok := data["ID"].(string); ok {
			return val
		}
		if q, ok := data["question"].(map[string]any); ok {
			if val, ok := q["ID"].(string); ok {
				return val
			}
			if val, ok := q["id"].(string); ok {
				return val
			}
		}
	}
	return ""
}

func extractVersionID(m map[string]any) string {
	if data, ok := m["data"].(map[string]any); ok {
		if v, ok := data["version"].(map[string]any); ok {
			if val, ok := v["ID"].(string); ok {
				return val
			}
			if val, ok := v["id"].(string); ok {
				return val
			}
		}
	}
	return extractID(m)
}

func main() {
	ts := time.Now().UnixNano()
	candSub1 := uuid.New().String()
	candToken1 := fmt.Sprintf("Bearer dev:candidate:%s", candSub1)
	candSub2 := uuid.New().String()
	candToken2 := fmt.Sprintf("Bearer dev:candidate:%s", candSub2)

	fmt.Println("================================================================================")
	fmt.Println("TITANVAULT SENTINELEXAM COMPLETE API-LEVEL INTEGRATION SUITE")
	fmt.Println("================================================================================")

	// -------------------------------------------------------------------------
	// 0. HEALTH & PLATFORM ENDPOINTS
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 0: Health & Platform ---")
	{
		status, _, body := doReq("GET", healthLive, "", nil)
		record("Health", "Liveness Probe", "GET /health/live", "HTTP 200 UP", fmt.Sprintf("HTTP %d", status), status == 200, body)

		status, _, body = doReq("GET", healthReady, "", nil)
		record("Health", "Readiness Probe", "GET /health/ready", "HTTP 200 READY", fmt.Sprintf("HTTP %d", status), status == 200, body)

		status, _, body = apiReq("GET", "/info", "", nil)
		record("Platform", "Platform Info", "GET /api/v1/info", "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)
	}

	// -------------------------------------------------------------------------
	// 1. AUTHENTICATION & ROLE AUTHORIZATION
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 1: Authentication & Authorization ---")
	{
		// Missing Token on Admin Endpoint
		status, _, body := apiReq("GET", "/question-banks", "", nil)
		record("Auth", "Missing Token", "GET /api/v1/question-banks", "HTTP 401 UNAUTHORIZED", fmt.Sprintf("HTTP %d", status), status == 401, body)

		// Malformed Token
		status, _, body = apiReq("GET", "/question-banks", "Bearer invalid-junk-format", nil)
		record("Auth", "Malformed Token", "GET /api/v1/question-banks", "HTTP 401 UNAUTHORIZED", fmt.Sprintf("HTTP %d", status), status == 401, body)

		// Candidate Attempting Admin Endpoint
		status, _, body = apiReq("GET", "/question-banks", candToken1, nil)
		record("Auth", "Role Boundary (Candidate -> Admin API)", "GET /api/v1/question-banks", "HTTP 403 FORBIDDEN", fmt.Sprintf("HTTP %d", status), status == 403, body)

		// Valid Admin Token
		status, _, body = apiReq("GET", "/question-banks", adminToken, nil)
		record("Auth", "Valid Admin Token", "GET /api/v1/question-banks", "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)
	}

	// -------------------------------------------------------------------------
	// 2. QUESTION BANK LIFECYCLE
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 2: Question Banks ---")
	var qbID string
	{
		// List Question Banks
		status, _, body := apiReq("GET", "/question-banks", adminToken, nil)
		record("QuestionBank", "List Question Banks", "GET /api/v1/question-banks", "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Invalid Create (Missing Name)
		status, _, body = apiReq("POST", "/question-banks", adminToken, map[string]any{
			"description": "Missing name bank",
		})
		record("QuestionBank", "Invalid Create (Missing Name)", "POST /api/v1/question-banks", "HTTP 422 VALIDATION_ERROR", fmt.Sprintf("HTTP %d", status), status == 422, body)

		// Valid Create
		status, resp, body := apiReq("POST", "/question-banks", adminToken, map[string]any{
			"name":        fmt.Sprintf("Integration QB %d", ts),
			"description": "Integration Test Question Bank",
		})
		qbID = extractID(resp)
		record("QuestionBank", "Create Question Bank", "POST /api/v1/question-banks", "HTTP 201 with UUID", fmt.Sprintf("HTTP %d (ID: %s)", status, qbID), status == 201 && qbID != "", body)

		// Get Question Bank
		status, _, body = apiReq("GET", fmt.Sprintf("/question-banks/%s", qbID), adminToken, nil)
		record("QuestionBank", "Get Question Bank", fmt.Sprintf("GET /api/v1/question-banks/%s", qbID), "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Get Nonexistent Question Bank
		status, _, body = apiReq("GET", fmt.Sprintf("/question-banks/%s", uuid.New().String()), adminToken, nil)
		record("QuestionBank", "Get Nonexistent Bank", "GET /api/v1/question-banks/{randomUUID}", "HTTP 404 NOT_FOUND", fmt.Sprintf("HTTP %d", status), status == 404, body)
	}

	// -------------------------------------------------------------------------
	// 3. QUESTION & VERSION LIFECYCLE
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 3: Questions & Versions ---")
	var qID, qvID string
	{
		// Invalid Question Type
		status, _, body := apiReq("POST", fmt.Sprintf("/question-banks/%s/questions", qbID), adminToken, map[string]any{
			"type": "INVALID_QUESTION_TYPE",
			"content": map[string]any{"stem": "test"},
		})
		record("Question", "Invalid Question Type", "POST /api/v1/question-banks/{id}/questions", "HTTP 422 VALIDATION_ERROR", fmt.Sprintf("HTTP %d", status), status == 422, body)

		// Valid Create Question + Initial Draft Version
		status, resp, body := apiReq("POST", fmt.Sprintf("/question-banks/%s/questions", qbID), adminToken, map[string]any{
			"type": "SINGLE_CHOICE",
			"content": map[string]any{
				"stem": "What is the port of HTTPS?",
				"options": []map[string]string{
					{"key": "A", "text": "80"},
					{"key": "B", "text": "443"},
					{"key": "C", "text": "22"},
				},
			},
			"correct_answer": map[string]string{"selected": "B"},
			"marks":          10,
			"negative_marks": 0,
		})
		qID = extractID(resp)
		qvID = extractVersionID(resp)
		record("Question", "Create Question & Draft Version", "POST /api/v1/question-banks/{id}/questions", "HTTP 201 with Question & Version ID", fmt.Sprintf("HTTP %d (Q: %s, V: %s)", status, qID, qvID), status == 201 && qID != "" && qvID != "", body)

		// Get Question
		status, _, body = apiReq("GET", fmt.Sprintf("/questions/%s", qID), adminToken, nil)
		record("Question", "Get Question", fmt.Sprintf("GET /api/v1/questions/%s", qID), "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Candidate attempting to get question directly
		status, _, body = apiReq("GET", fmt.Sprintf("/questions/%s", qID), candToken1, nil)
		record("Question", "Candidate Accessing Question Authoring API", "GET /api/v1/questions/{id}", "HTTP 403 FORBIDDEN", fmt.Sprintf("HTTP %d", status), status == 403, body)

		// Publish Question Version
		status, _, body = apiReq("POST", fmt.Sprintf("/questions/%s/publish", qID), adminToken, nil)
		record("Question", "Publish Question Version", fmt.Sprintf("POST /api/v1/questions/%s/publish", qID), "HTTP 200 (ACTIVE)", fmt.Sprintf("HTTP %d", status), status == 200, body)
	}

	// -------------------------------------------------------------------------
	// 4. EXAM LIFECYCLE & STATE MACHINE
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 4: Exams & State Machine ---")
	var examID string
	{
		// Invalid Exam (Duration <= 0)
		status, _, body := apiReq("POST", "/exams", adminToken, map[string]any{
			"title":            "Invalid Duration Exam",
			"duration_seconds": 0,
		})
		record("Exam", "Invalid Duration", "POST /api/v1/exams", "HTTP 422 VALIDATION_ERROR", fmt.Sprintf("HTTP %d", status), status == 422, body)

		// Valid Create Draft Exam
		status, resp, body := apiReq("POST", "/exams", adminToken, map[string]any{
			"title":            fmt.Sprintf("Integration Exam %d", ts),
			"description":      "Exam for integration audit",
			"duration_seconds": 3600,
			"max_attempts":     1,
		})
		examID = extractID(resp)
		record("Exam", "Create Draft Exam", "POST /api/v1/exams", "HTTP 201 (DRAFT)", fmt.Sprintf("HTTP %d (ID: %s)", status, examID), status == 201 && examID != "", body)

		// Attempt to Publish Empty Exam (0 questions attached)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/publish", examID), adminToken, nil)
		record("Exam", "Publish Empty Exam", fmt.Sprintf("POST /api/v1/exams/%s/publish", examID), "HTTP 422 PUBLISH_VALIDATION_FAILED", fmt.Sprintf("HTTP %d", status), status == 422, body)

		// Add Question to Draft Exam
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/questions", examID), adminToken, map[string]any{
			"question_version_id": qvID,
			"position":            1,
		})
		record("Exam", "Attach Question to Draft Exam", fmt.Sprintf("POST /api/v1/exams/%s/questions", examID), "HTTP 201", fmt.Sprintf("HTTP %d", status), status == 201, body)

		// Attempt Invalid State Transition: Start directly from DRAFT (must be SCHEDULED/PUBLISHED first)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/start", examID), adminToken, nil)
		record("Exam", "Invalid Transition (DRAFT -> LIVE)", fmt.Sprintf("POST /api/v1/exams/%s/start", examID), "HTTP 409 INVALID_STATE_TRANSITION", fmt.Sprintf("HTTP %d", status), status == 409, body)

		// Publish Exam (DRAFT -> SCHEDULED)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/publish", examID), adminToken, nil)
		record("Exam", "Publish Exam (DRAFT -> SCHEDULED)", fmt.Sprintf("POST /api/v1/exams/%s/publish", examID), "HTTP 200 (SCHEDULED)", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Attempt to Add Question to SCHEDULED Exam (immutable)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/questions", examID), adminToken, map[string]any{
			"question_version_id": qvID,
			"position":            2,
		})
		record("Exam", "Mutate Non-Draft Exam", fmt.Sprintf("POST /api/v1/exams/%s/questions", examID), "HTTP 409 INVALID_STATE_TRANSITION", fmt.Sprintf("HTTP %d", status), status == 409, body)

		// Start Exam (SCHEDULED -> LIVE)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/start", examID), adminToken, nil)
		record("Exam", "Start Exam (SCHEDULED -> LIVE)", fmt.Sprintf("POST /api/v1/exams/%s/start", examID), "HTTP 200 (LIVE)", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Pause Exam (LIVE -> PAUSED)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/pause", examID), adminToken, nil)
		record("Exam", "Pause Exam (LIVE -> PAUSED)", fmt.Sprintf("POST /api/v1/exams/%s/pause", examID), "HTTP 200 (PAUSED)", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Resume Exam (PAUSED -> LIVE)
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/resume", examID), adminToken, nil)
		record("Exam", "Resume Exam (PAUSED -> LIVE)", fmt.Sprintf("POST /api/v1/exams/%s/resume", examID), "HTTP 200 (LIVE)", fmt.Sprintf("HTTP %d", status), status == 200, body)
	}

	// -------------------------------------------------------------------------
	// 5. CANDIDATE REGISTRATION & IDENTITY BINDING
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 5: Candidates & Identity Binding ---")
	var cand1ID, cand2ID string
	{
		// Register Candidate 1
		status, resp, body := apiReq("POST", "/candidates", candToken1, map[string]any{
			"keycloak_sub": candSub1,
			"email":        fmt.Sprintf("cand1_%d@example.com", ts),
			"full_name":    "Alice TestCandidate",
		})
		cand1ID = extractID(resp)
		record("Candidate", "Register Candidate 1", "POST /api/v1/candidates", "HTTP 201 with ID", fmt.Sprintf("HTTP %d (ID: %s)", status, cand1ID), status == 201 && cand1ID != "", body)

		// Register Candidate 2
		status, resp, body = apiReq("POST", "/candidates", candToken2, map[string]any{
			"keycloak_sub": candSub2,
			"email":        fmt.Sprintf("cand2_%d@example.com", ts),
			"full_name":    "Bob TestCandidate",
		})
		cand2ID = extractID(resp)
		record("Candidate", "Register Candidate 2", "POST /api/v1/candidates", "HTTP 201 with ID", fmt.Sprintf("HTTP %d (ID: %s)", status, cand2ID), status == 201 && cand2ID != "", body)

		// Idempotent Candidate Registration
		status, resp, body = apiReq("POST", "/candidates", candToken1, map[string]any{
			"keycloak_sub": candSub1,
			"full_name":    "Alice TestCandidate",
		})
		reID := extractID(resp)
		record("Candidate", "Idempotent Candidate Register", "POST /api/v1/candidates", "HTTP 201 (Same ID)", fmt.Sprintf("HTTP %d (ID: %s)", status, reID), status == 201 && reID == cand1ID, body)

		// Candidate attempting to view full candidate list (Admin only)
		status, _, body = apiReq("GET", "/candidates", candToken1, nil)
		record("Candidate", "Candidate Directory Boundary", "GET /api/v1/candidates", "HTTP 403 FORBIDDEN", fmt.Sprintf("HTTP %d", status), status == 403, body)
	}

	// -------------------------------------------------------------------------
	// 6. ASSIGNMENTS
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 6: Assignments ---")
	{
		// Candidate attempting to start unassigned exam
		status, _, body := apiReq("POST", fmt.Sprintf("/exams/%s/attempts", examID), candToken1, nil)
		record("Assignment", "Start Attempt Without Assignment", fmt.Sprintf("POST /api/v1/exams/%s/attempts", examID), "HTTP 403 FORBIDDEN", fmt.Sprintf("HTTP %d", status), status == 403, body)

		// Assign Candidate 1 to Exam
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/assignments", examID), adminToken, map[string]any{
			"candidate_id": cand1ID,
		})
		record("Assignment", "Assign Candidate 1", fmt.Sprintf("POST /api/v1/exams/%s/assignments", examID), "HTTP 201 ASSIGNED", fmt.Sprintf("HTTP %d", status), status == 201, body)

		// Idempotent Assignment
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/assignments", examID), adminToken, map[string]any{
			"candidate_id": cand1ID,
		})
		record("Assignment", "Duplicate/Idempotent Assignment", fmt.Sprintf("POST /api/v1/exams/%s/assignments", examID), "HTTP 201 ASSIGNED", fmt.Sprintf("HTTP %d", status), status == 201, body)

		// Revoke Assignment & Verify Block
		status, _, body = apiReq("DELETE", fmt.Sprintf("/exams/%s/assignments/%s", examID, cand1ID), adminToken, nil)
		record("Assignment", "Revoke Assignment", fmt.Sprintf("DELETE /api/v1/exams/%s/assignments/{id}", examID), "HTTP 200 REVOKED", fmt.Sprintf("HTTP %d", status), status == 200, body)

		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/attempts", examID), candToken1, nil)
		record("Assignment", "Attempt Start with Revoked Assignment", fmt.Sprintf("POST /api/v1/exams/%s/attempts", examID), "HTTP 403 ASSIGNMENT_REVOKED", fmt.Sprintf("HTTP %d", status), status == 403, body)

		// Re-assign Candidate 1
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/assignments", examID), adminToken, map[string]any{
			"candidate_id": cand1ID,
		})
		record("Assignment", "Re-assign Candidate 1", fmt.Sprintf("POST /api/v1/exams/%s/assignments", examID), "HTTP 201 ASSIGNED", fmt.Sprintf("HTTP %d", status), status == 201, body)

		// Candidate 1 fetches candidate-scoped assignments (GET /api/v1/candidate/assignments)
		status, _, body = apiReq("GET", "/candidate/assignments", candToken1, nil)
		record("Assignment", "Candidate 1 Get My Assignments", "GET /api/v1/candidate/assignments", "HTTP 200 (Has 1 Assignment)", fmt.Sprintf("HTTP %d", status), status == 200 && strings.Contains(body, examID), body)

		// Candidate 2 (unassigned) fetches candidate-scoped assignments -> Empty list
		status, _, body = apiReq("GET", "/candidate/assignments", candToken2, nil)
		record("Assignment", "Candidate 2 Get My Assignments (Empty)", "GET /api/v1/candidate/assignments", "HTTP 200 (Empty List)", fmt.Sprintf("HTTP %d", status), status == 200 && !strings.Contains(body, examID), body)

		// Admin calling candidate-scoped endpoint -> Forbidden (Role Boundary)
		status, _, body = apiReq("GET", "/candidate/assignments", adminToken, nil)
		record("Assignment", "Admin Accessing Candidate Assignment Endpoint", "GET /api/v1/candidate/assignments", "HTTP 403 FORBIDDEN", fmt.Sprintf("HTTP %d", status), status == 403, body)
	}

	// -------------------------------------------------------------------------
	// 7. ATTEMPT RUNTIME, SESSIONS, RESPONSES & SUBMISSION
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Section 7: Attempt Runtime, Session & Responses ---")
	var attemptID, sessionID string
	{
		// Start Attempt for Candidate 1
		status, resp, body := apiReq("POST", fmt.Sprintf("/exams/%s/attempts", examID), candToken1, nil)
		attemptID = extractID(resp)
		record("Attempt", "Start Attempt (Candidate 1)", fmt.Sprintf("POST /api/v1/exams/%s/attempts", examID), "HTTP 201 (ACTIVE)", fmt.Sprintf("HTTP %d (ID: %s)", status, attemptID), status == 201 && attemptID != "", body)

		// Attempting second attempt when max_attempts = 1
		status, _, body = apiReq("POST", fmt.Sprintf("/exams/%s/attempts", examID), candToken1, nil)
		record("Attempt", "Enforce max_attempts Limit", fmt.Sprintf("POST /api/v1/exams/%s/attempts", examID), "HTTP 422 VALIDATION_ERROR", fmt.Sprintf("HTTP %d", status), status == 422, body)

		// Candidate 2 accessing Candidate 1's attempt runtime (Isolation Check)
		status, _, body = apiReq("GET", fmt.Sprintf("/attempts/%s/runtime", attemptID), candToken2, nil)
		record("Security", "Cross-Candidate Attempt Isolation", fmt.Sprintf("GET /api/v1/attempts/%s/runtime", attemptID), "HTTP 404 NOT_FOUND (Enum Shield)", fmt.Sprintf("HTTP %d", status), status == 404, body)

		// Candidate 1 Fetch Runtime
		status, _, body = apiReq("GET", fmt.Sprintf("/attempts/%s/runtime", attemptID), candToken1, nil)
		hasAnswerLeak := strings.Contains(body, "correct_answer")
		record("Security", "Runtime Payload Security (No Answer Leak)", fmt.Sprintf("GET /api/v1/attempts/%s/runtime", attemptID), "HTTP 200 & correct_answer omitted", fmt.Sprintf("HTTP %d (Leaked: %t)", status, hasAnswerLeak), status == 200 && !hasAnswerLeak, body)

		// Create Session for Attempt
		status, resp, body = apiReq("POST", fmt.Sprintf("/attempts/%s/sessions", attemptID), candToken1, map[string]any{
			"client_instance_id": "integration-client-001",
		})
		sessionID = extractID(resp)
		record("Session", "Create Session", fmt.Sprintf("POST /api/v1/attempts/%s/sessions", attemptID), "HTTP 201 with Session ID", fmt.Sprintf("HTTP %d (ID: %s)", status, sessionID), status == 201 && sessionID != "", body)

		// Heartbeat
		status, _, body = apiReq("POST", fmt.Sprintf("/sessions/%s/heartbeat", sessionID), candToken1, map[string]any{
			"network_quality": "EXCELLENT",
		})
		record("Session", "Session Heartbeat", fmt.Sprintf("POST /api/v1/sessions/%s/heartbeat", sessionID), "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Save Response (Correct Answer: B)
		status, _, body = apiReq("PUT", fmt.Sprintf("/attempts/%s/responses/%s", attemptID, qvID), candToken1, map[string]any{
			"value": map[string]string{"selected": "B"},
		})
		record("Response", "Save Candidate Response", fmt.Sprintf("PUT /api/v1/attempts/%s/responses/{qvId}", attemptID), "HTTP 200", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Submit Attempt
		status, subResp, body := apiReq("POST", fmt.Sprintf("/attempts/%s/submit", attemptID), candToken1, nil)
		record("Submission", "Submit Attempt", fmt.Sprintf("POST /api/v1/attempts/%s/submit", attemptID), "HTTP 200 FINALIZED", fmt.Sprintf("HTTP %d", status), status == 200, body)

		// Verify Deterministic Scoring
		var score float64
		if data, ok := subResp["data"].(map[string]any); ok {
			if s, ok := data["Score"].(float64); ok {
				score = s
			} else if s, ok := data["score"].(float64); ok {
				score = s
			}
		}
		record("Submission", "Deterministic Score Verification", "Scoring Verification", "Score = 10.0", fmt.Sprintf("Score = %.1f", score), score == 10.0, body)

		// Post-Submission Response Mutation Lock (Must return HTTP 409 Conflict)
		status, _, body = apiReq("PUT", fmt.Sprintf("/attempts/%s/responses/%s", attemptID, qvID), candToken1, map[string]any{
			"value": map[string]string{"selected": "A"},
		})
		record("Security", "Post-Submit Response Immutability", fmt.Sprintf("PUT /api/v1/attempts/%s/responses/{qvId}", attemptID), "HTTP 409 CONFLICT", fmt.Sprintf("HTTP %d", status), status == 409, body)
	}

	// -------------------------------------------------------------------------
	// 8. SUMMARY REPORT
	// -------------------------------------------------------------------------
	fmt.Println("\n================================================================================")
	fmt.Println("INTEGRATION AUDIT SUMMARY REPORT")
	fmt.Println("================================================================================")
	total := len(results)
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("Total Tests: %d | Passed: %d | Failed: %d\n", total, passed, failed)

	if failed > 0 {
		fmt.Printf("\nFAILED TESTS:\n")
		for _, r := range results {
			if !r.Passed {
				fmt.Printf("- [%s] %s (%s) -> Expected: %s, Got: %s\n", r.Category, r.Feature, r.Endpoint, r.Expected, r.Actual)
			}
		}
		os.Exit(1)
	}

	fmt.Println("\nALL BACKEND API INTEGRATION AND SECURITY CONTRACTS PASSED (100%)")
}
