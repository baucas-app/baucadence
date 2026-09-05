// Package youtube implements two best-effort scrobble sources fed by
// Google's undocumented "InnerTube" API: YouTube Music listen history and
// regular YouTube watch history. Unlike Spotify (official API) and
// Jellyfin (official webhook), there is no supported public API for
// either of these, so this package authenticates using a copied browser
// session cookie and parses YouTube's internal response format. Expect to
// need to adjust the parsing logic in this file if YouTube changes its
// internal page structure; that is the accepted tradeoff for automatic,
// no-upload tracking of YouTube activity.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	storeKey = "youtube"

	musicClientLabel = "YouTube Music"
	videoClientLabel = "YouTube"

	maxSeenIDs = 2000
)

type submitStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

// storedState holds the pasted browser cookie and de-duplication state.
// YouTube's history pages group items by day rather than giving a precise
// timestamp per item, so unlike Spotify we de-duplicate by video ID
// instead of a time cursor, and record listens at "now" (poll time).
type storedState struct {
	UserID       int32    `json:"user_id"`
	Cookie       string   `json:"cookie"`
	SeenMusicIDs []string `json:"seen_music_ids"`
	SeenVideoIDs []string `json:"seen_video_ids"`
}

type Source struct {
	store submitStore
	mbz   mbz.MusicBrainzCaller
	kv    *sourcestore.Store
}

func New(store submitStore, mbzc mbz.MusicBrainzCaller, kv *sourcestore.Store) *Source {
	return &Source{store: store, mbz: mbzc, kv: kv}
}

type connectRequest struct {
	Cookie string `json:"cookie"`
}

// ConnectHandler stores the browser cookie pasted by the user (see the
// project README for how to copy it), tied to the currently logged-in
// Koito user.
func (s *Source) ConnectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := middleware.GetUserFromContext(r.Context())
		if u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req connectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Cookie == "" {
			http.Error(w, "missing cookie", http.StatusBadRequest)
			return
		}

		var existing storedState
		_ = s.kv.Load(storeKey, &existing) // ok if not found; we overwrite below

		state := storedState{
			UserID:       u.ID,
			Cookie:       req.Cookie,
			SeenMusicIDs: existing.SeenMusicIDs,
			SeenVideoIDs: existing.SeenVideoIDs,
		}
		if err := s.kv.Save(storeKey, state); err != nil {
			http.Error(w, "failed to save connection", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Source) StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var st storedState
		connected := s.kv.Load(storeKey, &st) == nil
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"connected": connected})
	}
}

