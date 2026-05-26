-- Custom SQL migration file, put your code below! --
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID DEFAULT gen_random_uuid(),
    project_id  UUID,
    user_id     UUID,
    token_id    UUID,
    action      TEXT NOT NULL,
    secret_hash BYTEA,
    ip          INET,
    user_agent  TEXT,
    result      TEXT NOT NULL DEFAULT 'success'
                  CHECK (result IN ('success', 'denied', 'error')),
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX IF NOT EXISTS idx_audit_logs_project_time ON audit_logs(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_time    ON audit_logs(user_id, created_at);
