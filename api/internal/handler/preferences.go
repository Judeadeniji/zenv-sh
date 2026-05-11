package handler

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
	"github.com/Judeadeniji/zenv-sh/api/internal/store/gen/zenv/public/table"
)

// PreferencesHandler handles user preference endpoints.
type PreferencesHandler struct {
	db *sql.DB
}

func NewPreferencesHandler(db *sql.DB) *PreferencesHandler {
	return &PreferencesHandler{db: db}
}

// Get returns the current user's preferences.
//
//	@Summary		Get user preferences
//	@Description	Returns the current user's preference JSON.
//	@Tags			preferences
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/preferences [get]
func (h *PreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.UserID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	var result struct {
		Preferences json.RawMessage
	}

	err := SELECT(
		table.Identities.Preferences,
	).FROM(
		table.Identities,
	).WHERE(
		table.Identities.IdentityID.EQ(UUID(userID)),
	).QueryContext(r.Context(), h.db, &result)

	if err != nil {
		if err == qrm.ErrNoRows {
			writeJSON(w, http.StatusOK, json.RawMessage(`{}`))
			return
		}
		slog.Error("preferences: get", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to read preferences"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result.Preferences)
}

// Update merges the provided JSON into the user's preferences.
//
//	@Summary		Update user preferences
//	@Description	Shallow-merges the provided JSON into the current preferences.
//	@Tags			preferences
//	@Accept			json
//	@Produce		json
//	@Param			body	body		map[string]any	true	"Preference keys to set"
//	@Success		200		{object}	map[string]any
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		SessionAuth
//	@Router			/preferences [put]
func (h *PreferencesHandler) Update(w http.ResponseWriter, r *http.Request) {
	sess := middleware.GetSession(r.Context())
	if sess == nil || sess.UserID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "authentication required"})
		return
	}
	userID, _ := uuid.Parse(sess.UserID)

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if !json.Valid(body) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}

	var result struct {
		Preferences json.RawMessage
	}

	// StringExp wraps Raw to satisfy the StringExpression interface (fixing the BETWEEN error)
	err = table.Identities.UPDATE(
		table.Identities.Preferences,
		table.Identities.UpdatedAt,
	).SET(
		Raw("identities.preferences || #payload#::jsonb", map[string]interface{}{"payload": string(body)}),
		TimestampT(time.Now().UTC()),
	).WHERE(
		table.Identities.IdentityID.EQ(UUID(userID)),
	).RETURNING(
		table.Identities.Preferences,
	).QueryContext(r.Context(), h.db, &result)

	if err != nil {
		slog.Error("preferences: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update preferences"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result.Preferences)
}
