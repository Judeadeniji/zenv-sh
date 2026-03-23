package handler

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/Judeadeniji/zenv-sh/api/internal/middleware"
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

	var prefs json.RawMessage
	err := h.db.QueryRowContext(r.Context(),
		`SELECT preferences FROM users WHERE id = $1`, sess.UserID,
	).Scan(&prefs)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, json.RawMessage(`{}`))
			return
		}
		slog.Error("preferences: get", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to read preferences"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(prefs)
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

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024)) // 64KB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	// Validate it's valid JSON.
	if !json.Valid(body) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON"})
		return
	}

	// Shallow merge with Postgres || operator and return the result.
	var merged json.RawMessage
	err = h.db.QueryRowContext(r.Context(),
		`UPDATE users SET preferences = preferences || $1, updated_at = NOW()
		 WHERE id = $2
		 RETURNING preferences`,
		body, sess.UserID,
	).Scan(&merged)
	if err != nil {
		slog.Error("preferences: update", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update preferences"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(merged)
}
