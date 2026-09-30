# Database Schema Specification

## 1. Relational Design & Invariants

SentinelExam relies on PostgreSQL as the single source of authoritative transactional truth. 

Key principles:
1. **Strong Typing & Enums**: Enums represent state machine states.
2. **Immutable History**: Core events and audit logs are append-only.
3. **Optimistic Locking**: State rows use integer `version` columns to prevent lost updates.
4. **Transactional Outbox**: Events published to NATS are first committed in an `outbox_events` table within the same ACID transaction.

---

## 2. Entity Overview

* `users`: Identity records synchronized from Keycloak (sub UUID, email, roles).
* `candidates`: Extended candidate profiles (accommodations, metadata).
* `question_banks`: Logical groupings of questions.
* `questions`: Item identifiers.
* `question_versions`: Immutable content, stimulus, scoring rubrics, and QTI JSON payloads.
* `exams`: High-level exam configuration, timers, security profile.
* `exam_sections`: Sections within an exam (e.g., Section A, Section B).
* `exam_questions`: Section-to-QuestionVersion mappings with weights.
* `exam_assignments`: Candidate assignments to exams.
* `exam_attempts`: Candidate runtime attempts (status, start_time, deadline, submitted_at, final_score).
* `assessment_sessions`: Device / browser sessions.
* `candidate_responses`: Consolidated answers for each question in an attempt.
* `candidate_events`: Sequenced append-only event stream per attempt.
* `submissions`: Final submission record with client signatures.
* `incidents`: Operational incidents and anomalies.
* `impact_assessments`: Impact calculation on attempts.
* `recovery_cases`: Formal compensation and remediation actions.
* `audit_events`: Hash-chained audit events (`prev_hash`, `payload_hash`, `signature`).
* `audit_checkpoints`: Merkle checkpoint trees.
* `outbox_events`: Transactional outbox table for JetStream publishing.
* `idempotency_records`: Request deduplication tracking.
* `operations`: System administrative logs.

---

## 3. Minimal Foundation Schema (Applied in Initial Migration)

For Phase 1 Foundation, the migration creates the schema metadata, migration version tracker, and baseline health proof tables.
