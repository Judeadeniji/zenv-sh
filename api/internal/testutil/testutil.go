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
// Consolidated from the auth server's drizzle migrations.
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
    updated_at TIMESTAMP NOT NULL
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

// SetupDB starts a Postgres container, runs all migrations, and returns a *sql.DB.
// The container is terminated when the test completes.
func SetupDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	pg, err := tcPostgres.Run(ctx,
		"postgres:17-alpine",
		tcPostgres.WithDatabase("zenv_test"),
		tcPostgres.WithUsername("test"),
		tcPostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { pg.Terminate(ctx) })

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get postgres connection string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create identity tables first (BA tables that the API reads).
	if _, err := db.ExecContext(ctx, identityTablesDDL); err != nil {
		t.Fatalf("create identity tables: %v", err)
	}

	// Run zEnv migrations.
	runMigrations(t, ctx, db)

	return db
}

// SetupDBForMain is like SetupDB but for use in TestMain where *testing.T is unavailable.
func SetupDBForMain() (*sql.DB, func()) {
	ctx := context.Background()

	pg, err := tcPostgres.Run(ctx,
		"postgres:17-alpine",
		tcPostgres.WithDatabase("zenv_test"),
		tcPostgres.WithUsername("test"),
		tcPostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("start postgres container: %v", err))
	}

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(fmt.Sprintf("get postgres connection string: %v", err))
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		panic(fmt.Sprintf("open postgres: %v", err))
	}

	if _, err := db.ExecContext(ctx, identityTablesDDL); err != nil {
		panic(fmt.Sprintf("create identity tables: %v", err))
	}

	runMigrationsForMain(ctx, db)

	cleanup := func() {
		db.Close()
		pg.Terminate(ctx)
	}
	return db, cleanup
}

// SetupRedis starts a Redis container and returns a *redis.Client.
func SetupRedis(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	rds, err := tcRedis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() { rds.Terminate(ctx) })

	endpoint, err := rds.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("get redis endpoint: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: endpoint})
	t.Cleanup(func() { rdb.Close() })

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}

	return rdb
}

// SetupRedisForMain is like SetupRedis but for TestMain.
func SetupRedisForMain() (*redis.Client, func()) {
	ctx := context.Background()

	rds, err := tcRedis.Run(ctx, "redis:7-alpine")
	if err != nil {
		panic(fmt.Sprintf("start redis container: %v", err))
	}

	endpoint, err := rds.Endpoint(ctx, "")
	if err != nil {
		panic(fmt.Sprintf("get redis endpoint: %v", err))
	}

	rdb := redis.NewClient(&redis.Options{Addr: endpoint})

	cleanup := func() {
		rdb.Close()
		rds.Terminate(ctx)
	}
	return rdb, cleanup
}

// migrationsDir resolves the path to api/migrations/ relative to this file.
func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}

func runMigrations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	dir := migrationsDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
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
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			t.Fatalf("run migration %s: %v", name, err)
		}
	}
}

func runMigrationsForMain(ctx context.Context, db *sql.DB) {
	dir := migrationsDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("read migrations dir: %v", err))
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
			panic(fmt.Sprintf("read migration %s: %v", name, err))
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			panic(fmt.Sprintf("run migration %s: %v", name, err))
		}
	}
}
