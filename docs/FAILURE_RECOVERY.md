# Failure & Recovery Specification

## 1. Resilience Philosophy

Online assessments must withstand client crashes, device reboots, flaky mobile internet, and infrastructure interruptions without data loss or unfair time deductions.

## 2. Candidate-Side Write-Ahead Log (WAL)

The frontend uses browser **IndexedDB** as a durable local Write-Ahead Log (WAL):
1. **Immediate Local Commit**: Every user action (selecting an option, typing a character) is synchronously persisted to IndexedDB before any network dispatch.
2. **Deterministic Sequence Numbers**: Events are assigned monotonic sequence numbers (`1, 2, 3...`) per attempt.
3. **Background Sync Worker**: Flushes unacknowledged events to `POST /api/v1/attempts/{attempt_id}/events`.
4. **Offline Mode**: If the network drops, the candidate continues answering questions locally. All events queue in IndexedDB.
5. **Reconnection & Replay**: Upon network restoration, the sync worker transmits the backlogged WAL in ordered batches.

## 3. Server-Side Split-Brain & Conflict Resolution

* If an event arrives out-of-sequence, the server issues a `CONFLICT` response specifying the latest received sequence number.
* The client sync engine resets its cursor to the specified sequence number and replays missing items.

## 4. Operational Recovery via Temporal

When a broad outage occurs (e.g. data center partition):
1. An **Incident** is created.
2. An **ImpactAssessment** evaluates affected attempts and calculates lost duration.
3. A **RecoveryCase** workflow in Temporal is triggered (subject to human approval) to extend deadlines, restore attempts, or issue makeup tokens.
