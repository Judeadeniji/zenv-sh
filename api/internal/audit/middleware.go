package audit

import (
	"context"
	"net/http"
	"strings"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// Middleware returns an HTTP middleware that automatically emits audit events
// for every request. It inspects the route path and method to determine the action.
func (w *Writer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ww := chimw.NewWrapResponseWriter(rw, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		status := ww.Status()
		action := routeToAction(r.Method, r.URL.Path)
		if action == "" {
			return
		}

		result := "success"
		if status >= 400 && status < 500 {
			result = "denied"
		} else if status >= 500 {
			result = "error"
		} else if status < 200 || status >= 300 {
			return
		}

		event := Event{
			Action:    action,
			IP:        extractIP(r),
			UserAgent: r.UserAgent(),
			Result:    result,
		}

		if pid := r.URL.Query().Get("project_id"); pid != "" {
			if u, err := uuid.Parse(pid); err == nil {
				event.ProjectID = &u
			}
		}
		if tid, ok := r.Context().Value(tokenIDCtxKey).(uuid.UUID); ok {
			event.TokenID = &tid
		}
		if uid, ok := r.Context().Value(userIDCtxKey).(uuid.UUID); ok {
			event.UserID = &uid
		}

		w.Emit(r.Context(), event)
	})
}

type ctxKey string

const (
	tokenIDCtxKey ctxKey = "audit_token_id"
	userIDCtxKey  ctxKey = "audit_user_id"
)

// SetTokenID stores the token ID in context for audit logging.
func SetTokenID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, tokenIDCtxKey, id)
}

// SetUserID stores the user ID in context for audit logging.
func SetUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDCtxKey, id)
}

func routeToAction(method, path string) string {
	p := strings.TrimPrefix(path, "/v1/")
	p = strings.TrimPrefix(p, "sdk/")

	switch {
	case strings.HasPrefix(p, "secrets/bulk") && method == "POST":
		return "secret.bulk_read"
	case strings.HasPrefix(p, "secrets") && method == "POST":
		return "secret.create"
	case strings.HasPrefix(p, "secrets") && method == "GET" && strings.Count(p, "/") > 0:
		return "secret.read"
	case strings.HasPrefix(p, "secrets") && method == "GET":
		return "secret.list"
	case strings.HasPrefix(p, "secrets") && method == "PUT":
		return "secret.update"
	case strings.HasPrefix(p, "secrets") && method == "DELETE":
		return "secret.delete"
	case strings.HasPrefix(p, "tokens") && method == "POST":
		return "token.create"
	case strings.HasPrefix(p, "tokens") && method == "DELETE":
		return "token.revoke"
	case strings.HasPrefix(p, "auth/signup"):
		return "auth.signup"
	case strings.HasPrefix(p, "auth/login"):
		return "auth.login"
	case strings.HasPrefix(p, "auth/unlock"):
		return "auth.unlock"
	case strings.HasPrefix(p, "auth/logout"):
		return "auth.logout"
	case strings.HasPrefix(p, "projects") && method == "POST":
		return "project.create"
	default:
		return ""
	}
}
