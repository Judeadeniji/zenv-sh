-- Store previous versions of secrets for rollback.
-- On every update, the old ciphertext/nonce is copied here before overwriting.

CREATE TABLE vault_item_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id     UUID NOT NULL REFERENCES vault_items(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    ciphertext  BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vault_item_versions_item ON vault_item_versions(item_id, version DESC);
