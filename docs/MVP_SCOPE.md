# MVP Scope Specification

## 1. In Scope (Target Hackathon Deliverables)

1. **Foundation & Infrastructure**:
   - Docker Compose running PostgreSQL, NATS JetStream, Keycloak, Temporal, and Toxiproxy.
   - Modular Go backend with health checks, metrics, and structured logging.
   - React + TypeScript + Vite frontend skeleton.
2. **Resilient Candidate Engine**:
   - Offline-capable candidate UI with IndexedDB WAL.
   - Idempotent event batch sync to Go backend.
3. **Operations & Chaos Testing**:
   - Incident detection and compensation workflows.
   - Toxiproxy chaos injection simulating dropped packets and high latency.
4. **Audit & Trust**:
   - SHA-256 event chaining and basic Merkle tree checkpoints.

## 2. Out of Scope (Non-Goals for Phase 1)

* Full QTI 3 XML/JSON authoring IDE.
* Multi-region active-active database replication.
* Biometric eye-tracking computer vision ML pipelines.
* Complex billing, e-commerce, or enterprise LMS sync (LTI 1.3).