func (s *Source) DisconnectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.kv.Delete(storeKey); err != nil {
			http.Error(w, "failed to disconnect", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Run polls YouTube Music and (if enabled) regular YouTube watch history on
// a fixed interval until ctx is cancelled.
func (s *Source) Run(ctx context.Context) {
	l := logger.FromContext(ctx)
	interval := sourcescfg.YtmusicPollInterval()
	l.Info().Msgf("youtube.Run: Starting YouTube source(s) (poll interval %s)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := s.poll(ctx); err != nil {
			l.Warn().Err(err).Msg("youtube.Run: poll cycle failed")
		}
		select {
		case <-ctx.Done():
			l.Info().Msg("youtube.Run: shutting down")
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
			return nil
		}
		return fmt.Errorf("failed to load state: %w", err)
	}
	if state.Cookie == "" {
		return nil
	}

	client := newInnerTubeClient(state.Cookie)
	changed := false

	musicSubmitted, newMusicIDs, err := s.pollMusic(ctx, client, state.SeenMusicIDs, state.UserID)
	if err != nil {
		l.Warn().Err(err).Msg("youtube.poll: YouTube Music history fetch failed")
	} else if len(newMusicIDs) > 0 {
		state.SeenMusicIDs = trimSeen(append(state.SeenMusicIDs, newMusicIDs...))
		changed = true
	}

	if sourcescfg.YtmusicTrackVideosEnabled() {
		videoSubmitted, newVideoIDs, err := s.pollVideos(ctx, client, state.SeenVideoIDs, state.UserID)
		if err != nil {
			l.Warn().Err(err).Msg("youtube.poll: YouTube watch history fetch failed")
		} else if len(newVideoIDs) > 0 {
			state.SeenVideoIDs = trimSeen(append(state.SeenVideoIDs, newVideoIDs...))
			changed = true
			l.Info().Msgf("youtube.poll: submitted %d new video watch(es)", videoSubmitted)
		}
	}

	if musicSubmitted > 0 {
		l.Info().Msgf("youtube.poll: submitted %d new YouTube Music listen(s)", musicSubmitted)
	}

	if changed {
		if err := s.kv.Save(storeKey, state); err != nil {
			return fmt.Errorf("failed to persist updated state: %w", err)
		}
	}
	return nil
}

func (s *Source) pollMusic(ctx context.Context, client *innerTubeClient, seen []string, userID int32) (submitted int, newIDs []string, err error) {
	seenSet := toSet(seen)

	resp, err := client.browse(ctx, musicHost, musicAPIKey, musicClientName, musicClientVer, musicHistoryID)
	if err != nil {
		return 0, nil, err
	}

	var renderers []map[string]any
	findRenderers(resp, "musicResponsiveListItemRenderer", &renderers)

	// Iterate oldest-first among the newly discovered items so submission
	// order is closer to actual listen order, though YouTube gives us no
	// precise per-item timestamp to sort by definitively.
	for i := len(renderers) - 1; i >= 0; i-- {
		item := renderers[i]
		videoID := extractVideoID(item)
		title := ""
		if cols, ok := item["flexColumns"].([]any); ok && len(cols) > 0 {
			title = flexColumnText(cols[0])
		}
		artist := ""
		if cols, ok := item["flexColumns"].([]any); ok && len(cols) > 1 {
			artist = flexColumnText(cols[1])
		}
		if videoID == "" || title == "" || artist == "" {
			continue
		}
		if _, ok := seenSet[videoID]; ok {
			continue
		}

		opts := catalog.SubmitListenOpts{
			MbzCaller:   s.mbz,
			Artist:      artist,
			ArtistNames: []string{artist},
			TrackTitle:  title,
			Time:        time.Now(),
			UserID:      userID,
			Client:      musicClientLabel,
		}
		if err := catalog.SubmitListen(ctx, s.store, opts); err != nil {
			logger.FromContext(ctx).Err(err).Str("track", title).Msg("youtube.pollMusic: failed to submit listen")
			continue
		}
		submitted++
		newIDs = append(newIDs, videoID)
	}
	return submitted, newIDs, nil
}

func (s *Source) pollVideos(ctx context.Context, client *innerTubeClient, seen []string, userID int32) (submitted int, newIDs []string, err error) {
	seenSet := toSet(seen)

	resp, err := client.browse(ctx, youtubeHost, youtubeAPIKey, youtubeClientName, youtubeClientVer, youtubeHistoryID)
	if err != nil {
		return 0, nil, err
	}

	var renderers []map[string]any
	findRenderers(resp, "videoRenderer", &renderers)

	for i := len(renderers) - 1; i >= 0; i-- {
		item := renderers[i]
		videoID, _ := item["videoId"].(string)
		title := firstRunText(item, "title")
		channel := firstRunText(item, "ownerText")
		if channel == "" {
			channel = firstRunText(item, "shortBylineText")
		}
		if videoID == "" || title == "" {
			continue
		}
		if _, ok := seenSet[videoID]; ok {
			continue
		}
		if channel == "" {
			channel = "YouTube"
		}

		// Videos are stored as "tracks" with the channel as the artist so
		// they show up in Koito's existing catalog/stats views alongside
		// music, clearly labeled by the "YouTube" client tag.
		opts := catalog.SubmitListenOpts{
			MbzCaller:   s.mbz,
			Artist:      channel,
			ArtistNames: []string{channel},
			TrackTitle:  title,
			Time:        time.Now(),
			UserID:      userID,
			Client:      videoClientLabel,
		}
		if err := catalog.SubmitListen(ctx, s.store, opts); err != nil {
			logger.FromContext(ctx).Err(err).Str("video", title).Msg("youtube.pollVideos: failed to submit listen")
			continue
		}
		submitted++
		newIDs = append(newIDs, videoID)
	}
	return submitted, newIDs, nil
}

// extractVideoID looks in the couple of places InnerTube commonly puts a
// music list item's video ID.
func extractVideoID(item map[string]any) string {
	if pid, ok := item["playlistItemData"].(map[string]any); ok {
		if id, ok := pid["videoId"].(string); ok && id != "" {
			return id
		}
	}
	if nav, ok := item["navigationEndpoint"].(map[string]any); ok {
		if we, ok := nav["watchEndpoint"].(map[string]any); ok {
			if id, ok := we["videoId"].(string); ok && id != "" {
				return id
			}
		}
	}
	return ""
}

// flexColumnText extracts the first run's text from a
// musicResponsiveListItemFlexColumnRenderer entry.
func flexColumnText(col any) string {
	m, ok := col.(map[string]any)
	if !ok {
		return ""
	}
	inner, ok := m["musicResponsiveListItemFlexColumnRenderer"].(map[string]any)
	if !ok {
		return ""
	}
	return firstRunText(inner, "text")
}

func toSet(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func trimSeen(ids []string) []string {
	if len(ids) <= maxSeenIDs {
		return ids
	}
	return ids[len(ids)-maxSeenIDs:]
}
