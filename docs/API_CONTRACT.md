# API Contract Specification

## 1. Global Standards

* **Base URL**: `/api/v1`
* **Format**: JSON (`Content-Type: application/json`)
* **Character Encoding**: UTF-8
* **Authentication**: Bearer JWT (`Authorization: Bearer <token>`) issued by Keycloak.
* **Correlation**: Every request returns a `X-Correlation-ID` header.

## 2. Standard Error Response Envelope

All API errors return a standard JSON structure:

```json
{
  "error": {
    "code": "SEQUENCE_CONFLICT",
    "message": "The event sequence number has a gap or unexpected sequence.",
    "details": {
      "attempt_id": "00000000-0000-0000-0000-000000000001",
      "expected_sequence": 45,
      "last_committed_sequence": 44
    },
    "correlation_id": "req-987654321",
    "timestamp": "2026-09-28T21:00:00Z"
  }
}
```

## 3. Core Health Endpoints

### 3.1 GET `/health/live`
* **Purpose**: Kubernetes/Docker liveness probe. Indicates if the HTTP server process is running.
* **Semantics**: Must NOT fail merely because external dependencies (PostgreSQL, NATS) are temporarily reconnecting.
* **Status**: `200 OK`
```json
{
  "status": "UP",
  "service": "sentinel-backend",
  "version": "1.0.0"
}
```

### 3.2 GET `/health/ready`
* **Purpose**: Readiness probe. Indicates if the service can safely handle user traffic.
* **Semantics**: Checks critical infrastructure (PostgreSQL & NATS connection pool status).
* **Isolation Guarantee**: AI / LiteLLM availability must NOT make the assessment API unready.
* **Status**: `200 OK` (when ready) or `503 Service Unavailable`.
```json
{
  "status": "READY",
  "components": {
    "database": "UP",
    "nats": "UP",
    "temporal": "UP",
    "ai_service": "DEGRADED" 
  }
}
```

### 3.3 GET `/health/startup`
* **Purpose**: Startup probe. Verifies database migrations and initial stream registrations are complete.
* **Status**: `200 OK`

---

## 4. Operational & Assessment Route Overview

### 4.1 Assessment Runtime (Candidate & Exam Core)
* `POST /api/v1/exams/{exam_id}/attempts`: Initiates the attempt, starts timer.
* `GET /api/v1/attempts/{attempt_id}/runtime`: Fetches attempt metadata, questions, and responses.
* `POST /api/v1/attempts/{attempt_id}/events:sync`: Ingests and synchronizes candidate WAL event batches (idempotent, sequential, transactional outbox).
* `PUT /api/v1/attempts/{attempt_id}/responses/{question_version_id}`: Direct response save / fallback endpoint.
* `POST /api/v1/attempts/{attempt_id}/sessions`: Creates or binds a client assessment session.
* `POST /api/v1/sessions/{session_id}/heartbeat`: Session heartbeat and network quality reporting.
* `POST /api/v1/sessions/{session_id}/terminate`: Explicit session termination.
* `POST /api/v1/attempts/{attempt_id}/submit`: Explicit candidate submission (freezes responses and scores attempt).

