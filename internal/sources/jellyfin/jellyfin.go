// Package jellyfin implements a scrobble source fed by Jellyfin's official
// "Webhook" plugin. Unlike Spotify/YouTube Music, this requires no polling:
// Jellyfin pushes an event to us the moment a track finishes playing.
package jellyfin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gabehf/koito/internal/catalog"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/mbz"
	"github.com/gabehf/koito/internal/sourcescfg"
)

const clientName = "Jellyfin"

type submitStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

// webhookPayload is the JSON body Jellyfin's Webhook plugin should be
// configured to send. Paste the template from this project's README into
// the plugin's "Item" notification template so the field names match.
type webhookPayload struct {
	NotificationType string `json:"NotificationType"`
	ItemType         string `json:"ItemType"`
	Name             string `json:"Name"`
	Album            string `json:"Album"`
	Artist           string `json:"Artist"`
	// RunTimeTicks is Jellyfin's duration unit: 10,000 ticks per millisecond.
	RunTimeTicks int64 `json:"RunTimeTicks"`
	// PlaybackPositionTicks lets us ignore tracks that were only skipped
	// through briefly, matching common scrobble conventions (roughly half
	// the track, or at least 4 minutes).
	PlaybackPositionTicks int64  `json:"PlaybackPositionTicks"`
	UserId                string `json:"UserId"`
}

// Source receives Jellyfin webhook events and writes them into the Koito
// catalog. There is no polling loop; Handler is mounted directly as an
// HTTP route.
type Source struct {
	store submitStore
	mbz   mbz.MusicBrainzCaller
	// userID is the Koito user these listens are attributed to. Jellyfin's
	// webhook payload identifies a Jellyfin user, not a Koito one, so for
	// now this is configured once (single-listener setup) rather than
	// mapped per Jellyfin account.
	userID int32
}

func New(store submitStore, mbzc mbz.MusicBrainzCaller, koitoUserID int32) *Source {
	return &Source{store: store, mbz: mbzc, userID: koitoUserID}
}

const ticksPerSecond = 10_000_000

// Handler validates the shared-secret token and, for a completed audio
// item, submits a listen.
func (s *Source) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logger.FromContext(r.Context())

		if r.URL.Query().Get("token") != sourcescfg.JellyfinWebhookSecret() {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		var payload webhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			l.Warn().Err(err).Msg("jellyfin.Handler: failed to decode webhook payload")
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

		if payload.ItemType != "Audio" {
			return
		}
		// Only scrobble once a track has actually finished (or been
		// listened to substantially), matching Last.fm/ListenBrainz
		// scrobble conventions: skip anything played for less than half
		// its duration or 4 minutes, whichever is shorter.
		if payload.NotificationType != "PlaybackStop" {
			return
		}
		durationSeconds := payload.RunTimeTicks / ticksPerSecond
		playedSeconds := payload.PlaybackPositionTicks / ticksPerSecond
		threshold := durationSeconds / 2
		if threshold > 240 {
			threshold = 240
		}
		if durationSeconds > 0 && playedSeconds < threshold {
			l.Debug().Str("track", payload.Name).Msg("jellyfin.Handler: playback too short, not scrobbling")
			return
		}
		if payload.Artist == "" || payload.Name == "" {
			return
		}

		opts := catalog.SubmitListenOpts{
			MbzCaller:    s.mbz,
			Artist:       payload.Artist,
			ArtistNames:  []string{payload.Artist},
			TrackTitle:   payload.Name,
			ReleaseTitle: payload.Album,
			Duration:     int32(durationSeconds),
			Time:         time.Now(),
			UserID:       s.userID,
			Client:       clientName,
		}
		if err := catalog.SubmitListen(context.Background(), s.store, opts); err != nil {
			l.Err(err).Str("track", payload.Name).Msg("jellyfin.Handler: failed to submit listen")
		}
	}
}
