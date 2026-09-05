// Package sourcescfg holds configuration for the built-in scrobble source
// connectors (Spotify, Jellyfin, YouTube Music, ...). It is kept separate
// from the core Koito internal/cfg package so the connectors can be added,
// removed, or reworked without touching upstream configuration internals.
package sourcescfg

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ENCRYPTION_KEY_ENV = "SCROBLLI_ENCRYPTION_KEY"

	SPOTIFY_CLIENT_ID_ENV     = "SCROBLLI_SPOTIFY_CLIENT_ID"
	SPOTIFY_CLIENT_SECRET_ENV = "SCROBLLI_SPOTIFY_CLIENT_SECRET"
	SPOTIFY_REDIRECT_URI_ENV  = "SCROBLLI_SPOTIFY_REDIRECT_URI"
	SPOTIFY_POLL_SECONDS_ENV  = "SCROBLLI_SPOTIFY_POLL_SECONDS"

	JELLYFIN_WEBHOOK_SECRET_ENV = "SCROBLLI_JELLYFIN_WEBHOOK_SECRET"

	YTMUSIC_ENABLED_ENV      = "SCROBLLI_YTMUSIC_ENABLED"
	YTMUSIC_POLL_SECONDS_ENV = "SCROBLLI_YTMUSIC_POLL_SECONDS"
)

const (
	defaultSpotifyPollSeconds  = 60
	defaultYtmusicPollSeconds  = 120
	minAllowedPollSeconds      = 15
)

// EncryptionKey returns the secret used to encrypt credentials/tokens
// persisted by source connectors. Empty means no source connector may be
// enabled (callers must check this before starting any poller).
func EncryptionKey() string {
	return os.Getenv(ENCRYPTION_KEY_ENV)
}

// Spotify

func SpotifyClientID() string     { return os.Getenv(SPOTIFY_CLIENT_ID_ENV) }
func SpotifyClientSecret() string { return os.Getenv(SPOTIFY_CLIENT_SECRET_ENV) }
func SpotifyRedirectURI() string  { return os.Getenv(SPOTIFY_REDIRECT_URI_ENV) }

// SpotifyEnabled reports whether enough configuration is present to offer
// the Spotify source. It does not mean a user has completed the OAuth
// authorization flow yet.
func SpotifyEnabled() bool {
	return SpotifyClientID() != "" && SpotifyClientSecret() != "" && SpotifyRedirectURI() != ""
}

func SpotifyPollInterval() time.Duration {
	return pollInterval(SPOTIFY_POLL_SECONDS_ENV, defaultSpotifyPollSeconds)
}

// Jellyfin

// JellyfinWebhookSecret returns the shared secret that incoming Jellyfin
// webhook requests must present (as a "?token=" query parameter) to be
// accepted. If empty, the Jellyfin webhook endpoint is disabled.
func JellyfinWebhookSecret() string { return os.Getenv(JELLYFIN_WEBHOOK_SECRET_ENV) }

func JellyfinEnabled() bool { return JellyfinWebhookSecret() != "" }

// YouTube Music

func YtmusicEnabled() bool {
	return strings.ToLower(os.Getenv(YTMUSIC_ENABLED_ENV)) == "true"
}

func YtmusicPollInterval() time.Duration {
	return pollInterval(YTMUSIC_POLL_SECONDS_ENV, defaultYtmusicPollSeconds)
}

func pollInterval(env string, def int) time.Duration {
	secs, err := strconv.Atoi(os.Getenv(env))
	if err != nil || secs < minAllowedPollSeconds {
		secs = def
	}
	return time.Duration(secs) * time.Second
}
