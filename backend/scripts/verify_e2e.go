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
	baseURL    = "http://localhost:8080/api/v1"
	adminToken = "Bearer dev:exam_admin:00000000-0000-0000-0000-000000000001"
)

func getID(m map[string]any) string {
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
	b, _ := json.Marshal(m)
	panic(fmt.Sprintf("could not find id in response: %s", string(b)))
}

func getVersionID(m map[string]any) string {
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
	return getID(m)
}

func main() {
	ts := time.Now().UnixNano()
	candSub := uuid.New().String()
	candToken := fmt.Sprintf("Bearer dev:candidate:%s", candSub)

	fmt.Println("=== 1. Create Question Bank ===")
	qbResp := req("POST", "/question-banks", adminToken, map[string]any{
		"name":        fmt.Sprintf("Go E2E Bank %d", ts),
		"code":        fmt.Sprintf("QB-%d", ts%100000),
		"description": "Go E2E test bank",
	})
	qbID := getID(qbResp)
	fmt.Printf("Question Bank ID: %s\n", qbID)

	fmt.Println("=== 2. Create Question & Initial Version ===")
	qResp := req("POST", fmt.Sprintf("/question-banks/%s/questions", qbID), adminToken, map[string]any{
		"type": "SINGLE_CHOICE",
		"content": map[string]any{
			"stem": "What is the result of 5 + 5?",
			"options": []map[string]string{
				{"key": "A", "text": "10"},
				{"key": "B", "text": "20"},
			},
		},
		"correct_answer": map[string]string{"selected": "A"},
		"marks":          10,
		"negative_marks": 0,
	})
	qID := getID(qResp)
	qvID := getVersionID(qResp)
	fmt.Printf("Question ID: %s, Version ID: %s\n", qID, qvID)

	fmt.Println("=== 3. Publish Question ===")
	req("POST", fmt.Sprintf("/questions/%s/publish", qID), adminToken, nil)
	fmt.Printf("Question Version Published ID: %s\n", qvID)

	fmt.Println("=== 4. Create, Publish & Start Exam (LIVE) ===")
	examResp := req("POST", "/exams", adminToken, map[string]any{
		"title":            fmt.Sprintf("Exam %d", ts),
		"code":             fmt.Sprintf("EX-%d", ts%100000),
		"duration_seconds": 3600,
		"pass_marks":       10,
		"total_marks":      100,
	})
	examID := getID(examResp)

	req("POST", fmt.Sprintf("/exams/%s/questions", examID), adminToken, map[string]any{
		"question_version_id": qvID,
		"position":            1,
	})

	req("POST", fmt.Sprintf("/exams/%s/publish", examID), adminToken, nil)
	req("POST", fmt.Sprintf("/exams/%s/start", examID), adminToken, nil)
	fmt.Printf("Exam Created, Published & Set LIVE ID: %s\n", examID)

	fmt.Println("=== 5. Register Candidate & Assign ===")
	candResp := req("POST", "/candidates", candToken, map[string]any{
		"keycloak_sub": candSub,
		"full_name":    "GoTest Candidate",
	})
	candID := getID(candResp)

	req("POST", fmt.Sprintf("/exams/%s/assignments", examID), adminToken, map[string]any{
		"candidate_id": candID,
	})
	fmt.Printf("Candidate Registered & Assigned ID: %s\n", candID)

	fmt.Println("=== 6. Candidate Start Attempt ===")
	attResp := req("POST", fmt.Sprintf("/exams/%s/attempts", examID), candToken, nil)
	attID := getID(attResp)
	fmt.Printf("Attempt Started ID: %s\n", attID)

	fmt.Println("=== 7. Candidate Fetch Runtime & Security Policy Check ===")
	runtimeBody := rawReq("GET", fmt.Sprintf("/attempts/%s/runtime", attID), candToken, nil, 200)
	if strings.Contains(runtimeBody, "correct_answer") {
		fmt.Println("[FAIL] Security policy failure: correct_answer exposed in candidate runtime payload!")
		os.Exit(1)
	}
	fmt.Println("[PASS] Security policy verified: correct_answer is stripped from candidate payload.")

	fmt.Println("=== 8. Candidate Session & Heartbeat ===")
	sessResp := req("POST", fmt.Sprintf("/attempts/%s/sessions", attID), candToken, map[string]any{
		"client_instance_id": "e2e-client-001",
	})
	sessID := getID(sessResp)

	req("POST", fmt.Sprintf("/sessions/%s/heartbeat", sessID), candToken, map[string]string{
		"network_quality": "EXCELLENT",
	})
	fmt.Printf("Session & Heartbeat Active ID: %s\n", sessID)

	fmt.Println("=== 9. Candidate Save Response ===")
	req("PUT", fmt.Sprintf("/attempts/%s/responses/%s", attID, qvID), candToken, map[string]any{
		"value": map[string]string{"selected": "A"},
	})
	fmt.Println("Candidate response saved to DB.")

	fmt.Println("=== 10. Submit Attempt ===")
	subResp := req("POST", fmt.Sprintf("/attempts/%s/submit", attID), candToken, nil)
	fmt.Printf("Attempt Submitted Successfully: Result = %v\n", subResp)

	fmt.Println("=== 11. Response Modification Lock Verification ===")
	lockCheckBody := rawReq("PUT", fmt.Sprintf("/attempts/%s/responses/%s", attID, qvID), candToken, map[string]any{
		"value": map[string]string{"selected": "B"},
	}, 409)

	if strings.Contains(lockCheckBody, "ATTEMPT_NOT_ACTIVE") || strings.Contains(lockCheckBody, "INVALID_STATE") || strings.Contains(lockCheckBody, "finalized") || strings.Contains(lockCheckBody, "submitted") || strings.Contains(lockCheckBody, "EXPIRED") {
		fmt.Println("[PASS] Finalized attempt response lock verified: post-submission edit rejected with HTTP 409.")
	} else {
		fmt.Printf("[PASS] Lock verified. Received expected HTTP 409 response: %s\n", lockCheckBody)
	}

	fmt.Println("==================================================")
	fmt.Println("ALL E2E FUNCTIONAL AND SECURITY TESTS SUCCEEDED 100%")
	fmt.Println("==================================================")
}

func req(method, path, token string, payload any) map[string]any {
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Authorization", token)
	r.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		panic(fmt.Sprintf("HTTP %d error for %s %s: %s", resp.StatusCode, method, path, string(bodyBytes)))
	}

	var res map[string]any
	json.Unmarshal(bodyBytes, &res)
	return res
}

func rawReq(method, path, token string, payload any, expectedStatus int) string {
	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Authorization", token)
	r.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		panic(fmt.Sprintf("Expected HTTP %d, got HTTP %d for %s %s: %s", expectedStatus, resp.StatusCode, method, path, string(bodyBytes)))
	}
	return string(bodyBytes)
}
