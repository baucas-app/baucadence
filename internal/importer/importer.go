package importer

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/mbz"
)

// isYoutubeTakeoutHTML reports whether name is the HTML variant of a
// Google Takeout "YouTube and YouTube Music" watch-history export -
// "watch-history.html" in English, "Wiedergabeverlauf.html" in German
// (Takeout names the file per the account's language, unlike the JSON
// output which is always "watch-history.json" regardless of locale).
func isYoutubeTakeoutHTML(name string) bool {
	lower := strings.ToLower(name)
	if !strings.HasSuffix(lower, ".html") {
		return false
	}
	return strings.Contains(lower, "watch-history") || strings.Contains(lower, "wiedergabeverlauf")
}

// RecognizeFilename reports whether name matches one of the supported
// export file naming conventions (used both by the startup import
// directory scan and by the web upload endpoint, so a file is only ever
// saved/kept if something will actually know how to import it).
func RecognizeFilename(name string) bool {
	switch {
	case strings.Contains(name, "Streaming_History_Audio"),
		strings.Contains(name, "maloja"),
		strings.Contains(name, "recenttracks"),
		strings.Contains(name, "listenbrainz"),
		strings.Contains(name, "koito"),
		strings.Contains(name, "watch-history"),
		isYoutubeTakeoutHTML(name):
		return true
	default:
		return false
	}
}

// SourceLabel returns a short, human-readable label for the service a
// recognized export filename came from (e.g. "Spotify"), or "" if the
// filename isn't recognized.
func SourceLabel(name string) string {
	switch {
	case strings.Contains(name, "Streaming_History_Audio"):
		return "Spotify"
	case strings.Contains(name, "maloja"):
		return "Maloja"
	case strings.Contains(name, "recenttracks"):
		return "Last.fm"
	case strings.Contains(name, "listenbrainz"):
		return "ListenBrainz"
	case strings.Contains(name, "koito"):
		return "Koito"
	case isYoutubeTakeoutHTML(name), strings.Contains(name, "watch-history"):
		return "YouTube / YouTube Music"
	default:
		return ""
	}
}

// DetectAndImportFile dispatches filename to the importer matching its
// naming convention. filename must already exist in the configured
// "import" directory.
func DetectAndImportFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	l := logger.FromContext(ctx)
	switch {
	case strings.Contains(filename, "Streaming_History_Audio"):
		l.Info().Msgf("Importer: Import file %s detecting as being Spotify export", filename)
		return ImportSpotifyFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "maloja"):
		l.Info().Msgf("Importer: Import file %s detecting as being Maloja export", filename)
		return ImportMalojaFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "recenttracks"):
		l.Info().Msgf("Importer: Import file %s detecting as being ghan.nl LastFM export", filename)
		return ImportLastFMFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "listenbrainz"):
		l.Info().Msgf("Importer: Import file %s detecting as being ListenBrainz export", filename)
		return ImportListenBrainzExport(ctx, store, mbzc, filename)
	case strings.Contains(filename, "koito"):
		l.Info().Msgf("Importer: Import file %s detecting as being Koito export", filename)
		return ImportKoitoFile(ctx, store, filename)
	case isYoutubeTakeoutHTML(filename):
		l.Info().Msgf("Importer: Import file %s detecting as being a Google Takeout YouTube export (HTML)", filename)
		return ImportYoutubeTakeoutHTMLFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "watch-history"):
		l.Info().Msgf("Importer: Import file %s detecting as being a Google Takeout YouTube export", filename)
		return ImportYoutubeTakeoutFile(ctx, store, mbzc, filename)
	default:
		return fmt.Errorf("file %s not recognized as a valid import file; make sure it is valid and named correctly", filename)
	}
}

// runs after every importer
func finishImport(ctx context.Context, filename string, numImported int) error {
	l := logger.FromContext(ctx)
	_, err := os.Stat(path.Join(cfg.ConfigDir(), "import_complete"))
	if err != nil {
		err = os.Mkdir(path.Join(cfg.ConfigDir(), "import_complete"), 0744)
		if err != nil {
			l.Err(err).Msg("Failed to create import_complete dir! Import files must be removed from the import directory manually, or else the importer will run on every app start")
		}
	}
	err = os.Rename(path.Join(cfg.ConfigDir(), "import", filename), path.Join(cfg.ConfigDir(), "import_complete", filename))
	if err != nil {
		l.Err(err).Msg("Failed to move file to import_complete dir! Import files must be removed from the import directory manually, or else the importer will run on every app start")
	}
	if numImported != 0 {
		l.Info().Msgf("Finished importing %s; imported %d items", filename, numImported)
	}
	return nil
}

// from https://stackoverflow.com/a/55093788 with modification to use cfg and check for zero values
func inImportTimeWindow(check time.Time) bool {
	end, start := cfg.ImportWindow()
	if start.IsZero() && end.IsZero() {
		return true
	}
	if !start.IsZero() && end.IsZero() {
		return !check.Before(start)
	}
	if start.IsZero() && !end.IsZero() {
		return !check.After(end)
	}
	return !check.Before(start) && !check.After(end)
}
