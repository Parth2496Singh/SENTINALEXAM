# SentinelExam Documentation Index

Welcome to the engineering source of truth for **SentinelExam** — a production-oriented, resilient, and trustworthy online assessment platform.

## Architecture v1.0 Frozen Specification

The architecture of this platform is frozen for Architecture v1.0. All implementation components must strictly adhere to these design principles and authority models.

### Engineering Documents

1. [ARCHITECTURE.md](file:///X:/SentinelExam/docs/ARCHITECTURE.md) - System architecture, authority hierarchy, component topologies, and runtime data flow.
2. [DOMAIN_MODEL.md](file:///X:/SentinelExam/docs/DOMAIN_MODEL.md) - Core domain entities, relationships, boundaries, and aggregates.
3. [STATE_MACHINES.md](file:///X:/SentinelExam/docs/STATE_MACHINES.md) - Formal state transition specifications for exams, sessions, attempts, responses, incidents, and recovery.
4. [API_CONTRACT.md](file:///X:/SentinelExam/docs/API_CONTRACT.md) - REST API endpoints, schemas, headers, health checks, and error envelopes.
5. [DATABASE_SCHEMA.md](file:///X:/SentinelExam/docs/DATABASE_SCHEMA.md) - PostgreSQL schema design, indexing, audit invariants, and outbox table patterns.
6. [EVENT_CONTRACT.md](file:///X:/SentinelExam/docs/EVENT_CONTRACT.md) - NATS JetStream subject taxonomy, stream configurations, cloud event schemas, and replay semantics.
7. [SECURITY.md](file:///X:/SentinelExam/docs/SECURITY.md) - Keycloak IAM integration, JWT validation, RBAC, tenant isolation, and cryptographic proofs.
8. [FAILURE_RECOVERY.md](file:///X:/SentinelExam/docs/FAILURE_RECOVERY.md) - Offline resilience, candidate event WAL, partition recovery, and split-brain resolution.
9. [AUDIT.md](file:///X:/SentinelExam/docs/AUDIT.md) - SHA-256 hash chains, Merkle checkpointing, and non-repudiation verification.
10. [AI_ML.md](file:///X:/SentinelExam/docs/AI_ML.md) - Advisory/read-only AI boundaries, LiteLLM abstraction, proctor assistance, and non-authoritative invariants.
11. [OBSERVABILITY.md](file:///X:/SentinelExam/docs/OBSERVABILITY.md) - OpenTelemetry distributed tracing, Prometheus metric standards, structured logging, and health probing.
12. [MVP_SCOPE.md](file:///X:/SentinelExam/docs/MVP_SCOPE.md) - Scope boundaries, core milestones, and out-of-scope non-goals.
13. [DECISIONS.md](file:///X:/SentinelExam/docs/DECISIONS.md) - Architectural Decision Records (ADRs) explaining core technology choices and constraints.
