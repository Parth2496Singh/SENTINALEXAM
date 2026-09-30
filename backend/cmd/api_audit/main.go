package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

const (
	baseURL    = "http://localhost:8080/api/v1"
	adminToken = "Bearer dev:exam_admin:00000000-0000-0000-0000-000000000001"
)

type testResult struct {
	Feature  string
	Endpoint string
	Expected string
	Actual   string
	Status   string
	Details  string
}

var results []testResult

func record(feature, endpoint, expected, actual, status, details string) {
	results = append(results, testResult{
		Feature:  feature,
		Endpoint: endpoint,
		Expected: expected,
		Actual:   actual,
		Status:   status,
		Details:  details,
	})
	fmt.Printf("[%s] %s | %s -> %s\n", status, feature, endpoint, details)
}

func doReq(method, path, token string, body any) (int, map[string]any, string) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, baseURL+path, bodyReader)
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

	respBytes, _ := io.ReadAll(resp.Body)
	var resMap map[string]any
	json.Unmarshal(respBytes, &resMap)
	return resp.StatusCode, resMap, string(respBytes)
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
			if val, ok := q["id"].(string); ok {
				return val
			}
			if val, ok := q["ID"].(string); ok {
				return val
			}
		}
	}
	return ""
}

func extractVersionID(m map[string]any) string {
	if data, ok := m["data"].(map[string]any); ok {
		if v, ok := data["version"].(map[string]any); ok {
			if val, ok := v["id"].(string); ok {
				return val
			}
			if val, ok := v["ID"].(string); ok {
				return val
			}
		}
	}
	return extractID(m)
}

