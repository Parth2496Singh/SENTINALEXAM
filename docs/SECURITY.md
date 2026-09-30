# Security & Identity Specification

## 1. Authentication via Keycloak

SentinelExam uses Keycloak as the centralized OpenID Connect (OIDC) identity provider.

* **JWT Verification**: The Go backend validates tokens using Keycloak's public JWKS (`/realms/{realm}/protocol/openid-connect/certs`).
* **Claims Validation**: Verifies `iss` (Issuer), `aud` (Audience), `exp` (Expiration), and `sub` (Subject).

## 2. Role-Based Access Control (RBAC)

Roles defined in the `sentinel` realm:
1. `candidate`: Access restricted to assigned exams, active sessions, and candidate event ingestion.
2. `proctor`: Access to real-time proctor monitoring streams, incident reports, and candidate session health.
3. `exam_admin`: Access to question bank authoring, exam scheduling, and configuration.
4. `system_admin`: Access to operations, recovery workflows, and audit verification.

## 3. Trust Model & Audit Cryptography

The frozen Architecture v1.0 trust pipeline operates without browser-side cryptographic key management:

```
Browser / Client (Untrusted Event Source)
  │
  ▼
Server Ingestion Gateway (Validation, Deduplication & Sequencing)
  │
  ▼
Canonical Server Event (PostgreSQL Outbox & Commit)
  │
  ▼
SHA-256 Event Hash Chain (prev_hash -> event_hash)
  │
  ▼
Merkle Checkpoints (Periodic Root Hashes for Cohort Verification)
```

* **Untrusted Client Boundary**: Browser/client inputs are strictly untrusted. The backend validates attempt state, timer validity, token claims, and monotonic event sequencing.
* **Server-Side Hash-Chained Events**: Canonical candidate events form an immutable linear SHA-256 hash chain stored in PostgreSQL.
* **Merkle Root Checkpoints**: Checkpoints are periodically computed server-side and committed to the audit stream for non-repudiation.
* **Client Keypair Signing (Optional / Future Only)**: Browser-side asymmetric keypair signing is strictly an optional/conditional future enhancement and is NOT part of the mandatory Architecture v1.0 foundation.

