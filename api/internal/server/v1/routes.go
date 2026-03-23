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
	identity := middleware.NewIdentitySession(db, rdb)
	ta := middleware.NewTokenAuth(db)
	auth := handler.NewAuthHandler(db, identity)
	recovery := handler.NewRecoveryHandler(db, identity)
	secrets := handler.NewSecretsHandler(db)
	tokens := handler.NewTokensHandler(db)
	projects := handler.NewProjectsHandler(db)
	orgs := handler.NewOrgsHandler(db)
	audit := handler.NewAuditHandler(db)

	// Identity session routes — verified via session cookie or Bearer token
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireSession)

		r.Get("/auth/me", auth.Me)
		r.Post("/auth/setup-vault", auth.SetupVault)
		r.Post("/auth/unlock", auth.Unlock)
		r.Put("/auth/change-vault-key", auth.ChangeVaultKey)

		// Recovery — session only (no vault unlock required)
		r.Get("/auth/recovery/status", recovery.Status)
		r.Get("/auth/recovery/kit", recovery.GetKit)
		r.Put("/auth/recovery/kit", recovery.RegenerateKit)
		r.Post("/auth/recovery/kit/recover", recovery.RecoverWithKit)
		r.Put("/auth/recovery/disable", recovery.DisableRecovery)
		r.Get("/auth/recovery/trusted-contact", recovery.Status) // reuses status for now
		r.Get("/auth/recovery/request", recovery.GetRecoveryRequest)
		r.Post("/auth/recovery/request", recovery.InitiateRecovery)
		r.Delete("/auth/recovery/request", recovery.CancelRecovery)
		r.Get("/users/public-key", recovery.GetPublicKey)
	})

	// Dashboard routes — require identity session + vault unlock
	r.Group(func(r chi.Router) {
		r.Use(identity.RequireSession)
		r.Use(identity.RequireVaultUnlocked)

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
			r.Get("/{projectID}/key-grant", projects.GetKeyGrant)
		})

		r.Route("/orgs", func(r chi.Router) {
			r.Post("/", orgs.Create)
			r.Get("/", orgs.List)
			r.Get("/{orgID}", orgs.Get)
			r.Get("/{orgID}/members", orgs.ListMembers)
			r.Post("/{orgID}/members", orgs.AddMember)
			r.Delete("/{orgID}/members/{memberID}", orgs.RemoveMember)
		})

		// Recovery — requires vault unlocked
		r.Post("/auth/recovery/trusted-contact", recovery.SetTrustedContact)
		r.Delete("/auth/recovery/trusted-contact", recovery.RemoveTrustedContact)
		r.Get("/auth/recovery/incoming-requests", recovery.GetIncomingRequests)
		r.Post("/auth/recovery/request/{id}/approve", recovery.ApproveRecovery)
		r.Post("/auth/recovery/request/{id}/complete", recovery.CompleteRecovery)

		r.Get("/audit-logs", audit.List)
		r.Get("/audit-logs/export", audit.Export)
	})

	// SDK/CLI routes — authenticate via service token (machine access)
	r.Route("/sdk", func(r chi.Router) {
		r.Use(ta.Authenticate)

		r.Get("/whoami", tokens.Whoami)
		r.Get("/vault", projects.GetVaultMaterial)

		// Organizations
		r.Get("/orgs", orgs.ListForToken)
		r.Post("/orgs", orgs.CreateForToken)
		r.Get("/orgs/{orgID}", orgs.Get)
		r.Get("/orgs/{orgID}/members", orgs.ListMembers)
		r.Post("/orgs/{orgID}/members", orgs.AddMemberForToken)
		r.Delete("/orgs/{orgID}/members/{memberID}", orgs.RemoveMemberForToken)

		// Projects
		r.Get("/projects", projects.List)
		r.Post("/projects", projects.CreateForToken)
		r.Get("/projects/{projectID}", projects.Get)
		r.Get("/projects/{projectID}/crypto", projects.GetCrypto)
		r.Get("/projects/{projectID}/key-grant", projects.GetKeyGrantForToken)

		// Tokens
		r.Get("/tokens", tokens.List)
		r.Post("/tokens", tokens.CreateForToken)
		r.Delete("/tokens/{tokenID}", tokens.Revoke)

		// Secrets — read operations
		r.Post("/secrets/bulk", secrets.BulkFetch)
		r.Get("/secrets", secrets.List)
		r.Get("/secrets/{nameHash}", secrets.Get)
		r.Get("/secrets/{nameHash}/versions", secrets.Versions)

		// Secrets — write operations (require read_write permission)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireWrite)

			r.Post("/secrets", secrets.Create)
			r.Put("/secrets/{nameHash}", secrets.Update)
			r.Post("/secrets/{nameHash}/rollback", secrets.Rollback)
			r.Delete("/secrets/{nameHash}", secrets.Delete)
		})
	})
}
