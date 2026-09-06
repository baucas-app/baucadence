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
	YTMUSIC_TRACK_VIDEOS_ENV = "SCROBLLI_YTMUSIC_TRACK_VIDEOS"

	YOUTUBE_DATA_API_KEY_ENV = "SCROBLLI_YOUTUBE_DATA_API_KEY"
)

const (
	defaultSpotifyPollSeconds = 60
	defaultYtmusicPollSeconds = 120
	minAllowedPollSeconds     = 15
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

// YtmusicTrackVideosEnabled reports whether regular (non-music) YouTube
// watch history should also be tracked, in addition to YouTube Music.
//
// This is only consulted when YoutubeDataApiKey is empty. Once that key is
// set, regular YouTube video entries are always tracked - imported as
// music listens if the video's category is Music, and as a Video watch
// otherwise - since the API lets us tell the two apart instead of
// shoehorning every non-Music video into the track catalog.
func YtmusicTrackVideosEnabled() bool {
	return strings.ToLower(os.Getenv(YTMUSIC_TRACK_VIDEOS_ENV)) == "true"
}

// YoutubeDataApiKey returns the API key used to look up video metadata
// (title, channel, thumbnail, category) for regular YouTube videos found
// in a Takeout watch-history import. This is a plain API key (no OAuth) -
// video metadata is public data, unlike watch history itself.
func YoutubeDataApiKey() string { return os.Getenv(YOUTUBE_DATA_API_KEY_ENV) }

func pollInterval(env string, def int) time.Duration {
	secs, err := strconv.Atoi(os.Getenv(env))
	if err != nil || secs < minAllowedPollSeconds {
		secs = def
	}
	return time.Duration(secs) * time.Second
}