func main() {
	ts := uuid.New().String()[:8]
	candSub := uuid.New().String()
	candToken := fmt.Sprintf("Bearer dev:candidate:%s", candSub)

	fmt.Println("==========================================================")
	fmt.Println("   SENTINELEXAM AUTOMATED API INTEGRATION TEST SUITE     ")
	fmt.Println("==========================================================")

	// --- 1. AUTH & ROLE RESTRICTIONS ---
	status, _, _ := doReq("GET", "/question-banks", "", nil)
	if status == 401 {
		record("Auth", "GET /question-banks (No token)", "HTTP 401", "HTTP 401", "PASS", "Missing token correctly rejected")
	} else {
		record("Auth", "GET /question-banks (No token)", "HTTP 401", fmt.Sprintf("HTTP %d", status), "FAIL", "Missing token allowed or wrong error")
	}

	status, _, _ = doReq("GET", "/question-banks", candToken, nil)
	if status == 403 {
		record("Auth", "GET /question-banks (Candidate token)", "HTTP 403", "HTTP 403", "PASS", "Role restriction candidate->admin enforced")
	} else {
		record("Auth", "GET /question-banks (Candidate token)", "HTTP 403", fmt.Sprintf("HTTP %d", status), "FAIL", "Candidate accessed admin route")
	}

	// --- 2. QUESTION BANKS ---
	status, qbResp, _ := doReq("POST", "/question-banks", adminToken, map[string]any{
		"name":        fmt.Sprintf("Audit Bank %s", ts),
		"description": "Integration test question bank",
	})
	qbID := extractID(qbResp)
	if status == 201 && qbID != "" {
		record("QuestionBank", "POST /question-banks", "HTTP 201 + ID", fmt.Sprintf("HTTP %d ID=%s", status, qbID), "PASS", "Question bank created")
	} else {
		record("QuestionBank", "POST /question-banks", "HTTP 201 + ID", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to create question bank")
	}

	status, _, _ = doReq("GET", fmt.Sprintf("/question-banks/%s", qbID), adminToken, nil)
	if status == 200 {
		record("QuestionBank", "GET /question-banks/{id}", "HTTP 200", "HTTP 200", "PASS", "Fetched question bank detail")
	} else {
		record("QuestionBank", "GET /question-banks/{id}", "HTTP 200", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to fetch question bank")
	}

	// --- 3. QUESTION AUTHORING & PUBLISHING ---
	status, qResp, _ := doReq("POST", fmt.Sprintf("/question-banks/%s/questions", qbID), adminToken, map[string]any{
		"type": "SINGLE_CHOICE",
		"content": map[string]any{
			"stem": "Select option A",
			"options": []map[string]string{
				{"key": "A", "text": "Option A"},
				{"key": "B", "text": "Option B"},
			},
		},
		"correct_answer": map[string]string{"selected": "A"},
		"marks":          10,
		"negative_marks": 0,
	})
	qID := extractID(qResp)
	qvID := extractVersionID(qResp)
	if status == 201 && qID != "" && qvID != "" {
		record("Questions", "POST /question-banks/{id}/questions", "HTTP 201 + QuestionID & VersionID", fmt.Sprintf("HTTP %d qID=%s qvID=%s", status, qID, qvID), "PASS", "Question + Version 1 created")
	} else {
		record("Questions", "POST /question-banks/{id}/questions", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to create question")
	}

	status, _, _ = doReq("POST", fmt.Sprintf("/questions/%s/publish", qID), adminToken, nil)
	if status == 200 {
		record("Questions", "POST /questions/{id}/publish", "HTTP 200", "HTTP 200", "PASS", "Question version published")
	} else {
		record("Questions", "POST /questions/{id}/publish", "HTTP 200", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to publish question")
	}

	// --- 4. EXAM LIFECYCLE ---
	status, examResp, _ := doReq("POST", "/exams", adminToken, map[string]any{
		"title":            fmt.Sprintf("Audit Exam %s", ts),
		"description":      "Integration test exam",
		"duration_seconds": 3600,
		"max_attempts":     1,
	})
	examID := extractID(examResp)
	if status == 201 && examID != "" {
		record("Exams", "POST /exams", "HTTP 201 + ExamID", fmt.Sprintf("HTTP %d examID=%s", status, examID), "PASS", "Draft exam created")
	} else {
		record("Exams", "POST /exams", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to create exam")
	}

	status, _, _ = doReq("POST", fmt.Sprintf("/exams/%s/questions", examID), adminToken, map[string]any{
		"question_version_id": qvID,
		"position":            1,
	})
	if status == 201 {
		record("Exams", "POST /exams/{id}/questions", "HTTP 201", "HTTP 201", "PASS", "Question attached to exam")
	} else {
		record("Exams", "POST /exams/{id}/questions", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to attach question")
	}

	status, _, _ = doReq("POST", fmt.Sprintf("/exams/%s/publish", examID), adminToken, nil)
	if status == 200 {
		record("Exams", "POST /exams/{id}/publish", "HTTP 200", "HTTP 200", "PASS", "Exam published")
	} else {
		record("Exams", "POST /exams/{id}/publish", "HTTP 200", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to publish exam")
	}

	status, _, _ = doReq("POST", fmt.Sprintf("/exams/%s/start", examID), adminToken, nil)
	if status == 200 {
		record("Exams", "POST /exams/{id}/start", "HTTP 200", "HTTP 200", "PASS", "Exam set LIVE")
	} else {
		record("Exams", "POST /exams/{id}/start", "HTTP 200", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to start exam")
	}

	// --- 5. CANDIDATE & ASSIGNMENT ---
	status, candResp, _ := doReq("POST", "/candidates", candToken, map[string]any{
		"keycloak_sub": candSub,
		"full_name":    "Audit Candidate",
	})
	candID := extractID(candResp)
	if status == 201 && candID != "" {
		record("Candidates", "POST /candidates", "HTTP 201 + CandidateID", fmt.Sprintf("HTTP %d candID=%s", status, candID), "PASS", "Candidate registered")
	} else {
		record("Candidates", "POST /candidates", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to register candidate")
	}

	status, _, _ = doReq("POST", fmt.Sprintf("/exams/%s/assignments", examID), adminToken, map[string]any{
		"candidate_id": candID,
	})
	if status == 201 {
		record("Assignments", "POST /exams/{id}/assignments", "HTTP 201", "HTTP 201", "PASS", "Candidate assigned to exam")
	} else {
		record("Assignments", "POST /exams/{id}/assignments", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to assign candidate")
	}

	// --- 6. ATTEMPT & RUNTIME ---
	status, attResp, _ := doReq("POST", fmt.Sprintf("/exams/%s/attempts", examID), candToken, nil)
	attID := extractID(attResp)
	if status == 201 && attID != "" {
		record("Attempts", "POST /exams/{id}/attempts", "HTTP 201 + AttemptID", fmt.Sprintf("HTTP %d attID=%s", status, attID), "PASS", "Attempt started")
	} else {
		record("Attempts", "POST /exams/{id}/attempts", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to start attempt")
	}

	status, _, rawRuntime := doReq("GET", fmt.Sprintf("/attempts/%s/runtime", attID), candToken, nil)
	if status == 200 && !strings.Contains(rawRuntime, "correct_answer") {
		record("Security Policy", "GET /attempts/{id}/runtime", "HTTP 200 without correct_answer", "HTTP 200 clean payload", "PASS", "Candidate runtime stripped of correct answers")
	} else {
		record("Security Policy", "GET /attempts/{id}/runtime", "HTTP 200 without correct_answer", fmt.Sprintf("HTTP %d exposed=%v", status, strings.Contains(rawRuntime, "correct_answer")), "FAIL", "Security policy violation in runtime")
	}

	// --- 7. SESSIONS & RESPONSES ---
	status, sessResp, _ := doReq("POST", fmt.Sprintf("/attempts/%s/sessions", attID), candToken, map[string]any{
		"client_instance_id": "audit-client-001",
	})
	sessID := extractID(sessResp)
	if status == 201 && sessID != "" {
		record("Sessions", "POST /attempts/{id}/sessions", "HTTP 201 + SessionID", fmt.Sprintf("HTTP %d sessID=%s", status, sessID), "PASS", "Session created")
	} else {
		record("Sessions", "POST /attempts/{id}/sessions", "HTTP 201", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to create session")
	}

	status, _, _ = doReq("PUT", fmt.Sprintf("/attempts/%s/responses/%s", attID, qvID), candToken, map[string]any{
		"value": map[string]string{"selected": "A"},
	})
	if status == 200 {
		record("Responses", "PUT /attempts/{id}/responses/{qvId}", "HTTP 200", "HTTP 200", "PASS", "Candidate response saved")
	} else {
		record("Responses", "PUT /attempts/{id}/responses/{qvId}", "HTTP 200", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to save response")
	}

	status, _, _ = doReq("POST", fmt.Sprintf("/attempts/%s/submit", attID), candToken, nil)
	if status == 200 {
		record("Submission", "POST /attempts/{id}/submit", "HTTP 200", "HTTP 200", "PASS", "Attempt submitted and scored")
	} else {
		record("Submission", "POST /attempts/{id}/submit", "HTTP 200", fmt.Sprintf("HTTP %d", status), "FAIL", "Failed to submit attempt")
	}

	status, _, _ = doReq("PUT", fmt.Sprintf("/attempts/%s/responses/%s", attID, qvID), candToken, map[string]any{
		"value": map[string]string{"selected": "B"},
	})
	if status == 409 {
		record("Immutability", "PUT /attempts/{id}/responses/{qvId} post-submit", "HTTP 409 ATTEMPT_NOT_ACTIVE", "HTTP 409", "PASS", "Post-submission edit locked")
	} else {
		record("Immutability", "PUT /attempts/{id}/responses/{qvId} post-submit", "HTTP 409", fmt.Sprintf("HTTP %d", status), "FAIL", "Post-submission edit allowed")
	}

	fmt.Println("\n==========================================================")
	fmt.Println("             INTEGRATION AUDIT SUMMARY TABLE              ")
	fmt.Println("==========================================================")
	fmt.Printf("%-18s | %-40s | %-6s | %s\n", "FEATURE", "ENDPOINT", "STATUS", "DETAILS")
	fmt.Println(strings.Repeat("-", 90))
	hasFailures := false
	for _, r := range results {
		fmt.Printf("%-18s | %-40s | %-6s | %s\n", r.Feature, r.Endpoint, r.Status, r.Details)
		if r.Status == "FAIL" {
			hasFailures = true
		}
	}
	fmt.Println(strings.Repeat("-", 90))
	if hasFailures {
		fmt.Println("STATUS: FAIL — API defects detected.")
		os.Exit(1)
	} else {
		fmt.Println("STATUS: PASS — ALL API ENDPOINTS VERIFIED 100% HEALTHY.")
	}
}
