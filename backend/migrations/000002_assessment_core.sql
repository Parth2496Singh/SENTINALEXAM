-- =============================================================================
-- SentinelExam Migration 002: Assessment Core Schema
-- =============================================================================

-- -----------------------------------------------------------------------
-- ENUMS
-- -----------------------------------------------------------------------

CREATE TYPE exam_status AS ENUM (
    'DRAFT', 'SCHEDULED', 'LIVE', 'PAUSED', 'COMPLETED', 'ARCHIVED'
);

CREATE TYPE question_status AS ENUM (
    'DRAFT', 'ACTIVE', 'SUPERSEDED', 'RETIRED'
);

CREATE TYPE question_type AS ENUM (
    'SINGLE_CHOICE', 'MULTIPLE_CHOICE', 'TRUE_FALSE', 'TEXT', 'NUMERIC'
);

CREATE TYPE assignment_status AS ENUM (
    'ASSIGNED', 'REVOKED'
);

CREATE TYPE attempt_status AS ENUM (
    'NOT_STARTED', 'ACTIVE', 'SUBMITTED', 'EXPIRED', 'TERMINATED', 'FINALIZED'
);

CREATE TYPE session_status AS ENUM (
    'CREATED', 'CONNECTED', 'DISCONNECTED', 'RECONNECTING', 'TERMINATED'
);

CREATE TYPE response_status AS ENUM (
    'UNANSWERED', 'ANSWERED', 'SUBMITTED'
);

CREATE TYPE submission_status AS ENUM (
    'PENDING', 'SUBMITTED', 'FINALIZED'
);

CREATE TYPE candidate_status AS ENUM (
    'ACTIVE', 'SUSPENDED'
);

CREATE TYPE question_bank_status AS ENUM (
    'ACTIVE', 'ARCHIVED'
);

-- -----------------------------------------------------------------------
-- USERS
-- -----------------------------------------------------------------------

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_sub UUID NOT NULL UNIQUE,
    email       TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    roles       TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_keycloak_sub ON users(keycloak_sub);

-- -----------------------------------------------------------------------
-- CANDIDATES
-- -----------------------------------------------------------------------

CREATE TABLE candidates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status          candidate_status NOT NULL DEFAULT 'ACTIVE',
    full_name       TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_candidates_user_id ON candidates(user_id);
CREATE INDEX idx_candidates_status ON candidates(status);

-- -----------------------------------------------------------------------
-- QUESTION BANKS
-- -----------------------------------------------------------------------

CREATE TABLE question_banks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      question_bank_status NOT NULL DEFAULT 'ACTIVE',
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- -----------------------------------------------------------------------
-- QUESTIONS
-- -----------------------------------------------------------------------

CREATE TABLE questions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_bank_id    UUID NOT NULL REFERENCES question_banks(id) ON DELETE RESTRICT,
    status              question_status NOT NULL DEFAULT 'DRAFT',
    current_version_id  UUID,          -- Set after first publish; FK added below
    created_by          UUID REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_questions_bank_id ON questions(question_bank_id);
CREATE INDEX idx_questions_status ON questions(status);

-- -----------------------------------------------------------------------
-- QUESTION VERSIONS (immutable once ACTIVE)
-- -----------------------------------------------------------------------

CREATE TABLE question_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id     UUID NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
    version         INTEGER NOT NULL,
    type            question_type NOT NULL,
    content         JSONB NOT NULL,          -- Stimulus, options, etc.
    correct_answer  JSONB NOT NULL,          -- Scoring key
    marks           NUMERIC(10,4) NOT NULL DEFAULT 1,
    negative_marks  NUMERIC(10,4) NOT NULL DEFAULT 0,
    status          question_status NOT NULL DEFAULT 'DRAFT',
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_question_version UNIQUE (question_id, version),
    CONSTRAINT marks_positive CHECK (marks > 0),
    CONSTRAINT negative_marks_non_negative CHECK (negative_marks >= 0)
);

CREATE INDEX idx_qv_question_id ON question_versions(question_id);
CREATE INDEX idx_qv_status ON question_versions(status);

-- Back-fill the FK: questions.current_version_id → question_versions
ALTER TABLE questions ADD CONSTRAINT fk_questions_current_version
    FOREIGN KEY (current_version_id) REFERENCES question_versions(id)
    DEFERRABLE INITIALLY DEFERRED;

-- -----------------------------------------------------------------------
-- EXAMS
-- -----------------------------------------------------------------------

CREATE TABLE exams (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title               TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    status              exam_status NOT NULL DEFAULT 'DRAFT',
    duration_seconds    INTEGER NOT NULL,
    scheduled_start     TIMESTAMPTZ,
    scheduled_end       TIMESTAMPTZ,
    max_attempts        INTEGER NOT NULL DEFAULT 1,
    -- Navigation settings (JSONB for extensibility)
    navigation          JSONB NOT NULL DEFAULT '{"allow_back": true, "allow_skip": true}',
    -- Scoring settings (JSONB for extensibility)
    scoring             JSONB NOT NULL DEFAULT '{"negative_marking": false, "pass_percentage": null}',
    version             INTEGER NOT NULL DEFAULT 1,
    created_by          UUID REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT duration_positive CHECK (duration_seconds > 0),
    CONSTRAINT max_attempts_positive CHECK (max_attempts > 0),
    CONSTRAINT schedule_order CHECK (
        scheduled_end IS NULL OR scheduled_start IS NULL OR scheduled_end > scheduled_start
    )
);

