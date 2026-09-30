# State Machines Specification

This document details the precise, valid state transitions for all domain aggregates within SentinelExam. Any state transition not explicitly listed here is prohibited and must be rejected by domain validation.

---

## 1. Exam Lifecycle

An Exam defines the template and schedule for assessments.

```
[DRAFT] ---> [SCHEDULED] ---> [LIVE] <---> [PAUSED]
                                |
                                v
                           [COMPLETED] ---> [ARCHIVED]
```

* **DRAFT**: Created, authoring questions and sections. Editable.
* **SCHEDULED**: Published with a future start time. Authoring locked.
* **LIVE**: Candidate attempts may be initiated.
* **PAUSED**: Temporarily suspended (e.g. operational hold). Active attempts freeze timers.
* **COMPLETED**: Time window closed. No new attempts allowed. Existing attempts finalize.
* **ARCHIVED**: Read-only historical record.

---

## 2. Question Lifecycle

Questions support immutable version snapshots.

```
[DRAFT] ---> [ACTIVE] ---> [SUPERSEDED] ---> [RETIRED]
```

* **DRAFT**: Authoring or revision in progress.
* **ACTIVE**: Approved and available for inclusion in exams.
* **SUPERSEDED**: A new QuestionVersion has replaced this item for future exams.
* **RETIRED**: No longer permitted in any future exams.

---

## 3. Assignment Lifecycle

```
[ASSIGNED] <---> [REVOKED]
```

* **ASSIGNED**: Candidate is eligible to start the assessment.
* **REVOKED**: Eligibility administratively revoked.

---

## 4. Attempt Lifecycle

An Attempt represents a candidate's execution of an exam.

```
[NOT_STARTED] ---> [ACTIVE] ---> [SUBMITTED]  ---> [FINALIZED]
                      |
                      +--------> [EXPIRED]    ---> [FINALIZED]
                      |
                      +--------> [TERMINATED] ---> [FINALIZED]
```

* **NOT_STARTED**: Created upon assignment, awaiting candidate launch.
* **ACTIVE**: Candidate has entered the exam; countdown timer is running.
* **SUBMITTED**: Candidate explicitly submitted the attempt.
* **EXPIRED**: Server-side deadline reached without explicit submission.
* **TERMINATED**: Administratively terminated due to security violation or proctor action.
* **FINALIZED**: Grading and audit checkpoints completed; results locked.

---

## 5. Assessment Session Lifecycle

Tracks physical client connections.

```
[CREATED] ---> [CONNECTED] <---> [DISCONNECTED]
                     |                  |
                     |                  v
                     +---------> [RECONNECTING] ---> [CONNECTED]
                     |                  |
                     v                  v
                [TERMINATED] <----------+
```

* **CREATED**: Client handshake initiated.
* **CONNECTED**: Active WebSocket / SSE / Heartbeat stream established.
* **DISCONNECTED**: Heartbeat missed or explicit disconnect.
* **RECONNECTING**: Re-authentication and state replay in progress.
* **TERMINATED**: Session ended (attempt closed or revoked).

---

## 6. Candidate Response Lifecycle

Tracks the status of a specific item answer within an Attempt.

```
[UNANSWERED] <---> [ANSWERED] ---> [SUBMITTED]
```

* **UNANSWERED**: No answer selected/entered.
* **ANSWERED**: Candidate has selected/typed an answer.
* *Note*: Transition `ANSWERED` → `UNANSWERED` is explicitly allowed when a candidate clears their answer.
* **SUBMITTED**: Attempt submitted; response frozen for grading.

---

## 7. Candidate Event Lifecycle (Ingestion Pipeline)

Every event emitted from the candidate client undergoes strict ingestion validation.

```
[RECEIVED] ---> [VALIDATING] ---> [VALIDATED] ---> [COMMITTED]
                     |
                     +----------> [REJECTED]
                     |
                     +----------> [DUPLICATE]
                     |
                     +----------> [CONFLICT]
```

* **RECEIVED**: Payload reached the ingestion gateway.
* **VALIDATING**: Checking cryptographic signature, sequence number, and attempt state.
* **VALIDATED**: Passed validation; queued to NATS JetStream & PostgreSQL Outbox.
* **COMMITTED**: Persisted in PostgreSQL and reflected in authoritative attempt state.
* **REJECTED**: Signature failure, schema violation, or post-deadline timestamp.
* **DUPLICATE**: Client resubmitted an already committed sequence ID.
* **CONFLICT**: Event sequence gap or optimistic concurrency mismatch requiring sync replay.

---

## 8. Incident Lifecycle

```
[DETECTED] ---> [ACKNOWLEDGED] ---> [INVESTIGATING] ---> [MITIGATING] ---> [RESOLVED] ---> [CLOSED]
```

---

## 9. Impact Assessment Lifecycle

```
[PENDING] ---> [EVALUATING] ---> [ASSESSED] ---> [RESOLVED]
                                    |
                                    +----------> [UNRESOLVED]
```

---

## 10. Recovery Case Lifecycle

```
[RECOVERY_REQUIRED] ---> [EVALUATING] ---> [AWAITING_APPROVAL] ---> [EXECUTING] ---> [VERIFYING] ---> [VERIFIED] ---> [CLOSED]
                                |                     |                  |
                                v                     v                  v
                             [FAILED]              [FAILED]           [FAILED]
```

---

## 11. Audit Checkpoint Lifecycle

```
[PENDING] ---> [BUILDING] ---> [COMPUTED] ---> [VERIFIED]
                                  |
                                  v
                               [INVALID]
```