### 4.2 Authoring & Exam Administration
* `GET /api/v1/question-banks`: List question banks.
* `POST /api/v1/question-banks`: Create question bank (`name`, `description`).
* `GET /api/v1/question-banks/{bankId}`: Get question bank and questions.
* `POST /api/v1/question-banks/{bankId}/questions`: Create question and initial version snapshot.
* `GET /api/v1/questions/{questionId}`: Get question details.
* `POST /api/v1/questions/{questionId}/publish`: Publish question version to ACTIVE status.
* `POST /api/v1/questions/{questionId}/retire`: Retire question.
* `GET /api/v1/exams`: List exams.
* `POST /api/v1/exams`: Create draft exam.
* `GET /api/v1/exams/{examId}`: Get exam details.
* `PUT /api/v1/exams/{examId}`: Update draft exam metadata.
* `POST /api/v1/exams/{examId}/questions`: Attach question version to draft exam.
* `POST /api/v1/exams/{examId}/publish`: Publish exam (DRAFT -> SCHEDULED).
* `POST /api/v1/exams/{examId}/start`: Start exam (SCHEDULED -> LIVE).
* `POST /api/v1/exams/{examId}/pause`: Pause exam (LIVE -> PAUSED).
* `POST /api/v1/exams/{examId}/resume`: Resume exam (PAUSED -> LIVE).
* `POST /api/v1/exams/{examId}/complete`: Mark exam completed.
* `POST /api/v1/exams/{examId}/archive`: Archive exam.
* `GET /api/v1/candidates`: List candidates (Admin).
* `POST /api/v1/candidates`: Register candidate identity.
* `GET /api/v1/candidates/{candidateId}`: Get candidate.
* `GET /api/v1/exams/{examId}/assignments`: List exam assignments.
* `POST /api/v1/exams/{examId}/assignments`: Assign candidate to exam.
* `DELETE /api/v1/exams/{examId}/assignments/{candidateId}`: Revoke candidate assignment.

### 4.3 Operations & Incident Command Center (Phase 5)
* `GET /api/v1/operations/exams/{exam_id}/live-status`: Aggregated metrics.
* `POST /api/v1/operations/incidents`: Report infrastructure or candidate incident.
* `POST /api/v1/operations/recovery/compensate`: Submit time extension or state repair.

### 4.4 Audit & Trust (Phase 6)
* `GET /api/v1/audit/attempts/{attempt_id}/chain`: Retrieve SHA-256 hash chain.
* `GET /api/v1/audit/checkpoints/{checkpoint_id}`: Retrieve Merkle root and proofs.

---

## 5. Phase 4 Resilience Contract Additions

### 5.1 POST `/api/v1/attempts/{attempt_id}/events:sync`
* **Purpose**: Idempotent batch synchronization of candidate Write-Ahead Log (WAL) events.
* **Authentication**: Bearer JWT (`candidate` or `exam_admin`).
* **Request Body**:
```json
{
  "events": [
    {
      "event_id": "00000000-0000-0000-0000-000000000101",
      "session_id": "00000000-0000-0000-0000-000000000010",
      "session_epoch": 1,
      "sequence_number": 42,
      "event_type": "candidate.option_selected",
      "client_instance_id": "browser-tab-xyz",
      "client_timestamp": "2026-09-30T18:20:00.100Z",
      "payload": {
        "question_version_id": "00000000-0000-0000-0000-000000000020",
        "value": { "selected": "B" }
      }
    }
  ]
}
```
* **Success Response (`200 OK`)**:
```json
{
  "data": {
    "accepted_count": 1,
    "last_committed_sequence": 42,
    "server_timestamp": "2026-09-30T18:20:00.250Z"
  }
}
```
* **Error Responses**:
  * `401 Unauthorized`: Missing or invalid bearer token.
  * `403 Forbidden`: Candidate does not own the attempt or assignment revoked.
  * `409 Conflict` (`SEQUENCE_CONFLICT`): Event sequence gap detected or stale session epoch.
    ```json
    {
      "error": {
        "code": "SEQUENCE_CONFLICT",
        "message": "Sequence gap detected",
        "details": {
          "expected_sequence": 42,
          "last_committed_sequence": 41
        }
      }
    }
    ```
  * `409 Conflict` (`STALE_SESSION_EPOCH`): Event emitted from a superseded session epoch.
  * `409 Conflict` (`ATTEMPT_NOT_ACTIVE` / `ATTEMPT_FINALIZED`): Attempt already finalized or terminated.
  * `410 Gone` (`ATTEMPT_EXPIRED`): Server-authoritative expiration reached.
  * `422 Unprocessable Entity` (`VALIDATION_ERROR`): Malformed event schema or invalid question reference.
