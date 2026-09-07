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

// GetTopVideoChannels ranks channels by watch count in the timeframe. A
// second query per channel picks that channel's most-watched video in the
// period as a representative thumbnail, since no channel-avatar data is
// fetched/stored - fine for the small (top 5-10) list this powers.
func (s *Sqlite) GetTopVideoChannels(ctx context.Context, timeframe db.Timeframe, limit int) ([]db.VideoChannelRank, error) {
	t1, t2 := db.TimeframeToTimeRange(timeframe)
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.channel_name, v.channel_id, COUNT(*) as cnt
		FROM video_watches vw
		JOIN videos v ON vw.video_id = v.id
		WHERE vw.watched_at BETWEEN ? AND ? AND v.channel_name != ''
		GROUP BY v.channel_name
		ORDER BY cnt DESC
		LIMIT ?`,
		t1.Unix(), t2.Unix(), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("GetTopVideoChannels: %w", err)
	}

	channels := make([]db.VideoChannelRank, 0)
	for rows.Next() {
		var c db.VideoChannelRank
		if err := rows.Scan(&c.ChannelName, &c.ChannelID, &c.WatchCount); err != nil {
			rows.Close()
			return nil, fmt.Errorf("GetTopVideoChannels: %w", err)
		}
		c.Rank = len(channels) + 1
		channels = append(channels, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("GetTopVideoChannels: %w", err)
	}
	rows.Close()

	for i := range channels {
		row := s.db.QueryRowContext(ctx, `
			SELECT v.thumbnail
			FROM video_watches vw
			JOIN videos v ON vw.video_id = v.id
			WHERE v.channel_name = ? AND vw.watched_at BETWEEN ? AND ?
			GROUP BY v.id
			ORDER BY COUNT(*) DESC
			LIMIT 1`,
			channels[i].ChannelName, t1.Unix(), t2.Unix(),
		)
		// best-effort: leave Thumbnail empty if this fails, not fatal
		_ = row.Scan(&channels[i].Thumbnail)
	}

	return channels, nil
}

// GetVideoFormatSplit counts watches by format ("short" vs everything else,
// which UpsertVideo always defaults to "video") within the timeframe.
func (s *Sqlite) GetVideoFormatSplit(ctx context.Context, timeframe db.Timeframe) (db.VideoFormatSplit, error) {
	t1, t2 := db.TimeframeToTimeRange(timeframe)
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.format, COUNT(*)
		FROM video_watches vw
		JOIN videos v ON vw.video_id = v.id
		WHERE vw.watched_at BETWEEN ? AND ?
		GROUP BY v.format`,
		t1.Unix(), t2.Unix(),
	)
	if err != nil {
		return db.VideoFormatSplit{}, fmt.Errorf("GetVideoFormatSplit: %w", err)
	}
	defer rows.Close()

	var split db.VideoFormatSplit
	for rows.Next() {
		var format string
		var cnt int64
		if err := rows.Scan(&format, &cnt); err != nil {
			return db.VideoFormatSplit{}, fmt.Errorf("GetVideoFormatSplit: %w", err)
		}
		if format == "short" {
			split.Shortform += cnt
		} else {
			split.Longform += cnt
		}
	}
	if err := rows.Err(); err != nil {
		return db.VideoFormatSplit{}, fmt.Errorf("GetVideoFormatSplit: %w", err)
	}
	return split, nil
}

// GetVideoDailyFormatCounts returns raw per-day, per-format watch counts
// between from and to. The caller (handler layer) buckets these into
// arbitrary chart steps, mirroring how listen activity is bucketed.
func (s *Sqlite) GetVideoDailyFormatCounts(ctx context.Context, from, to time.Time) ([]db.VideoDailyFormatCount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date(vw.watched_at, 'unixepoch') as day, v.format, COUNT(*) as cnt
		FROM video_watches vw
		JOIN videos v ON vw.video_id = v.id
		WHERE vw.watched_at BETWEEN ? AND ?
		GROUP BY day, v.format
		ORDER BY day`,
		from.Unix(), to.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("GetVideoDailyFormatCounts: %w", err)
	}
	defer rows.Close()

	counts := make([]db.VideoDailyFormatCount, 0)
	for rows.Next() {
		var dayStr, format string
		var cnt int64
		if err := rows.Scan(&dayStr, &format, &cnt); err != nil {
			return nil, fmt.Errorf("GetVideoDailyFormatCounts: %w", err)
		}
		day, err := time.Parse("2006-01-02", dayStr)
		if err != nil {
			continue
		}
		counts = append(counts, db.VideoDailyFormatCount{Date: day, Format: format, Count: cnt})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetVideoDailyFormatCounts: %w", err)
	}
	return counts, nil
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
