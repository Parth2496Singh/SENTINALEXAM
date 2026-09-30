# Event Contract Specification

## 1. NATS JetStream Stream Topology

All event streaming within SentinelExam is mediated by NATS JetStream.

### Streams
* `SENTINEL_CANDIDATE_EVENTS`: Stream of raw candidate events (`sentinel.candidate.>`)
* `SENTINEL_ASSESSMENT_EVENTS`: Stream of lifecycle events (`sentinel.assessment.>`)
* `SENTINEL_INCIDENT_EVENTS`: Stream of operational incidents (`sentinel.incident.>`)
* `SENTINEL_AUDIT_EVENTS`: Stream of audit entries (`sentinel.audit.>`)

## 2. Event Envelope Standard

All events emitted to NATS JetStream conform to a standard envelope:

```json
{
  "specversion": "1.0",
  "id": "evt_01HXYZ...",
  "source": "sentinel.backend/ingest",
  "type": "sentinel.candidate.option_selected",
  "datacontenttype": "application/json",
  "time": "2026-09-28T21:00:00.123Z",
  "correlation_id": "req-987654321",
  "data": {
    "attempt_id": "att_01H...",
    "session_id": "ses_01H...",
    "sequence_number": 42,
    "question_id": "q_01H...",
    "selected_option": "B",
    "client_timestamp": "2026-09-28T21:00:00.100Z"
  }
}
```

## 3. Replay and Deduplication Semantics

1. **NATS JetStream Deduplication**: Uses `Nats-Msg-Id` derived from `<attempt_id>:<sequence_number>`.
2. **Transactional Outbox Worker**: Reads from PostgreSQL `outbox_events` and commits to NATS JetStream.
3. **Consumer Idempotency**: Consumers track processed event IDs in PostgreSQL `idempotency_records`.
