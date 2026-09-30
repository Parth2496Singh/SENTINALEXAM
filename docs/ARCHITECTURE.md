# Architecture Specification (v1.0 Frozen)

## 1. System Overview

**SentinelExam** is a high-integrity, resilient online assessment engine designed to survive severe network partitioning, client disconnects, and infrastructure failure without compromising assessment validity or candidate work.

```
+-----------------------------------------------------------------------------------+
|                                Client Tier (React / TS)                           |
|  +-----------------------------------------------------------------------------+  |
|  | [Candidate UI / Proctor Dashboard / Ops UI]                                 |  |
|  | - Client State Engine & Sync Worker                                         |  |
|  | - IndexedDB Event Write-Ahead Log (WAL) & Merkle Candidate Leaf             |  |
|  +-----------------------------------------------------------------------------+  |
+----------------------------------------+------------------------------------------+
                                         | HTTPS / REST / SSE
                                         v
+-----------------------------------------------------------------------------------+
|                           Sentinel Modular Go Monolith                            |
|  +-----------------------------------------------------------------------------+  |
|  |  HTTP Router / Middleware (Auth, CORS, Correlation, Telemetry)              |  |
|  +-----------------------------------------------------------------------------+  |
|  |  Modules:                                                                   |  |
|  |  [Assessment]    [Execution]    [Operations]   [Trust & Audit]              |  |
|  |  [Auth/IAM]      [Intelligence] [Observability][Platform/Persistence]       |  |
|  +-----------------------+-------------------------------+---------------------+  |
+--------------------------|-------------------------------|------------------------+
                           |                               |
          +----------------+--------------+                |
          |                               |                |
          v                               v                v
+-------------------+           +-------------------+   +--------------------+
|    PostgreSQL     |           |   NATS JetStream  |   |   Temporal Engine  |
| (Authoritative    |           | (Durable Event    |   | (Long-Running Sagas|
|  Transactional    |           |  Distribution &   |   |  & Recovery        |
|  State Engine)    |           |  Replay Backbone) |   |  Workflows)        |
+-------------------+           +-------------------+   +--------------------+
```

## 2. Mandatory Authority Model

The authority hierarchy is strictly linear and inviolable:

1. **PostgreSQL**: Authoritative transactional truth. All state changes that affect grading, attempt states, timer validity, or incident resolution are finalized here.
2. **NATS JetStream**: Durable, ordered event distribution and replay backbone. It is NOT the source of truth, but an immutable stream buffer between clients, services, and asynchronous consumers.
3. **Temporal**: Orchestration engine for long-running workflows (e.g., attempt expiry timers, bulk recovery operations, session lifecycle escalations, post-exam grading pipelines). Temporal orchestrates tasks; it is NOT the candidate answer hot path.
4. **IndexedDB**: Candidate-side durable evidence storage. Provides zero-loss write-ahead logging (WAL) when offline, but client data is never authoritative until verified and committed by the backend.
5. **AI / ML**: Purely advisory and read-only. AI outputs (anomaly flags, proctoring alerts) can never automatically mutate attempt state or invalidate a candidate.

**Rule**: *Browser/client input is NEVER authoritative.*

## 3. Technology Stack

* **Frontend**: React 18+, TypeScript, Vite, Tailwind CSS, shadcn/ui.
* **Backend**: Go (Modular Monolith) with standard library `net/http` or lightweight routing, `pgx` for PostgreSQL, `nats.go` for JetStream.
* **Identity**: Keycloak (OIDC / OAuth2 with JWT validation).
* **Database**: PostgreSQL 16+ with transaction isolation and transactional outbox.
* **Messaging**: NATS JetStream 2.10+.
* **Workflow Engine**: Temporal 1.24+.
* **Chaos Testing**: Toxiproxy.
* **Telemetry**: OpenTelemetry (Traces & Metrics) + Prometheus metrics endpoint (`/metrics`).

## 4. Anti-Patterns and Explicit Prohibitions

1. **No Redis**: In-memory caching without transactional guarantees introduces cache incoherency. PostgreSQL with proper indexing and connection pooling handles all hot operational data.
2. **No Kafka**: Kafka adds massive operational overhead. NATS JetStream satisfies all event streaming, replay, and durability needs with minimal footprint.
3. **No MongoDB / NoSQL**: Assessment relational integrity, foreign key consistency, and ACID guarantees are strictly non-negotiable.
4. **No Elasticsearch / OpenSearch**: Search indexing will utilize PostgreSQL full-text search (`tsvector`) and JSONB indexing.
5. **No Microservice Zoo**: The backend is structured as a single cohesive modular Go codebase to eliminate network hop latency, distributed transaction complexity, and multi-service deployment failure modes during assessments.

## 5. Development Environment Execution Model

SentinelExam standardizes on **Windows + Docker Desktop + PowerShell** as its single developer execution environment.

* **Single Execution Target**: Docker is the single execution environment for application tooling and infrastructure.
* **No Host Runtime Dependencies**: Native host installation of Go, Node/npm, PostgreSQL, NATS, Keycloak, Temporal, or Toxiproxy is not required.
* **No Linux-only Host Dependencies**: Host interactions are executed strictly via PowerShell Docker commands (`docker compose up -d`, `docker compose exec`, `docker compose run --rm`). Linux-only host scripts (`bash`, `sudo`, `export`, `chmod`) are prohibited on the Windows host.
* **Live Development**: Source code directories are bind-mounted (`./backend:/app`, `./frontend:/app`) into development containers for immediate live-reloading.

