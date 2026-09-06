package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/models"
)

// GetTracksMissingTags returns up to 20 tracks (ordered by id ascending,
// starting after `from`) that have not yet been synced with LastFM mood tags.
func (s *Sqlite) GetTracksMissingTags(ctx context.Context, from int32) ([]*models.Track, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title
		FROM tracks_with_title
		WHERE tags_synced_at IS NULL AND id > ?
		ORDER BY id ASC LIMIT 20`,
		from)
	if err != nil {
		return nil, fmt.Errorf("GetTracksMissingTags: %w", err)
	}
	defer rows.Close()

	var tracks []*models.Track
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.Title); err != nil {
			return nil, fmt.Errorf("GetTracksMissingTags: scan: %w", err)
		}
		tracks = append(tracks, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetTracksMissingTags: %w", err)
	}

	for _, t := range tracks {
		artists, err := s.artistsForTrack(ctx, t.ID)
		if err != nil {
			return nil, fmt.Errorf("GetTracksMissingTags: artistsForTrack: %w", err)
		}
		t.Artists = artists
	}

	return tracks, nil
}

// SetTrackTags replaces the stored tags for a track and marks it as synced,
// even if the resulting tag list is empty (so it isn't retried forever).
func (s *Sqlite) SetTrackTags(ctx context.Context, id int32, tags []db.TagWeight) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SetTrackTags: BeginTx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM track_tags WHERE track_id = ?`, id); err != nil {
		return fmt.Errorf("SetTrackTags: delete existing: %w", err)
	}

	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO track_tags (track_id, tag, weight) VALUES (?, ?, ?)`,
			id, tag.Tag, tag.Weight); err != nil {
			return fmt.Errorf("SetTrackTags: insert: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE tracks SET tags_synced_at = ? WHERE id = ?`, time.Now().Unix(), id); err != nil {
		return fmt.Errorf("SetTrackTags: mark synced: %w", err)
	}

	return tx.Commit()
}

// GetTagsForTracks batch-loads stored tags for a set of tracks.
func (s *Sqlite) GetTagsForTracks(ctx context.Context, ids []int32) (map[int32][]db.TagWeight, error) {
	result := make(map[int32][]db.TagWeight)
	if len(ids) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT track_id, tag, weight FROM track_tags WHERE track_id IN (%s)`,
		strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("GetTagsForTracks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var trackID int32
		var tag db.TagWeight
		if err := rows.Scan(&trackID, &tag.Tag, &tag.Weight); err != nil {
			return nil, fmt.Errorf("GetTagsForTracks: scan: %w", err)
		}
		result[trackID] = append(result[trackID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetTagsForTracks: %w", err)
	}

	return result, nil
}
