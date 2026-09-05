// Package spotify implements a background scrobble source that polls the
// official Spotify Web API for recently played tracks and submits them to
// Koito's catalog, in-process, via catalog.SubmitListen.
package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gabehf/koito/engine/middleware"
	"github.com/gabehf/koito/internal/catalog"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/mbz"
	"github.com/gabehf/koito/internal/sourcescfg"
	"github.com/gabehf/koito/internal/sourcestore"
)

const (
	authorizeURL = "https://accounts.spotify.com/authorize"
	tokenURL     = "https://accounts.spotify.com/api/token"
	apiBaseURL   = "https://api.spotify.com/v1"
	scopes       = "user-read-recently-played user-read-playback-state user-read-currently-playing"

	storeKey        = "spotify"
	oauthCookieName = "scrobbler_spotify_oauth_state"

	clientName = "Spotify"
)

type submitStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

// storedState is the encrypted, on-disk credential/cursor state for the
// Spotify source. There is one document for the whole app: Spotify only
// allows a single active authorization per Client ID/redirect combo, which
// matches Koito's typical single-user (or single-listener) deployment.
type storedState struct {
	UserID             int32  `json:"user_id"`
	RefreshToken       string `json:"refresh_token"`
	LastPlayedAtUnixMs int64  `json:"last_played_at_unix_ms"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type playHistoryItem struct {
	Track struct {
		Name       string `json:"name"`
		DurationMs int32  `json:"duration_ms"`
		Album      struct {
			Name string `json:"name"`
		} `json:"album"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
	} `json:"track"`
	PlayedAt string `json:"played_at"`
}

