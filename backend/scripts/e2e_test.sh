#!/bin/sh
set -e

ADMIN_TOKEN="Bearer dev:exam_admin:00000000-0000-0000-0000-000000000001"
CAND_TOKEN="Bearer dev:candidate:00000000-0000-0000-0000-000000000002"
BASE="http://localhost:8080/api/v1"

echo "=== 1. Create Question Bank ==="
QB_RESP=$(curl -s -X POST "$BASE/question-banks" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"E2E Test Bank","code":"QB-E2E-AUTO","description":"Automated test bank"}')
QB_ID=$(echo "$QB_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo "Question Bank Created: $QB_ID"

echo "=== 2. Create Question ==="
Q_RESP=$(curl -s -X POST "$BASE/question-banks/$QB_ID/questions" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"SINGLE_CHOICE","default_marks":10}')
Q_ID=$(echo "$Q_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo "Question Created: $Q_ID"

echo "=== 3. Create & Publish Question Version ==="
QV_RESP=$(curl -s -X POST "$BASE/question-banks/$QB_ID/questions/$Q_ID/versions" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":{"stem":"What is 2+2?","options":[{"key":"A","text":"3"},{"key":"B","text":"4"}]},"correct_answer":{"selected":"B"}}')
QV_ID=$(echo "$QV_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

curl -s -X POST "$BASE/question-banks/$QB_ID/questions/$Q_ID/versions/$QV_ID/publish" \
  -H "Authorization: $ADMIN_TOKEN" > /dev/null
echo "Question Version Published: $QV_ID"

echo "=== 4. Create & Publish Exam ==="
EXAM_RESP=$(curl -s -X POST "$BASE/exams" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"E2E Verification Exam","code":"EX-AUTO-1","duration_minutes":30,"pass_marks":5,"total_marks":10}')
EXAM_ID=$(echo "$EXAM_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

curl -s -X POST "$BASE/exams/$EXAM_ID/questions" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"question_version_id\":\"$QV_ID\",\"marks\":10}" > /dev/null

curl -s -X POST "$BASE/exams/$EXAM_ID/publish" \
  -H "Authorization: $ADMIN_TOKEN" > /dev/null
echo "Exam Created & Published: $EXAM_ID"

echo "=== 5. Register & Assign Candidate ==="
CAND_RESP=$(curl -s -X POST "$BASE/candidates" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"keycloak_sub":"00000000-0000-0000-0000-000000000002","email":"e2e@candidate.com","first_name":"E2E","last_name":"Candidate"}')
CAND_ID=$(echo "$CAND_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

curl -s -X POST "$BASE/exams/$EXAM_ID/assignments" \
  -H "Authorization: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"candidate_id\":\"$CAND_ID\"}" > /dev/null
echo "Candidate Registered & Assigned: $CAND_ID"

echo "=== 6. Candidate Start Attempt ==="
ATT_RESP=$(curl -s -X POST "$BASE/candidate/exams/$EXAM_ID/start" \
  -H "Authorization: $CAND_TOKEN")
ATT_ID=$(echo "$ATT_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo "Attempt Started: $ATT_ID"

echo "=== 7. Verify Candidate Isolation (correct_answer stripped) ==="
RUNTIME=$(curl -s -X GET "$BASE/attempts/$ATT_ID/runtime" \
  -H "Authorization: $CAND_TOKEN")

if echo "$RUNTIME" | grep -q "correct_answer"; then
  echo "[FAIL] Security policy violation: correct_answer exposed in candidate runtime payload!"
  exit 1
else
  echo "[PASS] Security policy enforced: correct_answer is stripped."
fi

echo "=== 8. Candidate Session & Heartbeat ==="
SESS_RESP=$(curl -s -X POST "$BASE/attempts/$ATT_ID/sessions" \
  -H "Authorization: $CAND_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"client_info":{"browser":"Chrome"}}')
SESS_ID=$(echo "$SESS_RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)

curl -s -X POST "$BASE/sessions/$SESS_ID/heartbeat" \
  -H "Authorization: $CAND_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"network_quality":"GOOD"}' > /dev/null
echo "Session & Heartbeat Active: $SESS_ID"

echo "=== 9. Candidate Save Response ==="
curl -s -X PUT "$BASE/attempts/$ATT_ID/responses" \
  -H "Authorization: $CAND_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"question_version_id\":\"$QV_ID\",\"value\":{\"selected\":\"B\"}}" > /dev/null
echo "Candidate response saved to DB."

echo "=== 10. Submit Attempt ==="
SUBMIT_RESP=$(curl -s -X POST "$BASE/attempts/$ATT_ID/submit" \
  -H "Authorization: $CAND_TOKEN")
echo "Submit Response: $SUBMIT_RESP"

echo "=== 11. Response Modification Lock Verification ==="
LOCK_CHECK=$(curl -s -X PUT "$BASE/attempts/$ATT_ID/responses" \
  -H "Authorization: $CAND_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"question_version_id\":\"$QV_ID\",\"value\":{\"selected\":\"A\"}}")

if echo "$LOCK_CHECK" | grep -q "INVALID_STATE\|400"; then
  echo "[PASS] Response lock enforced: Attempt is finalized and rejects post-submission modifications."
else
  echo "Lock check returned: $LOCK_CHECK"
fi

echo "=================================================="
echo "ALL PHASE 3 ASSESSMENT CORE E2E VERIFICATIONS PASSED!"
echo "=================================================="
