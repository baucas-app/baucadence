package importer

import (
	"context"
	"fmt"
	"html"
	"os"
	"path"
	"regexp"
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

// Google Takeout offers two output formats for "YouTube and YouTube
// Music" history: JSON (watch-history.json, see youtube.go) and HTML
// (watch-history.html / "Wiedergabeverlauf.html" depending on the
// account's language). Users who didn't explicitly pick JSON in
// Takeout's format picker get HTML by default, so this file supports
// that export too.
//
// The HTML export has no equivalent of the JSON "header" field, so there
// is no way to tell a YouTube Music play apart from a regular YouTube
// video from the markup alone - every entry is rendered under a plain
// "YouTube" heading. Classification therefore always goes through the
// YouTube Data API (SCROBLLI_YOUTUBE_DATA_API_KEY), the same lookup the
// JSON path already uses for its own regular-video entries. Without that
// key configured, entries can't be told apart from a music listen at
// all, so - to avoid guessing wrong and polluting the Artist/Track
// catalog - they are only imported as generic Video watches.

var (
	youtubeHtmlEntryRe = regexp.MustCompile(`(?s)<div class="content-cell mdl-cell mdl-cell--6-col mdl-typography--body-1">(.*?)</div>`)
	youtubeHtmlLinkRe  = regexp.MustCompile(`<a href="([^"]*)">([^<]*)</a>`)
	youtubeHtmlTimeRe  = regexp.MustCompile(`(\d{2})\.(\d{2})\.(\d{4}), (\d{2}:\d{2}:\d{2}) (\S+)<br>`)
)

// youtubeHtmlTzOffsets maps the timezone abbreviations Google Takeout
// writes into German-locale HTML exports to their UTC offset in seconds.
// Takeout renders timestamps in the account's local time at export time,
// so a long history can mix standard and daylight-saving abbreviations.
var youtubeHtmlTzOffsets = map[string]int{
	"MESZ": 2 * 3600, // Mitteleuropäische Sommerzeit / CEST
	"MEZ":  1 * 3600, // Mitteleuropäische Zeit / CET
	"CEST": 2 * 3600,
	"CET":  1 * 3600,
	"UTC":  0,
	"GMT":  0,
}

type youtubeHtmlEntry struct {
	VideoID     string
	VideoTitle  string
	ChannelName string
	Time        time.Time
}

// ImportYoutubeTakeoutHTMLFile imports a Google Takeout watch-history.html
// (or German "Wiedergabeverlauf.html") export. See the package comment
// above for how this differs from the JSON import path.
func ImportYoutubeTakeoutHTMLFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) (err error) {
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning YouTube Takeout HTML import on file: %s", filename)
	defer func() { importprogress.Finish(filename, err) }()

	raw, err := os.ReadFile(path.Join(cfg.ConfigDir(), "import", filename))
	if err != nil {
		l.Err(err).Msgf("Failed to read import file: %s", filename)
		return fmt.Errorf("ImportYoutubeTakeoutHTMLFile: %w", err)
	}

	entries := parseYoutubeHtmlEntries(string(raw))
	importprogress.SetTotal(filename, len(entries))

	apiKey := sourcescfg.YoutubeDataApiKey()
	trackVideos := sourcescfg.YtmusicTrackVideosEnabled()

	videoInfo := fetchVideoInfoForHtmlEntries(ctx, apiKey, entries)

	var throttleFunc = func() {}
	if ms := cfg.ThrottleImportMs(); ms > 0 {
		throttleFunc = func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}

	imported := 0
	for _, e := range entries {
		importprogress.Advance(filename)
		if e.VideoID == "" || !inImportTimeWindow(e.Time) {
			continue
		}

		if info, ok := videoInfo[e.VideoID]; ok {
			if n := importClassifiedHtmlEntry(ctx, store, mbzc, e, info); n > 0 {
				imported += n
				throttleFunc()
			}
			continue
		}

		if !trackVideos {
			if n := importGenericVideoWatch(ctx, store, e); n > 0 {
				imported += n
				throttleFunc()
			}
			continue
		}

		artist := cleanYoutubeChannelName(e.ChannelName)
		if artist == "" || e.VideoTitle == "" {
			continue
		}
		opts := catalog.SubmitListenOpts{
			MbzCaller:      mbzc,
			Artist:         artist,
			ArtistNames:    []string{artist},
			TrackTitle:     e.VideoTitle,
			Time:           e.Time,
			Client:         "YouTube",
			UserID:         1,
			SkipCacheImage: !cfg.FetchImagesDuringImport(),
		}
		if err := catalog.SubmitListen(ctx, store, opts); err != nil {
			l.Err(err).Str("track", e.VideoTitle).Msg("Failed to import YouTube Takeout HTML item")
			continue
		}
		imported++
		throttleFunc()
	}

	return finishImport(ctx, filename, imported)
}

