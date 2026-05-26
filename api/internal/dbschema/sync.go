// Package dbschema applies Drizzle SQL migrations from apps/identity/drizzle.
package dbschema

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// findDrizzleDir walks up from start until apps/identity/drizzle exists (works from repo root or any api/ subdir).
func findDrizzleDir(start string) (string, error) {
	dir := start
	for {
		candidate := filepath.Join(dir, "apps", "identity", "drizzle")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("drizzle directory not found in %s or its parents", start)
		}
		dir = parent
	}
}

// Sync applies sorted *.sql from apps/identity/drizzle. If the schema already exists (sessions table),
// only additive patches run — same rules as API integration tests on reused containers.
func Sync(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_lock(1234)"); err != nil {
		return fmt.Errorf("lock for migration: %w", err)
	}
	defer db.ExecContext(ctx, "SELECT pg_advisory_unlock(1234)")

	var sessionsExists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='sessions')`,
	).Scan(&sessionsExists)
	if err != nil {
		return fmt.Errorf("check schema existence: %w", err)
	}

	if sessionsExists {
		return ensureAdditiveMigrations(ctx, db)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	migrationDir, err := findDrizzleDir(cwd)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("read drizzle migrations at %s: %w", migrationDir, err)
	}

	var sqlFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, e.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(migrationDir, file))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}

	return nil
}

func ensureAdditiveMigrations(ctx context.Context, db *sql.DB) error {
	var hasMetadata bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'vault_items' AND column_name = 'metadata'
		)`,
	).Scan(&hasMetadata)
	if err != nil {
		return fmt.Errorf("check vault_items.metadata: %w", err)
	}
	if hasMetadata {
		return nil
	}
	var hasVaultItems bool
	if err = db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'vault_items'
		)`,
	).Scan(&hasVaultItems); err != nil {
		return fmt.Errorf("check vault_items table: %w", err)
	}
	if !hasVaultItems {
		return nil
	}
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE "vault_items" ADD COLUMN IF NOT EXISTS "metadata" jsonb DEFAULT '{}'::jsonb NOT NULL`,
	); err != nil {
		return fmt.Errorf("add vault_items.metadata: %w", err)
	}
	return nil
}
