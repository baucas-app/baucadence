package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gabehf/koito/engine/middleware"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/moodtags"
	"github.com/gabehf/koito/internal/utils"
)

type lastfmSettingsStore interface {
	db.SettingsStore
	db.TrackStore
}

type LastFMStatusResponse struct {
	Connected bool `json:"connected"`
}

// GetLastFMStatusHandler reports whether a LastFM API key is currently
// configured (via the Settings UI or the KOITO_LASTFM_API_KEY env var),
// without ever returning the key itself to the client.
func GetLastFMStatusHandler(store lastfmSettingsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		key, err := moodtags.ResolveApiKey(ctx, store)
		if err != nil {
			l.Err(err).Msg("GetLastFMStatusHandler: Failed to resolve LastFM API key")
			utils.WriteError(w, "failed to get LastFM status", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, LastFMStatusResponse{Connected: key != ""})
	}
}

// ConnectLastFMHandler saves a LastFM API key entered through the Settings
// UI and kicks off a mood/genre tag backfill for the existing library, the
// same backfill that normally only runs once at startup.
func ConnectLastFMHandler(store lastfmSettingsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		var body struct {
			ApiKey string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.WriteError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body.ApiKey == "" {
			utils.WriteError(w, "api_key is required", http.StatusBadRequest)
			return
		}

		if err := store.SetSetting(ctx, moodtags.SettingKeyApiKey, body.ApiKey); err != nil {
			l.Err(err).Msg("ConnectLastFMHandler: Failed to save LastFM API key")
			utils.WriteError(w, "failed to save LastFM API key", http.StatusInternalServerError)
			return
		}

		l.Info().Msg("ConnectLastFMHandler: LastFM API key saved, starting mood tag backfill")
		go moodtags.BackfillTrackTags(logger.NewContext(l), store, moodtags.NewClient(body.ApiKey))

		utils.WriteJSON(w, http.StatusOK, LastFMStatusResponse{Connected: true})
	}
}

// DisconnectLastFMHandler clears a LastFM API key saved through the
// Settings UI. If KOITO_LASTFM_API_KEY is still set in the environment,
// LastFM features remain enabled using that value instead.
func DisconnectLastFMHandler(store lastfmSettingsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		if err := store.SetSetting(ctx, moodtags.SettingKeyApiKey, ""); err != nil {
			l.Err(err).Msg("DisconnectLastFMHandler: Failed to clear LastFM API key")
			utils.WriteError(w, "failed to clear LastFM API key", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
