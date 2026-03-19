-- Link zEnv tables to Better Auth identity.
-- Better Auth owns the identity layer (user, session, account tables).
-- zEnv users table stores only crypto material, linked via better_auth_user_id.

ALTER TABLE users ADD COLUMN better_auth_user_id TEXT UNIQUE;

ALTER TABLE organizations ADD COLUMN better_auth_org_id TEXT UNIQUE;
