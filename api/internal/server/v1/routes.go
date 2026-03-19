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
	ta := middleware.NewTokenAuth(db)
	auth := handler.NewAuthHandler(db, sm)
	secrets := handler.NewSecretsHandler(db)
	tokens := handler.NewTokensHandler(db)
	projects := handler.NewProjectsHandler(db)
	orgs := handler.NewOrgsHandler(db)

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

	// Dashboard routes — require both identity + vault unlock (human access)
	r.Group(func(r chi.Router) {
		r.Use(sm.RequireSession)
		r.Use(sm.RequireVaultUnlocked)

		r.Route("/secrets", func(r chi.Router) {
			r.Post("/", secrets.Create)
			r.Post("/bulk", secrets.BulkFetch)
			r.Get("/", secrets.List)
			r.Get("/{nameHash}", secrets.Get)
			r.Get("/{nameHash}/versions", secrets.Versions)
			r.Post("/{nameHash}/rollback", secrets.Rollback)
			r.Put("/{nameHash}", secrets.Update)
			r.Delete("/{nameHash}", secrets.Delete)
		})

		r.Route("/tokens", func(r chi.Router) {
			r.Post("/", tokens.Create)
			r.Get("/", tokens.List)
			r.Delete("/{tokenID}", tokens.Revoke)
		})

		r.Route("/projects", func(r chi.Router) {
			r.Post("/", projects.Create)
			r.Get("/", projects.List)
			r.Get("/{projectID}", projects.Get)
		})

		r.Route("/orgs", func(r chi.Router) {
			r.Post("/", orgs.Create)
			r.Get("/", orgs.List)
			r.Get("/{orgID}", orgs.Get)
			r.Get("/{orgID}/members", orgs.ListMembers)
			r.Post("/{orgID}/members", orgs.AddMember)
			r.Delete("/{orgID}/members/{memberID}", orgs.RemoveMember)
		})
	})

	// SDK/CLI routes — authenticate via service token (machine access)
	r.Route("/sdk", func(r chi.Router) {
		r.Use(ta.Authenticate)

		// Project crypto — SDK needs salt + wrapped DEK to derive keys
		r.Get("/projects/{projectID}/crypto", projects.GetCrypto)

		// Read operations
		r.Post("/secrets/bulk", secrets.BulkFetch)
		r.Get("/secrets", secrets.List)
		r.Get("/secrets/{nameHash}", secrets.Get)
		r.Get("/secrets/{nameHash}/versions", secrets.Versions)

		// Write operations — require read_write permission
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireWrite)

			r.Post("/secrets", secrets.Create)
			r.Put("/secrets/{nameHash}", secrets.Update)
			r.Post("/secrets/{nameHash}/rollback", secrets.Rollback)
			r.Delete("/secrets/{nameHash}", secrets.Delete)
		})
	})
}
