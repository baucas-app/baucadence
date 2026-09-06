package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/catalog"
	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/importprogress"
	"github.com/gabehf/koito/internal/logger"
	mbz "github.com/gabehf/koito/internal/mbz"
	"github.com/gabehf/koito/internal/sourcescfg"
	"github.com/gabehf/koito/internal/ytdata"
)

// youtubeTakeoutSubtitle is the "channel" a watch-history entry links to.
// For YouTube Music tracks this is normally an auto-generated artist
// channel, often suffixed "- Topic".
type youtubeTakeoutSubtitle struct {
	Name string `json:"name"`
}

// youtubeTakeoutItem is one entry of a Google Takeout
// "YouTube and YouTube Music" > history > watch-history.json export.
type youtubeTakeoutItem struct {
	Header    string                   `json:"header"` // "YouTube Music" or "YouTube"
	Title     string                   `json:"title"`  // e.g. "Watched Song Name"
	TitleUrl  string                   `json:"titleUrl"`
	Subtitles []youtubeTakeoutSubtitle `json:"subtitles"`
	Time      time.Time                `json:"time"`
}

// ImportYoutubeTakeoutFile imports a Google Takeout watch-history.json
// export. The live YouTube source (internal/sources/youtube) only ever
// sees a recent slice of history through YouTube's undocumented internal
// API and has no real per-item timestamp to work with, so it records
// listens at poll time. Takeout, by contrast, gives the user's full
// history with a real timestamp per entry, so it's the way to backfill
// older listens rather than relying on the live poller alone.
//
// Regular (non-Music) YouTube entries are classified with the YouTube
// Data API when SCROBLLI_YOUTUBE_DATA_API_KEY is configured: videos in
// the "Music" category are imported as music listens same as before,
// everything else is imported as a Video watch (internal/models.Video),
// kept separate from the Artist/Track catalog. Without that key, the
// legacy behavior applies: regular videos are only imported (as music
// listens, channel-as-artist) when SCROBLLI_YTMUSIC_TRACK_VIDEOS=true.
func ImportYoutubeTakeoutFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) (err error) {
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning YouTube Takeout import on file: %s", filename)
	defer func() { importprogress.Finish(filename, err) }()
	file, err := os.Open(path.Join(cfg.ConfigDir(), "import", filename))
	if err != nil {
		l.Err(err).Msgf("Failed to read import file: %s", filename)
		return fmt.Errorf("ImportYoutubeTakeoutFile: %w", err)
	}
	defer file.Close()

	var throttleFunc = func() {}
	if ms := cfg.ThrottleImportMs(); ms > 0 {
		throttleFunc = func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}

	items := make([]youtubeTakeoutItem, 0)
	if err := json.NewDecoder(file).Decode(&items); err != nil {
		return fmt.Errorf("ImportYoutubeTakeoutFile: %w", err)
	}
	importprogress.SetTotal(filename, len(items))

	apiKey := sourcescfg.YoutubeDataApiKey()
	trackVideos := sourcescfg.YtmusicTrackVideosEnabled()

	var videoInfo map[string]ytdata.VideoInfo
	if apiKey != "" {
		videoInfo, err = fetchVideoInfoForItems(ctx, apiKey, items)
		if err != nil {
			// Metadata lookup failing shouldn't block the whole import -
			// fall back to legacy behavior for regular videos this run.
			l.Warn().Err(err).Msg("ImportYoutubeTakeoutFile: YouTube Data API lookup failed, falling back to legacy video handling")
			videoInfo = nil
		}
	}

	imported := 0

	for _, item := range items {
		importprogress.Advance(filename)
		isMusic := item.Header == "YouTube Music"
		isVideo := item.Header == "YouTube"
		if !isMusic && !isVideo {
			continue
		}
		if !inImportTimeWindow(item.Time) {
			continue
		}

		if isVideo && videoInfo != nil {
			if imported0, ok := importVideoItem(ctx, store, mbzc, item, videoInfo); ok {
				imported += imported0
				throttleFunc()
			}
			continue
		}

		if isVideo && !trackVideos {
			continue
		}

		title := strings.TrimPrefix(item.Title, "Watched ")
		if title == "" || len(item.Subtitles) == 0 {
			continue
		}
		artist := cleanYoutubeChannelName(item.Subtitles[0].Name)
		if artist == "" {
			continue
		}

		client := "YouTube Music"
		if isVideo {
			client = "YouTube"
		}

		opts := catalog.SubmitListenOpts{
			MbzCaller:      mbzc,
			Artist:         artist,
			ArtistNames:    []string{artist},
			TrackTitle:     title,
			Time:           item.Time,
			Client:         client,
			UserID:         1,
			SkipCacheImage: !cfg.FetchImagesDuringImport(),
		}
		if err := catalog.SubmitListen(ctx, store, opts); err != nil {
			l.Err(err).Str("track", title).Msg("Failed to import YouTube Takeout item")
			continue
		}
		imported++
		throttleFunc()
	}
	return finishImport(ctx, filename, imported)
}