CREATE INDEX idx_exams_status ON exams(status);

-- -----------------------------------------------------------------------
-- EXAM QUESTIONS (links exam → specific question VERSION)
-- -----------------------------------------------------------------------

CREATE TABLE exam_questions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id             UUID NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    question_version_id UUID NOT NULL REFERENCES question_versions(id) ON DELETE RESTRICT,
    position            INTEGER NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_exam_position UNIQUE (exam_id, position),
    CONSTRAINT unique_exam_question_version UNIQUE (exam_id, question_version_id),
    CONSTRAINT position_positive CHECK (position > 0)
);

CREATE INDEX idx_eq_exam_id ON exam_questions(exam_id);

-- -----------------------------------------------------------------------
-- EXAM ASSIGNMENTS
-- -----------------------------------------------------------------------

CREATE TABLE exam_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id         UUID NOT NULL REFERENCES exams(id) ON DELETE RESTRICT,
    candidate_id    UUID NOT NULL REFERENCES candidates(id) ON DELETE RESTRICT,
    status          assignment_status NOT NULL DEFAULT 'ASSIGNED',
    assigned_by     UUID REFERENCES users(id),
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,

    CONSTRAINT unique_exam_candidate UNIQUE (exam_id, candidate_id)
);

CREATE INDEX idx_ea_exam_id ON exam_assignments(exam_id);
CREATE INDEX idx_ea_candidate_id ON exam_assignments(candidate_id);
CREATE INDEX idx_ea_status ON exam_assignments(status);

-- -----------------------------------------------------------------------
-- EXAM ATTEMPTS
-- -----------------------------------------------------------------------

CREATE TABLE exam_attempts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id             UUID NOT NULL REFERENCES exams(id) ON DELETE RESTRICT,
    candidate_id        UUID NOT NULL REFERENCES candidates(id) ON DELETE RESTRICT,
    assignment_id       UUID NOT NULL REFERENCES exam_assignments(id) ON DELETE RESTRICT,
    status              attempt_status NOT NULL DEFAULT 'NOT_STARTED',
    started_at          TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ,   -- server-authoritative deadline
    submitted_at        TIMESTAMPTZ,
    finalized_at        TIMESTAMPTZ,
    session_epoch       INTEGER NOT NULL DEFAULT 0,
    current_question_id UUID REFERENCES question_versions(id),
    version             INTEGER NOT NULL DEFAULT 1,  -- optimistic locking
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ea_attempt_exam ON exam_attempts(exam_id);
CREATE INDEX idx_ea_attempt_candidate ON exam_attempts(candidate_id);
CREATE INDEX idx_ea_attempt_status ON exam_attempts(status);

-- -----------------------------------------------------------------------
-- ASSESSMENT SESSIONS
-- -----------------------------------------------------------------------

CREATE TABLE assessment_sessions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id           UUID NOT NULL REFERENCES exam_attempts(id) ON DELETE RESTRICT,
    client_instance_id   TEXT NOT NULL,  -- Browser fingerprint / tab ID
    session_epoch        INTEGER NOT NULL,
    status               session_status NOT NULL DEFAULT 'CREATED',
    connected_at         TIMESTAMPTZ,
    disconnected_at      TIMESTAMPTZ,
    terminated_at        TIMESTAMPTZ,
    last_heartbeat_at    TIMESTAMPTZ,
    last_client_sequence BIGINT NOT NULL DEFAULT 0,
    last_server_sequence BIGINT NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_attempt_epoch UNIQUE (attempt_id, session_epoch)
);

CREATE INDEX idx_sessions_attempt ON assessment_sessions(attempt_id);
CREATE INDEX idx_sessions_status ON assessment_sessions(status);

-- -----------------------------------------------------------------------
-- CANDIDATE RESPONSES
-- -----------------------------------------------------------------------

CREATE TABLE candidate_responses (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id           UUID NOT NULL REFERENCES exam_attempts(id) ON DELETE RESTRICT,
    question_version_id  UUID NOT NULL REFERENCES question_versions(id) ON DELETE RESTRICT,
    current_value        JSONB,            -- null = UNANSWERED
    status               response_status NOT NULL DEFAULT 'UNANSWERED',
    version              INTEGER NOT NULL DEFAULT 1,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_attempt_question UNIQUE (attempt_id, question_version_id)
);

CREATE INDEX idx_cr_attempt_id ON candidate_responses(attempt_id);

-- -----------------------------------------------------------------------
-- SUBMISSIONS
-- -----------------------------------------------------------------------

CREATE TABLE submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id      UUID NOT NULL UNIQUE REFERENCES exam_attempts(id) ON DELETE RESTRICT,
    status          submission_status NOT NULL DEFAULT 'PENDING',
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at    TIMESTAMPTZ,
    finalized_at    TIMESTAMPTZ,
    score           NUMERIC(10,4),
    max_score       NUMERIC(10,4),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_submissions_attempt ON submissions(attempt_id);

-- -----------------------------------------------------------------------
-- RECORD MIGRATION
-- -----------------------------------------------------------------------

INSERT INTO schema_version (version, description)
VALUES ('002_assessment_core', 'Assessment core schema: question banks, questions, versions, exams, assignments, attempts, sessions, responses, submissions')
ON CONFLICT (version) DO NOTHING;
