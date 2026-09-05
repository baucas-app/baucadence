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
	"github.com/gabehf/koito/internal/logger"
	mbz "github.com/gabehf/koito/internal/mbz"
	"github.com/gabehf/koito/internal/sourcescfg"
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
func ImportYoutubeTakeoutFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning YouTube Takeout import on file: %s", filename)
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

	trackVideos := sourcescfg.YtmusicTrackVideosEnabled()
	imported := 0

	for _, item := range items {
		isMusic := item.Header == "YouTube Music"
		isVideo := item.Header == "YouTube"
		if !isMusic && !isVideo {
			continue
		}
		if isVideo && !trackVideos {
			continue
		}
		if !inImportTimeWindow(item.Time) {
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

// cleanYoutubeChannelName strips the "- Topic" suffix Google appends to
// auto-generated artist channels on YouTube Music, so imported artist
// names match what the live YouTube Music source already produces.
func cleanYoutubeChannelName(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), " - Topic")
}
