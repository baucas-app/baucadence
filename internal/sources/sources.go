// Package sources wires together the built-in scrobble source connectors
// (Spotify, Jellyfin, YouTube Music, ...): it validates configuration,
// builds the shared encrypted credential store, mounts each connector's
// HTTP routes, and starts its background poller (if it has one).
package sources

import (
	"context"
	"path/filepath"

	"github.com/gabehf/koito/engine/middleware"
	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/mbz"
	"github.com/gabehf/koito/internal/secure"
	"github.com/gabehf/koito/internal/sources/jellyfin"
	"github.com/gabehf/koito/internal/sources/spotify"
	"github.com/gabehf/koito/internal/sources/youtube"
	"github.com/gabehf/koito/internal/sourcescfg"
	"github.com/gabehf/koito/internal/sourcestore"
	"github.com/go-chi/chi/v5"
)

// BindRoutes mounts HTTP endpoints for every enabled source connector
// (OAuth flows, webhook receivers, status checks) under r. It is a no-op
// (per-source) for any connector that is not configured.
func BindRoutes(r chi.Router, store db.DB, mbzc mbz.MusicBrainzCaller) {
	l := logger.Get()

	key := sourcescfg.EncryptionKey()
	if key == "" {
		if sourcescfg.SpotifyEnabled() || sourcescfg.JellyfinEnabled() || sourcescfg.YtmusicEnabled() {
			l.Warn().Msgf("sources.BindRoutes: %s is not set; no scrobble sources will be enabled even though they appear configured", sourcescfg.ENCRYPTION_KEY_ENV)
		}
		return
	}
	box, err := secure.NewBox(key)
	if err != nil {
		l.Error().Err(err).Msg("sources.BindRoutes: failed to initialize credential encryption; sources disabled")
		return
	}
	kv, err := sourcestore.New(filepath.Join(cfg.ConfigDir(), "sources"), box)
	if err != nil {
		l.Error().Err(err).Msg("sources.BindRoutes: failed to initialize source credential store; sources disabled")
		return
	}

	r.Route("/apis/sources/v1", func(r chi.Router) {
		if sourcescfg.SpotifyEnabled() {
			sp := spotify.New(store, mbzc, kv)
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(store, middleware.AuthModeSessionCookie))
				r.Get("/spotify/authorize", sp.AuthorizeHandler())
				r.Get("/spotify/status", sp.StatusHandler())
				r.Delete("/spotify", sp.DisconnectHandler())
			})
			// The OAuth callback is hit by Spotify's servers redirecting the
			// user's browser back; the browser still carries the Koito
			// session cookie set before authorize, so session auth applies
			// here too.
			r.With(middleware.Authenticate(store, middleware.AuthModeSessionCookie)).
				Get("/spotify/callback", sp.CallbackHandler())
			l.Info().Msg("sources.BindRoutes: Spotify source enabled")
		}

		if sourcescfg.JellyfinEnabled() {
			adminUser, err := store.GetAdminUser(context.Background())
			if err != nil {
				l.Error().Err(err).Msg("sources.BindRoutes: failed to resolve admin user for Jellyfin source; Jellyfin disabled")
			} else {
				jf := jellyfin.New(store, mbzc, adminUser.ID)
				r.Post("/jellyfin/webhook", jf.Handler())
				l.Info().Msg("sources.BindRoutes: Jellyfin source enabled")
			}
		}

		if sourcescfg.YtmusicEnabled() {
			yt := youtube.New(store, mbzc, kv)
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(store, middleware.AuthModeSessionCookie))
				r.Post("/youtube/connect", yt.ConnectHandler())
				r.Get("/youtube/status", yt.StatusHandler())
				r.Delete("/youtube", yt.DisconnectHandler())
			})
			l.Info().Msg("sources.BindRoutes: YouTube source enabled")
		}
	})
}

// StartPollers launches the background goroutines for every enabled
// polling-based source (currently: Spotify, and later YouTube Music).
// Webhook-based sources (Jellyfin) need no poller. It returns immediately;
// each source runs until ctx is cancelled.
func StartPollers(ctx context.Context, store db.DB, mbzc mbz.MusicBrainzCaller) {
	l := logger.FromContext(ctx)

	key := sourcescfg.EncryptionKey()
	if key == "" {
		return
	}
	box, err := secure.NewBox(key)
	if err != nil {
		return
	}
	kv, err := sourcestore.New(filepath.Join(cfg.ConfigDir(), "sources"), box)
	if err != nil {
		return
	}

	if sourcescfg.SpotifyEnabled() {
		sp := spotify.New(store, mbzc, kv)
		go sp.Run(ctx)
	} else {
		l.Debug().Msg("sources.StartPollers: Spotify not configured, skipping")
	}

	if sourcescfg.YtmusicEnabled() {
		yt := youtube.New(store, mbzc, kv)
		go yt.Run(ctx)
	} else {
		l.Debug().Msg("sources.StartPollers: YouTube not configured, skipping")
	}
}
