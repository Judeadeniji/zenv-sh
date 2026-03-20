package testutil

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Judeadeniji/zenv-sh/api/internal/config"
	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/server"
)

// TestServer holds the httptest server and supporting resources.
type TestServer struct {
	Server *httptest.Server
	DB     *sql.DB
	Redis  *redis.Client
	URL    string
}

// SetupServer starts containers, runs migrations, and creates an httptest.Server
// with the full production router. For use in individual test functions.
func SetupServer(t *testing.T) *TestServer {
	t.Helper()

	db := SetupDB(t)
	rdb := SetupRedis(t)

	cfg := &config.Config{
		CORSOrigins: "*",
	}
	router := server.New(db, rdb, cfg)
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close() })

	return &TestServer{
		Server: srv,
		DB:     db,
		Redis:  rdb,
		URL:    srv.URL,
	}
}

// SetupServerForMain is like SetupServer but for TestMain.
// Returns a cleanup function that must be called after m.Run().
func SetupServerForMain() (*TestServer, func()) {
	db, dbCleanup := SetupDBForMain()
	rdb, redisCleanup := SetupRedisForMain()

	cfg := &config.Config{
		CORSOrigins: "*",
	}
	router := server.New(db, rdb, cfg)
	srv := httptest.NewServer(router)

	ts := &TestServer{
		Server: srv,
		DB:     db,
		Redis:  rdb,
		URL:    srv.URL,
	}

	cleanup := func() {
		srv.Close()
		dbCleanup()
		redisCleanup()
	}

	return ts, cleanup
}

// AuthRequest creates an *http.Request with the identity session cookie set.
func AuthRequest(method, url string, body *http.Request) *http.Request {
	return body
}

// SessionCookie returns a cookie for authenticating API requests.
func SessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:  middleware.IdentitySessionCookie,
		Value: token,
	}
}

// Ctx returns a background context.
func Ctx() context.Context {
	return context.Background()
}

// FutureTime returns a time 24 hours from now.
func FutureTime() time.Time {
	return time.Now().Add(24 * time.Hour)
}
