package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/models"
)

func (s *Sqlite) UpsertVideo(ctx context.Context, opts db.SaveVideoOpts) (*models.Video, error) {
	if opts.YoutubeID == "" {
		return nil, errors.New("UpsertVideo: required parameter YoutubeID missing")
	}
	format := opts.Format
	if format == "" {
		format = "video"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO videos (youtube_id, title, channel_id, channel_name, thumbnail, category, format)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(youtube_id) DO UPDATE SET
			title=excluded.title,
			channel_id=excluded.channel_id,
			channel_name=excluded.channel_name,
			thumbnail=excluded.thumbnail,
			category=excluded.category,
			format=excluded.format`,
		opts.YoutubeID, opts.Title, opts.ChannelID, opts.ChannelName, opts.Thumbnail, opts.Category, format,
	)
	if err != nil {
		return nil, fmt.Errorf("UpsertVideo: %w", err)
	}

	v := &models.Video{}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, youtube_id, title, channel_id, channel_name, thumbnail, category, format FROM videos WHERE youtube_id = ?`,
		opts.YoutubeID,
	)
	if err := row.Scan(&v.ID, &v.YoutubeID, &v.Title, &v.ChannelID, &v.ChannelName, &v.Thumbnail, &v.Category, &v.Format); err != nil {
		return nil, fmt.Errorf("UpsertVideo: %w", err)
	}
	return v, nil
}

func (s *Sqlite) SaveVideoWatch(ctx context.Context, opts db.SaveVideoWatchOpts) error {
	if opts.VideoID == 0 {
		return errors.New("SaveVideoWatch: required parameter VideoID missing")
	}
	if opts.Time.IsZero() {
		opts.Time = time.Now()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO video_watches (video_id, watched_at, user_id, client) VALUES (?,?,?,?)`,
		opts.VideoID, opts.Time.Unix(), opts.UserID, opts.Client,
	)
	return err
}

func (s *Sqlite) GetVideoWatchesPaginated(ctx context.Context, opts db.GetItemsOpts) (*db.PaginatedResponse[*models.VideoWatch], error) {
	if opts.Limit == 0 {
		opts.Limit = defaultItemsPerPage
	}
	offset := (opts.Page - 1) * opts.Limit

	whereClause := ""
	args := []any{}
	if opts.VideoFormat != "" {
		whereClause = "WHERE v.format = ?"
		args = append(args, opts.VideoFormat)
	}

	var count int64
	countQuery := "SELECT COUNT(*) FROM video_watches vw JOIN videos v ON vw.video_id = v.id " + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("GetVideoWatchesPaginated: %w", err)
	}

	rowsQuery := `
		SELECT vw.watched_at, v.id, v.youtube_id, v.title, v.channel_id, v.channel_name, v.thumbnail, v.category, v.format
		FROM video_watches vw
		JOIN videos v ON vw.video_id = v.id ` + whereClause + `
		ORDER BY vw.watched_at DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, rowsQuery, append(args, opts.Limit, offset)...)
	if err != nil {
		return nil, fmt.Errorf("GetVideoWatchesPaginated: %w", err)
	}
	defer rows.Close()

	items := make([]*models.VideoWatch, 0)
	for rows.Next() {
		var watchedAt int64
		vw := &models.VideoWatch{}
		if err := rows.Scan(&watchedAt, &vw.Video.ID, &vw.Video.YoutubeID, &vw.Video.Title,
			&vw.Video.ChannelID, &vw.Video.ChannelName, &vw.Video.Thumbnail, &vw.Video.Category, &vw.Video.Format); err != nil {
			return nil, fmt.Errorf("GetVideoWatchesPaginated: %w", err)
		}
		vw.Time = time.Unix(watchedAt, 0).UTC()
		items = append(items, vw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetVideoWatchesPaginated: %w", err)
	}

	return &db.PaginatedResponse[*models.VideoWatch]{
		Items:        items,
		TotalCount:   count,
		ItemsPerPage: int32(opts.Limit),
		HasNextPage:  int64(offset+len(items)) < count,
		CurrentPage:  int32(opts.Page),
	}, nil
}

func (s *Sqlite) GetVideoCategoryCounts(ctx context.Context, timeframe db.Timeframe) ([]db.VideoCategoryCount, error) {
	t1, t2 := db.TimeframeToTimeRange(timeframe)
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.category, COUNT(*) as cnt
		FROM video_watches vw
		JOIN videos v ON vw.video_id = v.id
		WHERE vw.watched_at BETWEEN ? AND ?
		GROUP BY v.category
		ORDER BY cnt DESC`,
		t1.Unix(), t2.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("GetVideoCategoryCounts: %w", err)
	}
	defer rows.Close()

	counts := make([]db.VideoCategoryCount, 0)
	for rows.Next() {
		var c db.VideoCategoryCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			return nil, fmt.Errorf("GetVideoCategoryCounts: %w", err)
		}
		counts = append(counts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetVideoCategoryCounts: %w", err)
	}
	return counts, nil
}
