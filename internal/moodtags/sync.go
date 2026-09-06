package moodtags

import (
	"context"
	"fmt"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
)

// BackfillTrackTags fetches Last.fm tags for every track that hasn't been
// synced yet. It is intended to be run once in its own goroutine at startup,
// the same way catalog.BackfillTrackDurationsFromMusicBrainz backfills
// durations from MusicBrainz.
func BackfillTrackTags(ctx context.Context, store db.TrackStore, client *Client) error {
	l := logger.FromContext(ctx)
	l.Info().Msg("BackfillTrackTags: Starting backfill of track mood tags from LastFM")

	var from int32 = 0

	for {
		tracks, err := store.GetTracksMissingTags(ctx, from)
		if err != nil {
			return fmt.Errorf("BackfillTrackTags: failed to fetch tracks needing tags: %w", err)
		}

		if len(tracks) == 0 {
			if from == 0 {
				l.Info().Msg("BackfillTrackTags: No tracks need tagging. Skipping backfill...")
			} else {
				l.Info().Msg("BackfillTrackTags: Backfill complete")
			}
			return nil
		}

		for _, track := range tracks {
			from = track.ID

			if len(track.Artists) == 0 {
				continue
			}
			artist := track.Artists[0].Name

			tags, err := client.GetTags(ctx, artist, track.Title)
			if err != nil {
				l.Err(err).Str("title", track.Title).Msg("BackfillTrackTags: Failed to fetch tags from LastFM")
				continue
			}

			if err := store.SetTrackTags(ctx, track.ID, tags); err != nil {
				l.Err(err).Str("title", track.Title).Msg("BackfillTrackTags: Failed to save tags")
				continue
			}

			l.Debug().Str("title", track.Title).Int("tag_count", len(tags)).
				Msg("BackfillTrackTags: Tags synced")
		}
	}
}
