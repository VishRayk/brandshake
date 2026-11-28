-- Background jobs, processed by workers inside the API process.
-- Workers claim jobs with FOR UPDATE SKIP LOCKED, so any number of
-- workers (or API instances) can run without double-processing.
CREATE TABLE jobs (
    id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind         text        NOT NULL CHECK (length(kind) BETWEEN 1 AND 60),
    payload      jsonb       NOT NULL DEFAULT '{}',
    status       text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'done', 'failed')),
    run_at       timestamptz NOT NULL DEFAULT now(),
    attempts     integer     NOT NULL DEFAULT 0,
    max_attempts integer     NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 50),
    last_error   text        NOT NULL DEFAULT '',
    locked_at    timestamptz,
    -- Enqueueing the same dedupe key twice is a no-op (e.g. one reminder per invoice per week).
    dedupe_key   text        UNIQUE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX jobs_pending_idx ON jobs (run_at) WHERE status = 'pending';
CREATE INDEX jobs_running_idx ON jobs (locked_at) WHERE status = 'running';

-- Every email Brandshake sends (or, in development, would send).
CREATE TABLE emails (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    to_address  text        NOT NULL,
    subject     text        NOT NULL,
    body_text   text        NOT NULL,
    kind        text        NOT NULL,
    provider    text        NOT NULL,
    provider_id text        NOT NULL DEFAULT '',
    job_id      bigint      REFERENCES jobs (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX emails_to_idx ON emails (to_address, created_at DESC);
