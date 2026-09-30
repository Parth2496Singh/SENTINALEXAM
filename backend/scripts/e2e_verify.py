import json
import urllib.request
import sys

BASE_URL = "http://localhost:8080/api/v1"
ADMIN_TOKEN = "Bearer dev:exam_admin:00000000-0000-0000-0000-000000000001"
CAND_TOKEN = "Bearer dev:candidate:00000000-0000-0000-0000-000000000002"

def request(method, path, token, payload=None):
    url = f"{BASE_URL}{path}"
    data = json.dumps(payload).encode('utf-8') if payload else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", token)
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req) as resp:
            body = resp.read().decode('utf-8')
            return resp.status, json.loads(body) if body else {}
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8')
        return e.code, json.loads(body) if body else {}

print("=== 1. Create Question Bank ===")
status, qb = request("POST", "/question-banks", ADMIN_TOKEN, {
    "name": "Python E2E Test Bank",
    "code": "QB-PY-E2E",
    "description": "Python verification bank"
})
assert status == 201, f"Failed QB create: {qb}"
qb_id = qb["id"]
print(f"Question Bank ID: {qb_id}")

print("=== 2. Create Question ===")
status, q = request("POST", f"/question-banks/{qb_id}/questions", ADMIN_TOKEN, {
    "type": "SINGLE_CHOICE",
    "default_marks": 10
})
assert status == 201, f"Failed Q create: {q}"
q_id = q["id"]
print(f"Question ID: {q_id}")

print("=== 3. Create & Publish Question Version ===")
status, qv = request("POST", f"/question-banks/{qb_id}/questions/{q_id}/versions", ADMIN_TOKEN, {
    "content": {"stem": "What is the capital of France?", "options": [{"key": "A", "text": "Paris"}, {"key": "B", "text": "Berlin"}]},
    "correct_answer": {"selected": "A"}
})
assert status == 201, f"Failed QV create: {qv}"
qv_id = qv["id"]

status, _ = request("POST", f"/question-banks/{qb_id}/questions/{q_id}/versions/{qv_id}/publish", ADMIN_TOKEN)
assert status == 200, "Failed publish QV"
print(f"Question Version Published ID: {qv_id}")

print("=== 4. Create & Publish Exam ==="
status, exam = request("POST", "/exams", ADMIN_TOKEN, {
    "title": "Cybersecurity Fundamentals",
    "code": "EX-CYBER-101",
    "duration_minutes": 60,
    "pass_marks": 10,
    "total_marks": 100
})
assert status == 201, f"Failed Exam create: {exam}"
exam_id = exam["id"]

status, _ = request("POST", f"/exams/{exam_id}/questions", ADMIN_TOKEN, {
    "question_version_id": qv_id,
    "marks": 10
})
assert status == 201, "Failed attach question"

status, _ = request("POST", f"/exams/{exam_id}/publish", ADMIN_TOKEN)
assert status == 200, "Failed publish exam"
print(f"Exam Created & Published ID: {exam_id}")

print("=== 5. Register Candidate & Assign ===")
status, cand = request("POST", "/candidates", ADMIN_TOKEN, {
    "keycloak_sub": "00000000-0000-0000-0000-000000000002",
    "email": "candidate.py@sentinelexam.com",
    "first_name": "Python",
    "last_name": "Candidate"
})
assert status in (201, 200), f"Failed candidate create: {cand}"
cand_id = cand["id"]

status, _ = request("POST", f"/exams/{exam_id}/assignments", ADMIN_TOKEN, {
    "candidate_id": cand_id
})
assert status == 201, "Failed candidate assignment"
print(f"Candidate Assigned ID: {cand_id}")

print("=== 6. Candidate Start Attempt ===")
status, att = request("POST", f"/candidate/exams/{exam_id}/start", CAND_TOKEN)
assert status == 201, f"Failed attempt start: {att}"
att_id = att["id"]
print(f"Attempt Started ID: {att_id}")

print("=== 7. Candidate Fetch Runtime & Security Verification ===")
status, runtime = request("GET", f"/attempts/{att_id}/runtime", CAND_TOKEN)
assert status == 200, f"Failed runtime fetch: {runtime}"

questions = runtime.get("questions", [])
assert len(questions) > 0, "No questions in candidate runtime!"

for r_q in questions:
    assert "correct_answer" not in r_q, f"SECURITY VIOLATION: correct_answer exposed in {r_q}"

print("[PASS] Security policy verified: correct_answer is stripped from candidate payload.")

print("=== 8. Session & Heartbeat ===")
status, sess = request("POST", f"/attempts/{att_id}/sessions", CAND_TOKEN, {"client_info": {"browser": "Automated Verified Browser"}})
assert status == 201, f"Failed session create: {sess}"
sess_id = sess["id"]

status, _ = request("POST", f"/sessions/{sess_id}/heartbeat", CAND_TOKEN, {"network_quality": "GOOD"})
assert status == 200, "Failed heartbeat"
print(f"Session & Heartbeat verified ID: {sess_id}")

print("=== 9. Save Response ===")
status, resp = request("PUT", f"/attempts/{att_id}/responses", CAND_TOKEN, {
    "question_version_id": qv_id,
    "value": {"selected": "A"}
})
assert status == 200, f"Failed save response: {resp}"
print("Candidate response successfully saved.")

print("=== 10. Submit Attempt ==="
status, sub = request("POST", f"/attempts/{att_id}/submit", CAND_TOKEN)
assert status == 200, f"Failed submit: {sub}"
print(f"Attempt Submitted: Score = {sub.get('score', 0)}")

print("=== 11. Response Modification Lock Verification ==="
status, err_resp = request("PUT", f"/attempts/{att_id}/responses", CAND_TOKEN, {
    "question_version_id": qv_id,
    "value": {"selected": "B"}
})
assert status == 400, f"SECURITY VIOLATION: finalized attempt allowed response mutation! Got status {status}"
print("[PASS] Finalized attempt response lock verified: post-submission edit rejected with HTTP 400.")

print("==================================================")
print("ALL E2E FUNCTIONAL AND SECURITY TESTS SUCCEEDED 100%")
print("==================================================")
