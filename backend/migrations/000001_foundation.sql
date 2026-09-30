-- SentinelExam Foundation Migration 001
-- Minimal proof-of-concept migration to verify database connectivity,
-- schema versioning, and baseline health checks.

CREATE TABLE IF NOT EXISTS schema_version (
    version VARCHAR(64) PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Baseline system proof table
CREATE TABLE IF NOT EXISTS platform_health (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(128) NOT NULL,
    last_ping TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Record foundation migration
INSERT INTO schema_version (version, description)
VALUES ('001_initial_foundation', 'Initial foundation tables for platform health and schema validation')
ON CONFLICT (version) DO NOTHING;
