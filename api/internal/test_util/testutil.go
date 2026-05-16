package test_util

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Judeadeniji/zenv-sh/api/internal/dbschema"
)

const (
	pgContainerName    = "zenv-test-postgres"
	redisContainerName = "zenv-test-redis"
)

func init() {
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
}

func SetupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := setupDB()
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

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
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second),
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

	// Retry ping to ensure the process inside the container is actually accepting queries
	var pingErr error
	for i := 0; i < 10; i++ {
		if pingErr = db.PingContext(ctx); pingErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if pingErr != nil {
		return nil, fmt.Errorf("ping db: %w", pingErr)
	}

	if err := dbschema.Sync(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func SetupRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb, err := setupRedis()
	if err != nil {
		t.Fatalf("setup redis: %v", err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

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
