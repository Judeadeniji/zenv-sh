ALTER TABLE users RENAME COLUMN identity_id TO better_auth_user_id;
ALTER TABLE organizations RENAME COLUMN identity_org_id TO better_auth_org_id;