// fetchVideoInfoForHtmlEntries resolves every video ID referenced by
// entries up front via the YouTube Data API, mirroring youtube.go's
// fetchVideoInfoForItems. Returns an empty map (not an error) if no key
// is configured or the lookup fails, so the caller always falls back to
// the "unknown category" handling instead of failing the whole import.
func fetchVideoInfoForHtmlEntries(ctx context.Context, apiKey string, entries []youtubeHtmlEntry) map[string]ytdata.VideoInfo {
	if apiKey == "" {
		return nil
	}
	l := logger.FromContext(ctx)
	idSet := make(map[string]struct{})
	for _, e := range entries {
		if e.VideoID == "" || !inImportTimeWindow(e.Time) {
			continue
		}
		idSet[e.VideoID] = struct{}{}
	}
	if len(idSet) == 0 {
		return nil
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	client := ytdata.New(apiKey)
	info, err := client.GetVideos(ctx, ids)
	if err != nil {
		l.Warn().Err(err).Msg("ImportYoutubeTakeoutHTMLFile: YouTube Data API lookup failed, falling back to generic video handling")
		return nil
	}
	return info
}

// importClassifiedHtmlEntry saves one entry the Data API identified as
// either a music video (recorded as a normal listen) or a non-music
// video (recorded as a Video watch), same split as the JSON path's
// importVideoItem.
func importClassifiedHtmlEntry(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, e youtubeHtmlEntry, info ytdata.VideoInfo) int {
	l := logger.FromContext(ctx)

	if info.IsMusic() {
		artist := cleanYoutubeChannelName(info.ChannelName)
		if artist == "" {
			artist = cleanYoutubeChannelName(e.ChannelName)
		}
		title := info.Title
		if title == "" {
			title = e.VideoTitle
		}
		if artist == "" || title == "" {
			return 0
		}
		opts := catalog.SubmitListenOpts{
			MbzCaller:      mbzc,
			Artist:         artist,
			ArtistNames:    []string{artist},
			TrackTitle:     title,
			Time:           e.Time,
			Client:         "YouTube",
			UserID:         1,
			SkipCacheImage: !cfg.FetchImagesDuringImport(),
		}
		if err := catalog.SubmitListen(ctx, store, opts); err != nil {
			l.Err(err).Str("track", title).Msg("Failed to import YouTube Takeout HTML music video")
			return 0
		}
		return 1
	}

	format := "video"
	if info.IsShort() {
		format = "short"
	}
	video, err := store.UpsertVideo(ctx, db.SaveVideoOpts{
		YoutubeID:   e.VideoID,
		Title:       info.Title,
		ChannelID:   info.ChannelID,
		ChannelName: info.ChannelName,
		Thumbnail:   info.Thumbnail,
		Category:    info.Category,
		Format:      format,
	})
	if err != nil {
		l.Err(err).Str("video_id", e.VideoID).Msg("Failed to save YouTube Takeout HTML video")
		return 0
	}
	if err := store.SaveVideoWatch(ctx, db.SaveVideoWatchOpts{
		VideoID: video.ID,
		Time:    e.Time,
		UserID:  1,
		Client:  "YouTube",
	}); err != nil {
		l.Err(err).Str("video_id", e.VideoID).Msg("Failed to save YouTube Takeout HTML video watch")
		return 0
	}
	return 1
}

// importGenericVideoWatch saves an entry the Data API couldn't classify
// (no API key configured, lookup failed, or the video is no longer
// available) as a Video watch with whatever the HTML export itself gave
// us - no category, no thumbnail.
func importGenericVideoWatch(ctx context.Context, store importStore, e youtubeHtmlEntry) int {
	l := logger.FromContext(ctx)
	video, err := store.UpsertVideo(ctx, db.SaveVideoOpts{
		YoutubeID:   e.VideoID,
		Title:       e.VideoTitle,
		ChannelName: cleanYoutubeChannelName(e.ChannelName),
		Format:      "video",
	})
	if err != nil {
		l.Err(err).Str("video_id", e.VideoID).Msg("Failed to save YouTube Takeout HTML video")
		return 0
	}
	if err := store.SaveVideoWatch(ctx, db.SaveVideoWatchOpts{
		VideoID: video.ID,
		Time:    e.Time,
		UserID:  1,
		Client:  "YouTube",
	}); err != nil {
		l.Err(err).Str("video_id", e.VideoID).Msg("Failed to save YouTube Takeout HTML video watch")
		return 0
	}
	return 1
}

// parseYoutubeHtmlEntries extracts every history row from a Takeout
// watch-history.html document. Each row is a
// `content-cell ... body-1` div containing a video link, an optional
// channel link, and a trailing "DD.MM.YYYY, HH:MM:SS TZ" timestamp line.
// Rows with no video link at all (e.g. "Hat Anzeigen auf der
// YouTube-Startseite angesehen" - homepage ad impressions with nothing to
// attach a watch to) are skipped.
func parseYoutubeHtmlEntries(document string) []youtubeHtmlEntry {
	cells := youtubeHtmlEntryRe.FindAllStringSubmatch(document, -1)
	entries := make([]youtubeHtmlEntry, 0, len(cells))
	for _, cell := range cells {
		body := cell[1]
		links := youtubeHtmlLinkRe.FindAllStringSubmatch(body, -1)
		if len(links) == 0 {
			continue
		}
		videoID := ytdata.VideoIDFromURL(html.UnescapeString(links[0][1]))
		if videoID == "" {
			continue
		}
		videoTitle := html.UnescapeString(links[0][2])
		channelName := ""
		if len(links) >= 2 {
			channelName = html.UnescapeString(links[1][2])
		}

		t, ok := parseYoutubeHtmlTimestamp(body)
		if !ok {
			continue
		}

		entries = append(entries, youtubeHtmlEntry{
			VideoID:     videoID,
			VideoTitle:  videoTitle,
			ChannelName: channelName,
			Time:        t,
		})
	}
	return entries
}

// parseYoutubeHtmlTimestamp parses the trailing "DD.MM.YYYY, HH:MM:SS TZ"
// line Takeout's German-locale HTML export puts at the end of every
// entry. Unrecognized timezone abbreviations fall back to UTC rather
// than failing the entry outright.
func parseYoutubeHtmlTimestamp(body string) (time.Time, bool) {
	m := youtubeHtmlTimeRe.FindStringSubmatch(body)
	if m == nil {
		return time.Time{}, false
	}
	offset, ok := youtubeHtmlTzOffsets[m[5]]
	if !ok {
		offset = 0
	}
	loc := time.FixedZone(m[5], offset)
	t, err := time.ParseInLocation("02.01.2006, 15:04:05", m[1]+"."+m[2]+"."+m[3]+", "+m[4], loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
