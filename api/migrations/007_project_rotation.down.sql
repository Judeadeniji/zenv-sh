DROP TABLE IF EXISTS vault_item_rotations;
DROP TABLE IF EXISTS project_rotations;
ALTER TABLE vault_items DROP COLUMN IF EXISTS dek_version;
ALTER TABLE project_vault_keys DROP COLUMN IF EXISTS dek_version;
