// Package user_lookup reads Better Auth user profile rows from public.users in the API database.
// The zEnv API does not call Better Auth admin routes for identity display — same Postgres holds
// the Drizzle/Better Auth schema (see apps/identity/drizzle).
package user_lookup

import (
	"context"
	"errors"
	"strings"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/model"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

var ErrNotFound = errors.New("user_lookup: not found")

// ByID returns name and email for a Better Auth user id, or ok=false when absent.
func ByID(ctx context.Context, db qrm.Queryable, id uuid.UUID) (name, email string, ok bool, err error) {
	var row model.Users
	err = SELECT(table.Users.Name, table.Users.Email).
		FROM(table.Users).
		WHERE(table.Users.ID.EQ(UUID(id))).
		QueryContext(ctx, db, &row)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	return row.Name, row.Email, true, nil
}

// IDByEmail returns the user id for an exact email match (case-insensitive).
func IDByEmail(ctx context.Context, db qrm.Queryable, email string) (uuid.UUID, error) {
	norm := strings.TrimSpace(strings.ToLower(email))
	if norm == "" {
		return uuid.Nil, ErrNotFound
	}
	var row struct {
		ID uuid.UUID `alias:"users.id"`
	}
	err := SELECT(table.Users.ID).
		FROM(table.Users).
		WHERE(LOWER(table.Users.Email).EQ(String(norm))).
		LIMIT(1).
		QueryContext(ctx, db, &row)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return row.ID, nil
}

// BatchByIDs returns name+email keyed by user id string. Missing ids are omitted (no error).
func BatchByIDs(ctx context.Context, db qrm.Queryable, ids []uuid.UUID) (map[string]struct{ Name, Email string }, error) {
	out := make(map[string]struct{ Name, Email string })
	if len(ids) == 0 {
		return out, nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	var uniq []uuid.UUID
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out, nil
	}
	exprs := make([]Expression, len(uniq))
	for i, id := range uniq {
		exprs[i] = UUID(id)
	}
	var rows []model.Users
	err := SELECT(table.Users.ID, table.Users.Name, table.Users.Email).
		FROM(table.Users).
		WHERE(table.Users.ID.IN(exprs...)).
		QueryContext(ctx, db, &rows)
	if err != nil {
		return nil, err
	}
	for _, u := range rows {
		if u.ID == uuid.Nil {
			continue
		}
		out[u.ID.String()] = struct{ Name, Email string }{Name: u.Name, Email: u.Email}
	}
	return out, nil
}
