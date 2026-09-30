# Domain Model Specification

## 1. Core Domain Aggregates

SentinelExam defines distinct domain boundaries organized as aggregates within the modular architecture.

```
+---------------------------------------------------------------------------------+
|                                 DOMAIN BOUNDARIES                               |
+---------------------------------------------------------------------------------+
| 1. Assessment Authoring & Definition                                            |
|    - QuestionBank, Question, QuestionVersion, Exam, ExamSection, ExamQuestion   |
+---------------------------------------------------------------------------------+
| 2. Candidate & Identity                                                         |
|    - User, CandidateProfile, ExamAssignment                                     |
+---------------------------------------------------------------------------------+
| 3. Examination Runtime & Execution                                              |
|    - ExamAttempt, AssessmentSession, CandidateResponse, CandidateEvent, WAL     |
+---------------------------------------------------------------------------------+
| 4. Operations & Incident Management                                             |
|    - Incident, ImpactAssessment, RecoveryCase, OperationAction                  |
+---------------------------------------------------------------------------------+
| 5. Trust, Verification & Audit                                                  |
|    - AuditEvent, AuditCheckpoint, MerkleTreeRoot, SignatureProof                |
+---------------------------------------------------------------------------------+
```

## 2. Entity Descriptions & Invariants

### 2.1 Question & Versioning
* **QuestionBank**: Logical container of subject matter items.
* **Question**: Identity wrapper for an item across its revisions.
* **QuestionVersion**: Immutable snapshot of content, stimulus, scoring rubric, and metadata (e.g. QTI 3 JSON/XML export mapping). Once referenced in a LIVE exam, a QuestionVersion is locked.

### 2.2 Exam & Sections
* **Exam**: The aggregate root defining duration, time windows, security policies, and section progression rules.
* **ExamSection**: Sub-unit of an exam (e.g., Quantitative, Verbal) with independent timers and item randomizers.
* **ExamQuestion**: The junction between an exam section and an immutable `QuestionVersion`, with assigned weights and display ordering.

### 2.3 Candidate & Assignment
* **User**: IAM mirror entity from Keycloak.
* **Candidate**: Specific assessment participant profile.
* **ExamAssignment**: Pre-condition binding a candidate to an exam with allowed attempt windows and accommodation multipliers.

### 2.4 Attempt & Session Runtime
* **ExamAttempt**: The primary stateful assessment transaction. Tracks start time, computed hard deadline, current status, and cumulative score.
* **AssessmentSession**: Individual connected client lifecycle (browser tab / device session). A single Attempt may span multiple sequential Sessions (e.g. following browser restart).
* **CandidateResponse**: Authoritative consolidated answer state for an item in an attempt.
* **CandidateEvent**: Granular, sequenced, immutable audit trail event (e.g., `OPTION_SELECTED`, `FOCUS_LOST`, `TEXT_MUTATED`).

### 2.5 Operations & Resilience
* **Incident**: System- or candidate-level anomaly (e.g. widespread network partition, proctor intervention).
* **ImpactAssessment**: Quantified analysis of candidate time or data lost during an incident.
* **RecoveryCase**: Formal workflow for compensating candidate attempts (e.g. granted time credit, session unlock).

### 2.6 Trust & Audit
* **AuditEvent**: Cryptographically linked audit entry (`prev_hash`, `payload_hash`, `signature`).
* **AuditCheckpoint**: Periodic Merkle tree root hash computed over all events within an assessment window, anchored for non-repudiation.
