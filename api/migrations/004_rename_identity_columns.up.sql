-- Rename Better Auth linking columns to generic identity names.
-- The identity provider is an implementation detail — not exposed in the schema.

ALTER TABLE users RENAME COLUMN better_auth_user_id TO identity_id;
ALTER TABLE organizations RENAME COLUMN better_auth_org_id TO identity_org_id;
