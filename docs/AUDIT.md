# Audit & Non-Repudiation Specification

## 1. SHA-256 Event Hash Chain

Every candidate action is cryptographically linked to the preceding action:

```
H_0 = SHA256(AttemptID + StartTimestamp)
H_n = SHA256(H_(n-1) + SeqID + EventType + PayloadJSON + ClientTimestamp)
```

Each event stored in the PostgreSQL database includes `prev_hash` and `event_hash`. Any modification or deletion of past events breaks the hash continuity.

## 2. Merkle Checkpoints

At regular intervals (e.g. every 5 minutes or upon section completion), the server:
1. Gathers all committed event hashes across the attempt or examination cohort.
2. Builds a binary Merkle Tree.
3. Computes the **Merkle Root Hash**.
4. Stores the checkpoint in `audit_checkpoints` and commits it to NATS stream `SENTINEL_AUDIT_EVENTS`.

## 3. Independent Verification

Any third-party auditor or grading verification committee can execute an offline verification script that:
1. Re-computes the hash chain from sequence `1` to `N`.
2. Validates inclusion proofs against the Merkle checkpoints.
3. Proves that answers were recorded precisely in the sequence and timestamps claimed.
