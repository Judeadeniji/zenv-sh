-- 005_vault_recovery.down.sql

DROP TABLE IF EXISTS recovery_requests;
DROP TABLE IF EXISTS trusted_contacts;

ALTER TABLE users DROP COLUMN IF EXISTS recovery_wrapped_dek;
ALTER TABLE users DROP COLUMN IF EXISTS recovery_disabled;
