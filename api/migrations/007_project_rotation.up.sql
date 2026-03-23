-- Track which DEK version encrypted each item
ALTER TABLE project_vault_keys ADD COLUMN dek_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE vault_items ADD COLUMN dek_version INTEGER NOT NULL DEFAULT 1;

-- In-progress rotation metadata
CREATE TABLE project_rotations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    rotation_id   UUID NOT NULL UNIQUE,
    status        TEXT NOT NULL DEFAULT 'staging'
                  CHECK (status IN ('staging', 'committing', 'complete', 'failed')),
    total_items   INTEGER NOT NULL,
    staged_items  INTEGER NOT NULL DEFAULT 0,
    initiated_by  UUID NOT NULL REFERENCES users(id),
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ
);

-- Staging table for new ciphertexts (Phase 1 target)
CREATE TABLE vault_item_rotations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    rotation_id     UUID NOT NULL REFERENCES project_rotations(rotation_id) ON DELETE CASCADE,
    vault_item_id   UUID NOT NULL REFERENCES vault_items(id) ON DELETE CASCADE,
    new_ciphertext  BYTEA NOT NULL,
    new_nonce       BYTEA NOT NULL,
    staged_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vault_item_rotations_rotation ON vault_item_rotations(rotation_id);
CREATE INDEX idx_project_rotations_project ON project_rotations(project_id);
