-- 001_init.up.sql
-- Initial schema for zEnv — zero-knowledge secret manager.
-- The server stores only ciphertext, wrapped keys, and hashes. Never plaintext secrets.

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- USERS
-- ============================================================================

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,        -- canonical identity anchor for account linking
    auth_key_hash   BYTEA NOT NULL,              -- Argon2 hash of Auth Key (Vault Key verification)
    vault_key_type  TEXT NOT NULL DEFAULT 'passphrase' CHECK (vault_key_type IN ('pin', 'passphrase')),
    salt            BYTEA NOT NULL,              -- sent to client to re-derive KEK
    wrapped_dek     BYTEA NOT NULL,              -- DEK encrypted with KEK — server cannot unwrap
    public_key      BYTEA NOT NULL,              -- X25519 public key — for sharing
    wrapped_private_key BYTEA NOT NULL,          -- private key encrypted with DEK — server cannot read
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- LINKED PROVIDERS (account linking — multiple OAuth providers per user)
-- ============================================================================

CREATE TABLE linked_providers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL CHECK (provider IN ('github', 'google', 'okta', 'azure_ad', 'magic_link')),
    provider_user_id  TEXT NOT NULL,              -- the user's ID within that provider
    linked_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_linked_providers_user_id ON linked_providers(user_id);

-- ============================================================================
-- ORGANIZATIONS
-- ============================================================================

CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    owner_id    UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- PROJECTS
-- ============================================================================

CREATE TABLE projects (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (organization_id, name)
);

-- ============================================================================
-- PROJECT VAULT KEYS (machine access — zero-knowledge for SDK/CI)
-- ============================================================================

CREATE TABLE project_vault_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE UNIQUE,
    project_salt        BYTEA NOT NULL,          -- used by client for Argon2id
    wrapped_project_dek BYTEA NOT NULL,          -- Project DEK wrapped with Project KEK
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- PROJECT KEY GRANTS (share Project Vault Key with team members)
-- ============================================================================

CREATE TABLE project_key_grants (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id                  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id                     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wrapped_project_vault_key   BYTEA NOT NULL,  -- Project Vault Key wrapped with user's public key
    granted_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (project_id, user_id)
);

-- ============================================================================
-- VAULT ITEMS (each secret is its own encrypted row)
-- ============================================================================

CREATE TABLE vault_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment TEXT NOT NULL CHECK (environment IN ('development', 'staging', 'production')),
    name_hash   BYTEA NOT NULL,                  -- HMAC-SHA256 of secret name — indexed, reveals nothing
    ciphertext  BYTEA NOT NULL,                  -- AES-256-GCM encrypted item JSON (name + value + metadata)
    nonce       BYTEA NOT NULL,                  -- 96-bit random nonce, unique per item per write
    version     INTEGER NOT NULL DEFAULT 1,      -- increments on every write
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_vault_items_lookup
    ON vault_items(project_id, environment, name_hash);

CREATE INDEX idx_vault_items_project_env
    ON vault_items(project_id, environment);

-- ============================================================================
-- SERVICE TOKENS (machine credentials — hashed before storage)
-- ============================================================================

CREATE TABLE service_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,                -- human-readable label
    token_hash      BYTEA NOT NULL UNIQUE,       -- SHA-256 hash of the token — never store plaintext
    environment     TEXT NOT NULL CHECK (environment IN ('development', 'staging', 'production')),
    permission      TEXT NOT NULL DEFAULT 'read' CHECK (permission IN ('read', 'read_write')),
    created_by      UUID REFERENCES users(id),   -- who created it (audit trail)
    expires_at      TIMESTAMPTZ,                 -- optional auto-revoke date
    revoked_at      TIMESTAMPTZ,                 -- null = active, set = revoked
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_service_tokens_project ON service_tokens(project_id);
CREATE INDEX idx_service_tokens_hash ON service_tokens(token_hash) WHERE revoked_at IS NULL;

-- ============================================================================
-- ORGANIZATION MEMBERS
-- ============================================================================

CREATE TABLE organization_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL DEFAULT 'dev' CHECK (role IN ('admin', 'senior_dev', 'dev', 'contractor', 'ci_bot')),
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (organization_id, user_id)
);

-- ============================================================================
-- AUDIT LOGS (partitioned by month for fast drops and range queries)
-- ============================================================================

CREATE TABLE audit_logs (
    id          UUID DEFAULT gen_random_uuid(),
    project_id  UUID,
    user_id     UUID,
    token_id    UUID,
    action      TEXT NOT NULL,                   -- 'secret.read' | 'secret.write' | 'secret.delete' | 'token.create' | 'token.revoke' | ...
    secret_hash BYTEA,                           -- HMAC of secret name — never plaintext
    ip          INET,
    user_agent  TEXT,
    result      TEXT NOT NULL DEFAULT 'success' CHECK (result IN ('success', 'denied', 'error')),
    metadata    JSONB,                           -- flexible extra context per action type
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

-- Create partitions for the next 12 months.
-- In production, a cron job or the API creates future partitions automatically.
DO $$
DECLARE
    start_date DATE := date_trunc('month', CURRENT_DATE);
    end_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..11 LOOP
        end_date := start_date + INTERVAL '1 month';
        partition_name := 'audit_logs_' || to_char(start_date, 'YYYY_MM');

        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );

        start_date := end_date;
    END LOOP;
END $$;

CREATE INDEX idx_audit_logs_project_time ON audit_logs(project_id, created_at);
CREATE INDEX idx_audit_logs_user_time ON audit_logs(user_id, created_at);
