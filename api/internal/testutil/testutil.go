package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// identityTablesDDL creates the identity provider tables that the Go API reads.
// Consolidated from the auth server's drizzle migrations. Uses IF NOT EXISTS
// so it's safe to run on a reused container.
const identityTablesDDL = `
CREATE TABLE IF NOT EXISTS "user" (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    email_verified BOOLEAN DEFAULT false NOT NULL,
    image TEXT,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    updated_at TIMESTAMP DEFAULT now() NOT NULL,
    role TEXT,
    banned BOOLEAN DEFAULT false,
    ban_reason TEXT,
    ban_expires TIMESTAMP,
    two_factor_enabled BOOLEAN DEFAULT false
);

CREATE TABLE IF NOT EXISTS "session" (
    id TEXT PRIMARY KEY NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    token TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    updated_at TIMESTAMP DEFAULT now() NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    impersonated_by TEXT,
    active_organization_id TEXT
);

CREATE TABLE IF NOT EXISTS "account" (
    id TEXT PRIMARY KEY NOT NULL,
    account_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    access_token TEXT,
    refresh_token TEXT,
    id_token TEXT,
    access_token_expires_at TIMESTAMP,
    refresh_token_expires_at TIMESTAMP,
    scope TEXT,
    password TEXT,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "verification" (
    id TEXT PRIMARY KEY NOT NULL,
    identifier TEXT NOT NULL,
    value TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    updated_at TIMESTAMP DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS "two_factor" (
    id TEXT PRIMARY KEY NOT NULL,
    secret TEXT NOT NULL,
    backup_codes TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "organization" (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    logo TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    metadata TEXT
);

CREATE TABLE IF NOT EXISTS "member" (
    id TEXT PRIMARY KEY NOT NULL,
    organization_id TEXT NOT NULL REFERENCES "organization"(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "invitation" (
    id TEXT PRIMARY KEY NOT NULL,
    organization_id TEXT NOT NULL REFERENCES "organization"(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    inviter_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE
);
`

const (
	pgContainerName    = "zenv-test-postgres"
	redisContainerName = "zenv-test-redis"
)

func init() {
	// Disable Ryuk so reused containers persist after the test process exits.
	// Containers must be stopped manually: docker rm -f zenv-test-postgres zenv-test-redis
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
}

// SetupDB starts (or reuses) a Postgres container, runs migrations, returns *sql.DB.
func SetupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := setupDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// SetupDBForMain is like SetupDB but for TestMain.
func SetupDBForMain() (*sql.DB, func()) {
	db, err := setupDB()
	if err != nil {
		panic(fmt.Sprintf("setup db: %v", err))
	}
	return db, func() { db.Close() }
}

func setupDB() (*sql.DB, error) {
	ctx := context.Background()

	pg, err := tcPostgres.Run(ctx,
		"postgres:17-alpine",
		tcPostgres.WithDatabase("zenv_test"),
		tcPostgres.WithUsername("test"),
		tcPostgres.WithPassword("test"),
		testcontainers.WithReuseByName(pgContainerName),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres: %w", err)
	}

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("connection string: %w", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Identity tables (idempotent — IF NOT EXISTS).
	if _, err := db.ExecContext(ctx, identityTablesDDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("identity tables: %w", err)
	}

	// zEnv migrations (idempotent — check if already applied).
	if err := runMigrationsIdempotent(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

// SetupRedis starts (or reuses) a Redis container.
func SetupRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb, err := setupRedis()
	if err != nil {
		t.Fatalf("setup redis: %v", err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

// SetupRedisForMain is like SetupRedis but for TestMain.
func SetupRedisForMain() (*redis.Client, func()) {
	rdb, err := setupRedis()
	if err != nil {
		panic(fmt.Sprintf("setup redis: %v", err))
	}
	return rdb, func() { rdb.Close() }
}

func setupRedis() (*redis.Client, error) {
	ctx := context.Background()

	rds, err := tcRedis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithReuseByName(redisContainerName),
	)
	if err != nil {
		return nil, fmt.Errorf("start redis: %w", err)
	}

	endpoint, err := rds.Endpoint(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("redis endpoint: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: endpoint})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return rdb, nil
}

// migrationsDir resolves the path to api/migrations/ relative to this file.
func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}

// runMigrationsIdempotent runs migrations only if not already applied.
// Checks for the `users` table as a sentinel.
func runMigrationsIdempotent(ctx context.Context, db *sql.DB) error {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='users')`,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check migrations: %w", err)
	}
	if exists {
		return nil // already migrated
	}

	dir := migrationsDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var upFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			return fmt.Errorf("run %s: %w", name, err)
		}
	}

	return nil
}
