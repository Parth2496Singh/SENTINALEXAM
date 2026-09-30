-- =============================================================================
-- SentinelExam Migration 003: Resilience Events & Outbox
-- =============================================================================

-- -----------------------------------------------------------------------
-- CANDIDATE EVENTS (Sequenced Append-Only WAL Table)
-- -----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS candidate_events (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id             UUID NOT NULL UNIQUE,
    attempt_id           UUID NOT NULL REFERENCES exam_attempts(id) ON DELETE RESTRICT,
    session_id           UUID REFERENCES assessment_sessions(id) ON DELETE RESTRICT,
    session_epoch        INTEGER NOT NULL,
    sequence_number      BIGINT NOT NULL,
    event_type           TEXT NOT NULL,
    schema_version       TEXT NOT NULL DEFAULT '1.0',
    client_instance_id   TEXT NOT NULL,
    payload              JSONB NOT NULL DEFAULT '{}'::jsonb,
    client_timestamp     TIMESTAMPTZ NOT NULL,
    server_received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    prev_hash            TEXT,
    event_hash           TEXT,
    processing_status    TEXT NOT NULL DEFAULT 'COMMITTED',

    CONSTRAINT unique_attempt_sequence UNIQUE (attempt_id, sequence_number)
);

CREATE INDEX IF NOT EXISTS idx_candidate_events_attempt_seq ON candidate_events(attempt_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_candidate_events_event_id ON candidate_events(event_id);

-- -----------------------------------------------------------------------
-- OUTBOX EVENTS (Transactional Outbox for JetStream Relay)
-- -----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS outbox_events (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type           TEXT NOT NULL,
    stream_name          TEXT NOT NULL,
    subject              TEXT NOT NULL,
    payload              JSONB NOT NULL,
    status               TEXT NOT NULL DEFAULT 'PENDING',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(created_at) WHERE status = 'PENDING';

-- -----------------------------------------------------------------------
-- RECORD MIGRATION
-- -----------------------------------------------------------------------

INSERT INTO schema_version (version, description)
VALUES ('003_resilience_events', 'Candidate append-only events WAL and transactional outbox for JetStream relay')
ON CONFLICT (version) DO NOTHING;
