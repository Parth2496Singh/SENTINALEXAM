# SentinelExam — Resilient & Trustworthy Online Assessment Platform

[![MPOnline Hackathon Project](https://img.shields.io/badge/Hackathon-MPOnline%20Submission-0284C7.svg)](https://mponline.gov.in)
[![Backend Integration Tests](https://img.shields.io/badge/API%20Suite-50%2F50%20PASSED-059669.svg)]()
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-18.3-61DAFB.svg)](https://reactjs.org)
[![Database](https://img.shields.io/badge/PostgreSQL-16.0-336791.svg)](https://postgresql.org)
[![Event Stream](https://img.shields.io/badge/NATS-JetStream--2.10-008A00.svg)](https://nats.io)

**SentinelExam** is an enterprise-grade, offline-resilient digital assessment platform specifically engineered for high-concurrency public sector, government, and educational examinations. Designed with a Zero-Trust client policy, deterministic server-side evaluation, and an offline Write-Ahead Log (WAL) event engine, SentinelExam guarantees 100% exam session continuity even under catastrophic network degradation, power drops, or browser crashes.

---

## 📋 Table of Contents
1. [Core Innovation & Mission](#-core-innovation--mission)
2. [Key Architecture Pillars](#-key-architecture-pillars)
3. [End-to-End Product Lifecycle](#-end-to-end-product-lifecycle)
4. [Technology Stack & Infrastructure](#-technology-stack--infrastructure)
5. [Quick Start & Docker Deployment](#-quick-start--docker-deployment)
6. [API & Security Verification Suite](#-api--security-verification-suite)
7. [Comprehensive REST API Reference](#-comprehensive-rest-api-reference)
8. [Submission Document Sitemap](#-submission-document-sitemap)
9. [License & Acknowledgments](#-license--acknowledgments)

---

## 🎯 Core Innovation & Mission

High-stakes competitive and public recruitment examinations require strict data integrity, immutability, and resilience. Traditional assessment web applications fail when client connectivity drops or server pools experience sudden traffic spikes.

**SentinelExam resolves these challenges through three fundamental architectural breakthroughs**:

1. **Client-Side Write-Ahead Log (WAL)**: Candidates complete assessments locally inside a resilient IndexedDB outbox. Responses and event streams are captured sequentially at `t_0` regardless of internet availability.
2. **Idempotent Background Synchronization**: A dedicated outbox queue syncs logged candidate events (`candidate.option_selected`, `session.heartbeat`) to NATS JetStream in ordered batches (`POST /api/v1/attempts/{id}/events:sync`).
3. **Zero-Trust Authoritative Server Scoring**: The browser NEVER sends calculated scores or answer validation keys. Correct answers are stripped from candidate payloads, and all score evaluations are executed deterministically on the Go backend using transactional PostgreSQL state locks.

---

## 🛡️ Key Architecture Pillars

### 1. Assessment Lifecycle State Machine
Exams strictly enforce state transition boundaries to prevent unauthorized candidate access or post-hoc tampering:

```
[DRAFT] ──(Publish)──> [SCHEDULED] ──(Start)──> [LIVE] ──(Pause)──> [PAUSED]
  │                                               │                   │
  └──(Empty Publish Rejected)                     ├──(Resume)─────────┘
                                                  │
                                                  └──(Complete)──> [COMPLETED] ──> [ARCHIVED]
```

- **DRAFT**: Authoring phase. Questions, options, and marks can be mutated.
- **SCHEDULED**: Published & locked. Exam metadata and attached questions become **immutable**.
- **LIVE**: Open for candidate attempt initialization.
- **FINALIZED**: Post-submission state. Responses are locked permanently (`HTTP 409 CONFLICT` on edit attempts).

### 2. Candidate Identity & Data Isolation (Enum Shield)
- **Server-Derived Identity**: Candidate IDs are extracted strictly from Keycloak JWT claims (`claims.KeycloakSub`). Candidates cannot query or enumerate other candidates' assignments or attempts.
- **Enumeration Shielding**: Cross-candidate access attempts return `HTTP 404 NOT_FOUND` instead of `403` to prevent ID harvesting.

---

## 🔄 End-to-End Product Lifecycle

The complete lifecycle demonstrated in SentinelExam follows a strict 8-stage pipeline:

```
[1. Admin Login] ──> [2. Create Question Bank] ──> [3. Author & Publish Questions]
                                                                  │
[6. Candidate Portal] <── [5. Assign Candidate] <── [4. Create & Set Exam LIVE]
          │
          └──> [7. Assessment Runner (WAL + Sync)] ──> [8. Final Scorecard]
```

1. **Administrator Authentication**: Log in via IAM JWT token to access the Operations Dashboard.
2. **Question Bank Authoring**: Create domain-specific question repositories (`POST /api/v1/question-banks`).
3. **Versioned Question Creation**: Author questions with positive/negative scoring rules and publish version snapshots to `ACTIVE` status.
4. **Exam Configuration & Activation**: Create draft exam, attach published question versions, publish exam (`SCHEDULED`), and set to `LIVE`.
5. **Candidate Directory Registration**: Register test-taker identities and bind to Keycloak subject UUIDs.
6. **Exam Assignment**: Assign candidates to live exams (`POST /api/v1/exams/{id}/assignments`).
7. **Resilient Candidate Execution**: Candidate opens `/candidate`, launches attempt, answers questions with background IndexedDB WAL persistence and NATS event syncing.
8. **Final Submission & Scorecard**: Server scoring engine calculates score, locks response records, and outputs instant scorecard.

---

## 💻 Technology Stack & Infrastructure

| Layer | Component | Version | Purpose |
| :--- | :--- | :--- | :--- |
| **Frontend** | React SPA | 18.3.1 | Responsive candidate & admin portal UI |
| **Build System** | Vite | 5.4.21 | Lightning-fast HMR & production bundle transformation |
| **State & Fetch** | TanStack Query | 5.56.0 | Declarative server state & cache invalidation |
| **Local Storage** | IndexedDB WAL | Native | Client-side Write-Ahead Log event queue |
| **Backend Core** | Go REST API | 1.22.7 | High-throughput assessment API & state machine |
| **Database** | PostgreSQL | 16.0 | Authoritative transactional data store (`pgxpool`) |
| **Event Broker** | NATS JetStream | 2.10 | High-concurrency event stream distribution |
| **Orchestrator** | Temporal | 1.24.2 | Workflow engine for async timeouts & compensation |
| **Containers** | Docker Compose | v2.29 | Orchestrates 8 microservices & network topology |

---

## 🚀 Quick Start & Docker Deployment

### System Requirements
- **Docker Desktop** (with Compose v2)
- **RAM**: 4 GB minimum (8 GB recommended)
- **Ports Available**: `3000`, `8080`, `5432`, `4222`, `8081`, `8233`

### 1. Launch Microservices

```powershell
# Clone repository and navigate to root
cd X:\SentinelExam

# Build and start all 8 docker containers in background
docker compose up -d --build
```

### 2. Verify Container Health

```powershell
docker compose ps
```
*Expected Output: `sentinel-backend`, `sentinel-frontend`, `sentinel-postgres`, `sentinel-nats`, `sentinel-keycloak`, `sentinel-temporal`, `sentinel-temporal-ui`, `sentinel-toxiproxy` all UP/HEALTHY.*

### 3. Application Access Endpoints
- 🌐 **Candidate & Admin Web Portal**: [http://localhost:3000](http://localhost:3000)
- ⚙️ **Go REST API Base**: [http://localhost:8080/api/v1/info](http://localhost:8080/api/v1/info)
- 🟢 **Liveness Probe**: [http://localhost:8080/health/live](http://localhost:8080/health/live)
- 🟡 **Readiness Probe**: [http://localhost:8080/health/ready](http://localhost:8080/health/ready)
- 📊 **Temporal Workflow Console**: [http://localhost:8233](http://localhost:8233)

---

## 🧪 API & Security Verification Suite

SentinelExam comes equipped with a comprehensive 50-point automated integration and security test suite.

### 1. Execute Unit & Domain Tests

```powershell
docker compose exec backend go test ./...
```

### 2. Execute 50-Test Integration & Security Audit Suite

```powershell
docker compose exec backend go run ./cmd/api_integration_suite
```

**Output Summary**:
```text
================================================================================
INTEGRATION AUDIT SUMMARY REPORT
================================================================================
Total Tests: 50 | Passed: 50 | Failed: 0

ALL BACKEND API INTEGRATION AND SECURITY CONTRACTS PASSED (100%)
```

### 3. Execute End-to-End Simulation Script

```powershell
docker compose exec backend go run ./scripts/verify_e2e.go
```

### 4. Verify Frontend TypeScript & Production Build

```powershell
docker compose exec frontend npx tsc --noEmit
docker compose exec frontend npm run build
```

---

## 📚 Comprehensive REST API Reference

All requests expect `Content-Type: application/json` and `Authorization: Bearer <token>`.

### Admin Routes (`exam_admin` / `system_admin`)

```http
GET    /api/v1/question-banks                 # List Question Banks
POST   /api/v1/question-banks                 # Create Question Bank
GET    /api/v1/question-banks/{bankId}        # Get Question Bank & questions
POST   /api/v1/question-banks/{bankId}/q      # Author Question & Initial Version
POST   /api/v1/questions/{questionId}/publish # Publish Question Version (ACTIVE)
GET    /api/v1/exams                          # List Exams
POST   /api/v1/exams                          # Create Draft Exam
POST   /api/v1/exams/{examId}/questions       # Attach Published Question Version
POST   /api/v1/exams/{examId}/publish         # Publish Exam (DRAFT -> SCHEDULED)
POST   /api/v1/exams/{examId}/start           # Start Exam (SCHEDULED -> LIVE)
GET    /api/v1/candidates                     # List Registered Candidates
POST   /api/v1/candidates                    # Register Candidate Identity
POST   /api/v1/exams/{examId}/assignments     # Assign Candidate to Exam
```

### Candidate Routes (`candidate`)

```http
GET    /api/v1/candidate/assignments          # List My Assigned Exams
POST   /api/v1/exams/{examId}/attempts        # Start Exam Attempt
GET    /api/v1/attempts/{attemptId}/runtime   # Get Question Paper & Responses (Stripped)
POST   /api/v1/attempts/{attemptId}/sessions  # Bind Client Session
POST   /api/v1/sessions/{sessionId}/heartbeat # Heartbeat & Network Quality Ping
PUT    /api/v1/attempts/{id}/responses/{qvId}  # Direct Save Candidate Response
POST   /api/v1/attempts/{attemptId}/events:sync# Idempotent WAL Batch Sync
POST   /api/v1/attempts/{attemptId}/submit    # Submit & Freeze Attempt (Server Scoring)
```

---

## 📄 Submission Document Sitemap

For hackathon evaluators, detailed PDF submission documents are available in the repository:

- 📄 **Document 1**: Executive Overview & Problem Statement
- 📄 **Document 2**: System Architecture & Component Design
- 📄 **Document 3**: Domain Model & Database Schema Specification
- 📄 **Document 4**: API Contract & Event Ingestion Specification
- 📄 **Document 5**: Security, Trust & Immutability Architecture
- 📄 **Document 6**: Failure Recovery & Resilience Playbook
- 📄 **Document 7**: Test Plan & Integration Verification Suite
- 📄 [**Document 8 — Prototype Proof PDF**](screenshots/Document_8_Prototype_Demo_Proof_of_Concept.pdf): Verified screenshots, visual walkthroughs, and test evidence report.

---

## 📜 License & Acknowledgments

Developed specifically for the **MPOnline Digital Assessment & Examination Security Hackathon 2026**.

- **Built By**: SentinelExam Core Architecture & Engineering Team.
- **Standard Compliance**: Conforms strictly to MPOnline and NTA digital assessment security guidelines.

&copy; 2026 SentinelExam Platform. All rights reserved.
