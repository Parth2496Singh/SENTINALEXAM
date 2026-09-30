# Architecture Decision Records (ADRs)

## ADR-001: Selection of PostgreSQL as Authoritative Storage
* **Status**: Accepted
* **Context**: Need absolute transactional guarantees, ACID properties, foreign key integrity, and battle-tested consistency for exam submissions and grading.
* **Decision**: Standardize on PostgreSQL. Prohibit NoSQL/MongoDB and uncoordinated in-memory caches.

## ADR-002: Modular Go Monolith Over Microservices
* **Status**: Accepted
* **Context**: Microservice sprawl introduces distributed failure modes, network latency, and deployment complexity during live exams.
* **Decision**: Implement a single modular Go monolith. Internal communication is via Go interfaces/in-process calls; external async events use NATS JetStream.

## ADR-003: NATS JetStream for Event Backbone
* **Status**: Accepted
* **Context**: Need durable event streams, replayability, and consumer group tracking without the heavy footprint and ZooKeeper/KRaft complexity of Kafka.
* **Decision**: Adopt NATS JetStream.

## ADR-004: Temporal for Long-Running Sagas
* **Status**: Accepted
* **Context**: Timers, batch grading, multi-step incident remediation, and exam lifecycle events require durable execution and saga compensation.
* **Decision**: Use Temporal. Prohibit putting candidate answer hot paths through Temporal.

## ADR-005: Client-Side IndexedDB WAL
* **Status**: Accepted
* **Context**: Candidates must not lose typed work or answers during abrupt client-side network disconnects or browser refreshes.
* **Decision**: Implement an append-only Write-Ahead Log in IndexedDB before transmitting events to the backend.

## ADR-006: Advisory AI / ML Invariant
* **Status**: Accepted
* **Context**: AI models can hallucinate or produce false positives in proctoring and grading.
* **Decision**: AI is strictly advisory and read-only. AI outages must never block the assessment runtime or fail readiness probes.

## ADR-007: Windows + Docker Desktop + PowerShell as Single Execution Environment
* **Status**: Accepted
* **Context**: Developers need a clean, reproducible setup without installing various language runtimes and infrastructure tools natively on the Windows host. Linux-only scripts (bash, export, sudo) create cross-platform friction.
* **Decision**: SentinelExam uses Windows + Docker Desktop + PowerShell as the developer environment. Docker is the single execution environment for application tooling and infrastructure. Native host installation of Go, Node/npm, PostgreSQL, NATS, Keycloak, Temporal, or Toxiproxy is not required. All developer workflows (build, test, run, tidy) execute inside containers via Docker Compose.

## ADR-008: Untrusted Client Trust Model Without Mandatory Browser Key Management
* **Status**: Accepted
* **Context**: Browser environments cannot securely hold non-extractable private keys across arbitrary reboots, guest profiles, and lock-down browsers without complex PKI infrastructure.
* **Decision**: Client input is untrusted. The backend validates and sequences events, then generates canonical SHA-256 hash chains and Merkle checkpoints on the server. Browser-side keypair signing is strictly optional/conditional for future phases and is not required for Architecture v1.0.