type recentlyPlayedResponse struct {
	Items []playHistoryItem `json:"items"`
	Error *struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

// Source polls Spotify for one authorized user and writes listens directly
// into the Koito catalog.
type Source struct {
	store      submitStore
	mbz        mbz.MusicBrainzCaller
	kv         *sourcestore.Store
	httpClient *http.Client
}

func New(store submitStore, mbzc mbz.MusicBrainzCaller, kv *sourcestore.Store) *Source {
	return &Source{
		store:      store,
		mbz:        mbzc,
		kv:         kv,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// AuthorizeHandler starts the OAuth flow. It must be mounted behind
// session-cookie authentication so we know which Koito user to attribute
// future listens to.
func (s *Source) AuthorizeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := randomString(24)
		if err != nil {
			http.Error(w, "failed to generate state", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     oauthCookieName,
			Value:    state,
			Path:     "/",
			MaxAge:   600,
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
		})

		q := url.Values{}
		q.Set("client_id", sourcescfg.SpotifyClientID())
		q.Set("response_type", "code")
		q.Set("redirect_uri", sourcescfg.SpotifyRedirectURI())
		q.Set("scope", scopes)
		q.Set("state", state)
		http.Redirect(w, r, authorizeURL+"?"+q.Encode(), http.StatusFound)
	}
}

// CallbackHandler completes the OAuth flow: exchanges the authorization
// code for tokens and persists the refresh token, tied to the Koito user
// who initiated AuthorizeHandler.
func (s *Source) CallbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logger.FromContext(r.Context())

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			http.Error(w, "spotify authorization denied: "+errParam, http.StatusBadRequest)
			return
		}

		cookie, err := r.Cookie(oauthCookieName)
		if err != nil || cookie.Value == "" || cookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "invalid or missing oauth state", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			return
		}

		u := middleware.GetUserFromContext(r.Context())
		if u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		tok, err := s.exchangeCode(r.Context(), code)
		if err != nil {
			l.Err(err).Msg("spotify.CallbackHandler: failed to exchange code")
			http.Error(w, "failed to exchange authorization code", http.StatusBadGateway)
			return
		}
		if tok.RefreshToken == "" {
			http.Error(w, "spotify did not return a refresh token", http.StatusBadGateway)
			return
		}

		state := storedState{
			UserID:             u.ID,
			RefreshToken:       tok.RefreshToken,
			LastPlayedAtUnixMs: time.Now().Add(-24 * time.Hour).UnixMilli(),
		}
		if err := s.kv.Save(storeKey, state); err != nil {
			l.Err(err).Msg("spotify.CallbackHandler: failed to persist state")
			http.Error(w, "failed to save spotify connection", http.StatusInternalServerError)
			return
		}

		l.Info().Msg("spotify.CallbackHandler: Spotify account connected successfully")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><body style="font-family:sans-serif">Spotify connected. You can close this tab.</body></html>`))
	}
}

// StatusHandler reports whether a Spotify account is currently connected.
func (s *Source) StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var st storedState
		connected := s.kv.Load(storeKey, &st) == nil
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"connected": connected})
	}
}

// DisconnectHandler removes the stored Spotify credentials.
func (s *Source) DisconnectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.kv.Delete(storeKey); err != nil {
			http.Error(w, "failed to disconnect", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Run polls Spotify on a fixed interval until ctx is cancelled. It is
// intended to be started in its own goroutine at application startup.
func (s *Source) Run(ctx context.Context) {
	l := logger.FromContext(ctx)
	interval := sourcescfg.SpotifyPollInterval()
	l.Info().Msgf("spotify.Run: Starting Spotify source (poll interval %s)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := s.poll(ctx); err != nil {
			l.Warn().Err(err).Msg("spotify.Run: poll cycle failed")
		}
		select {
		case <-ctx.Done():
			l.Info().Msg("spotify.Run: shutting down")
			return
		case <-ticker.C:
		}
	}
}

func (s *Source) poll(ctx context.Context) error {
	l := logger.FromContext(ctx)

	var state storedState
	if err := s.kv.Load(storeKey, &state); err != nil {
		if err == sourcestore.ErrNotFound {
			// Not yet authorized; nothing to do.
			return nil
		}
		return fmt.Errorf("failed to load state: %w", err)
	}

	accessToken, newRefreshToken, err := s.refreshAccessToken(ctx, state.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh access token: %w", err)
	}
	if newRefreshToken != "" {
		state.RefreshToken = newRefreshToken
	}

	items, err := s.fetchRecentlyPlayed(ctx, accessToken, state.LastPlayedAtUnixMs)
	if err != nil {
		return fmt.Errorf("failed to fetch recently played: %w", err)
	}

	maxPlayedAtMs := state.LastPlayedAtUnixMs
	submitted := 0
	for _, item := range items {
		playedAt, err := time.Parse(time.RFC3339Nano, item.PlayedAt)
		if err != nil {
			l.Warn().Err(err).Str("played_at", item.PlayedAt).Msg("spotify.poll: failed to parse played_at, skipping item")
			continue
		}
		if playedAt.UnixMilli() <= state.LastPlayedAtUnixMs {
			continue
		}

		var artistNames []string
		for _, a := range item.Track.Artists {
			artistNames = append(artistNames, a.Name)
		}
		primaryArtist := ""
		if len(artistNames) > 0 {
			primaryArtist = artistNames[0]
		}

		opts := catalog.SubmitListenOpts{
			MbzCaller:    s.mbz,
			Artist:       primaryArtist,
			ArtistNames:  artistNames,
			TrackTitle:   item.Track.Name,
			ReleaseTitle: item.Track.Album.Name,
			Duration:     item.Track.DurationMs / 1000,
			Time:         playedAt,
			UserID:       state.UserID,
			Client:       clientName,
		}
		if err := catalog.SubmitListen(ctx, s.store, opts); err != nil {
			l.Err(err).Str("track", item.Track.Name).Msg("spotify.poll: failed to submit listen")
			continue
		}
		submitted++
		if ms := playedAt.UnixMilli(); ms > maxPlayedAtMs {
			maxPlayedAtMs = ms
		}
	}

	if maxPlayedAtMs != state.LastPlayedAtUnixMs || submitted > 0 {
		state.LastPlayedAtUnixMs = maxPlayedAtMs
	}
	if err := s.kv.Save(storeKey, state); err != nil {
		return fmt.Errorf("failed to persist updated state: %w", err)
	}
	if submitted > 0 {
		l.Info().Msgf("spotify.poll: submitted %d new listen(s)", submitted)
	}
	return nil
}

func (s *Source) exchangeCode(ctx context.Context, code string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", sourcescfg.SpotifyRedirectURI())
	return s.doTokenRequest(ctx, form)
}

// refreshAccessToken exchanges a refresh token for a new access token. It
// returns the access token, and a new refresh token if Spotify rotated it
// (empty string if unchanged).
func (s *Source) refreshAccessToken(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, err error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	tok, err := s.doTokenRequest(ctx, form)
	if err != nil {
		return "", "", err
	}
	return tok.AccessToken, tok.RefreshToken, nil
}

func (s *Source) doTokenRequest(ctx context.Context, form url.Values) (*tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(sourcescfg.SpotifyClientID(), sourcescfg.SpotifyClientSecret())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("failed to decode token response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify token endpoint returned %d: %s %s", resp.StatusCode, tok.Error, tok.ErrorDesc)
	}
	return &tok, nil
}

func (s *Source) fetchRecentlyPlayed(ctx context.Context, accessToken string, afterUnixMs int64) ([]playHistoryItem, error) {
	q := url.Values{}
	q.Set("limit", "50")
	if afterUnixMs > 0 {
		q.Set("after", fmt.Sprintf("%d", afterUnixMs))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+"/me/player/recently-played?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed recentlyPlayedResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode recently-played response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("spotify api error %d: %s", parsed.Error.Status, parsed.Error.Message)
	}
	return parsed.Items, nil
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
