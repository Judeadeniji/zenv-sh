package v1

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/Judeadeniji/zenv-sh/api/internal/handler"
	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
)

// Routes mounts all /v1 endpoints onto the given router.
func Routes(r chi.Router, db *sql.DB, rdb *redis.Client) {
	sm := middleware.NewSessionManager(rdb)
	auth := handler.NewAuthHandler(db, sm)
	secrets := handler.NewSecretsHandler(db)

	// Public — no session required
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", auth.Signup)
		r.Post("/login", auth.DevLogin) // dev only — replaced by OAuth later
	})

	// Require identity layer (session exists)
	r.Group(func(r chi.Router) {
		r.Use(sm.RequireSession)

		r.Post("/auth/unlock", auth.Unlock)
		r.Post("/auth/logout", auth.Logout)
	})

	// Require both identity + vault unlock
	r.Group(func(r chi.Router) {
		r.Use(sm.RequireSession)
		r.Use(sm.RequireVaultUnlocked)

		r.Route("/secrets", func(r chi.Router) {
			r.Post("/", secrets.Create)
			r.Post("/bulk", secrets.BulkFetch)
			r.Get("/", secrets.List)
			r.Get("/{nameHash}", secrets.Get)
			r.Put("/{nameHash}", secrets.Update)
			r.Delete("/{nameHash}", secrets.Delete)
		})

		r.Route("/tokens", func(r chi.Router) {
			// TODO: create, revoke
		})

		r.Route("/projects", func(r chi.Router) {
			// TODO: CRUD
		})
	})
}