// fetchVideoInfoForItems collects every video ID referenced by a regular
// (non-Music) YouTube entry within the import time window and resolves
// them all up front via the YouTube Data API, so the main import loop
// below is a simple map lookup per item instead of one API call per item.
func fetchVideoInfoForItems(ctx context.Context, apiKey string, items []youtubeTakeoutItem) (map[string]ytdata.VideoInfo, error) {
	idSet := make(map[string]struct{})
	for _, item := range items {
		if item.Header != "YouTube" || !inImportTimeWindow(item.Time) {
			continue
		}
		if id := ytdata.VideoIDFromURL(item.TitleUrl); id != "" {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return map[string]ytdata.VideoInfo{}, nil
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	client := ytdata.New(apiKey)
	return client.GetVideos(ctx, ids)
}

// importVideoItem classifies and imports one regular YouTube entry using
// pre-fetched Data API metadata: Music-category videos become a normal
// music listen (same as the legacy behavior), everything else becomes a
// Video watch. It reports how many listens/watches it recorded (0 or 1)
// and whether the item was recognized at all (false = fell through to the
// caller's legacy handling, e.g. a deleted video with no metadata).
func importVideoItem(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, item youtubeTakeoutItem, videoInfo map[string]ytdata.VideoInfo) (int, bool) {
	l := logger.FromContext(ctx)
	id := ytdata.VideoIDFromURL(item.TitleUrl)
	info, ok := videoInfo[id]
	if id == "" || !ok {
		return 0, false
	}

	if info.IsMusic() {
		artist := info.ChannelName
		if artist == "" && len(item.Subtitles) > 0 {
			artist = cleanYoutubeChannelName(item.Subtitles[0].Name)
		} else {
			artist = cleanYoutubeChannelName(artist)
		}
		title := info.Title
		if title == "" {
			title = strings.TrimPrefix(item.Title, "Watched ")
		}
		if artist == "" || title == "" {
			return 0, false
		}
		opts := catalog.SubmitListenOpts{
			MbzCaller:      mbzc,
			Artist:         artist,
			ArtistNames:    []string{artist},
			TrackTitle:     title,
			Time:           item.Time,
			Client:         "YouTube",
			UserID:         1,
			SkipCacheImage: !cfg.FetchImagesDuringImport(),
		}
		if err := catalog.SubmitListen(ctx, store, opts); err != nil {
			l.Err(err).Str("track", title).Msg("Failed to import YouTube Takeout music video")
			return 0, true
		}
		return 1, true
	}

	format := "video"
	if info.IsShort() {
		format = "short"
	}
	video, err := store.UpsertVideo(ctx, db.SaveVideoOpts{
		YoutubeID:   id,
		Title:       info.Title,
		ChannelID:   info.ChannelID,
		ChannelName: info.ChannelName,
		Thumbnail:   info.Thumbnail,
		Category:    info.Category,
		Format:      format,
	})
	if err != nil {
		l.Err(err).Str("video_id", id).Msg("Failed to save YouTube Takeout video")
		return 0, true
	}
	if err := store.SaveVideoWatch(ctx, db.SaveVideoWatchOpts{
		VideoID: video.ID,
		Time:    item.Time,
		UserID:  1,
		Client:  "YouTube",
	}); err != nil {
		l.Err(err).Str("video_id", id).Msg("Failed to save YouTube Takeout video watch")
		return 0, true
	}
	return 1, true
}

// cleanYoutubeChannelName strips the "- Topic" suffix Google appends to
// auto-generated artist channels on YouTube Music, so imported artist
// names match what the live YouTube Music source already produces.
func cleanYoutubeChannelName(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), " - Topic")
}
