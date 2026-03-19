-- 001_init.down.sql
-- Reverse the initial schema. Drop in dependency order.

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS organization_members CASCADE;
DROP TABLE IF EXISTS service_tokens CASCADE;
DROP TABLE IF EXISTS vault_items CASCADE;
DROP TABLE IF EXISTS project_key_grants CASCADE;
DROP TABLE IF EXISTS project_vault_keys CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS linked_providers CASCADE;
DROP TABLE IF EXISTS users CASCADE;
