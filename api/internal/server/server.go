package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"

	"github.com/Judeadeniji/zenv-sh/api/internal/audit"
	"github.com/Judeadeniji/zenv-sh/api/internal/config"
	v1 "github.com/Judeadeniji/zenv-sh/api/internal/server/v1"
)

// New creates the chi router with global middleware and versioned route groups.
func New(db *sql.DB, rdb *redis.Client, cfg *config.Config) *chi.Mux {
	// Audit log writer — LPUSH to Redis, background worker flushes to Postgres.
	al := audit.New(db, rdb)
	al.Start(context.Background())

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(requestLogger)

	// CORS — allow dashboard and auth server origins
	if cfg.CORSOrigins != "" {
		origins := strings.Split(cfg.CORSOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   origins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Health check
	r.Get("/health", healthHandler)

	// API versions
	r.Route("/v1", func(r chi.Router) {
		r.Use(al.Middleware) // Audit every /v1 request
		v1.Routes(r, db, rdb, cfg)
	})

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Debug("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start).String(),
		)
	})
}
