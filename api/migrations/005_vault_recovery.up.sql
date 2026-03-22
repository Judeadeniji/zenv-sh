-- 005_vault_recovery.up.sql
-- Adds vault recovery system: Recovery Kit, Trusted Contact Recovery, No Recovery opt-in.

-- ============================================================================
-- RECOVERY KIT — columns on users
-- ============================================================================

-- DEK wrapped with a random 256-bit recovery key (BIP39 mnemonic).
-- NULL until vault setup completes with recovery enabled.
ALTER TABLE users ADD COLUMN recovery_wrapped_dek BYTEA;

-- Enterprise opt-in: disable all recovery paths.
ALTER TABLE users ADD COLUMN recovery_disabled BOOLEAN NOT NULL DEFAULT false;

-- ============================================================================
-- TRUSTED CONTACTS — one trusted contact per user
-- ============================================================================

CREATE TABLE trusted_contacts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trusted_wrapped_dek BYTEA NOT NULL,      -- DEK wrapped with contact's X25519 public key (NaCl box)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, contact_user_id),
    CHECK (user_id != contact_user_id)
);

CREATE INDEX idx_trusted_contacts_user ON trusted_contacts(user_id);
CREATE INDEX idx_trusted_contacts_contact ON trusted_contacts(contact_user_id);

-- ============================================================================
-- RECOVERY REQUESTS — 72-hour waiting period for trusted contact recovery
-- ============================================================================

CREATE TABLE recovery_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'approved', 'cancelled', 'expired', 'completed')),
    recovery_public_key BYTEA,               -- ephemeral X25519 pubkey from recovering user
    recovery_payload    BYTEA,               -- DEK wrapped with ephemeral pubkey (set by contact on approve)
    requested_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    eligible_at         TIMESTAMPTZ NOT NULL, -- requested_at + 72h
    approved_at         TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ
);

-- Only one active recovery request per user at a time
CREATE UNIQUE INDEX idx_recovery_requests_active
    ON recovery_requests(user_id) WHERE status IN ('pending', 'approved');

CREATE INDEX idx_recovery_requests_contact
    ON recovery_requests(contact_user_id) WHERE status IN ('pending', 'approved');
