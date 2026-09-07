package db

import (
	"time"

	"github.com/gabehf/koito/internal/models"
	"github.com/google/uuid"
)

type InformationSource string

const (
	InformationSourceInferred     InformationSource = "Inferred"
	InformationSourceMusicBrainz  InformationSource = "MusicBrainz"
	InformationSourceUserProvided InformationSource = "User"
)

type ListenActivityItem struct {
	Start   time.Time `json:"start_time"`
	Listens int64     `json:"listens"`
}

type PaginatedResponse[T any] struct {
	Items        []T   `json:"items"`
	TotalCount   int64 `json:"total_record_count"`
	ItemsPerPage int32 `json:"items_per_page"`
	HasNextPage  bool  `json:"has_next_page"`
	CurrentPage  int32 `json:"current_page"`
}

type RankedItem[T any] struct {
	Item T     `json:"item"`
	Rank int64 `json:"rank"`
}

type VideoCategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// VideoChannelRank is one row of the "most watched YouTube channels" card.
// Thumbnail is the thumbnail of the channel's most-watched video in the
// period, standing in for a channel avatar (which isn't fetched/stored).
type VideoChannelRank struct {
	Rank        int    `json:"rank"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	WatchCount  int64  `json:"watch_count"`
	Thumbnail   string `json:"thumbnail"`
}

type VideoFormatSplit struct {
	Longform  int64 `json:"longform"`
	Shortform int64 `json:"shortform"`
}

// VideoDailyFormatCount is one (day, format) count, as returned raw from
// the database before being bucketed into chart steps by the handler.
type VideoDailyFormatCount struct {
	Date   time.Time
	Format string
	Count  int64
}

type VideoActivityItem struct {
	Start     time.Time `json:"start_time"`
	Longform  int64     `json:"longform"`
	Shortform int64     `json:"shortform"`
}

type ExportItem struct {
	ListenedAt         time.Time
	UserID             int32
	Client             *string
	TrackID            int32
	TrackMbid          *uuid.UUID
	TrackDuration      int32
	TrackAliases       []models.Alias
	ReleaseID          int32
	ReleaseMbid        *uuid.UUID
	ReleaseImage       *uuid.UUID
	ReleaseImageSource string
	VariousArtists     bool
	ReleaseAliases     []models.Alias
	Artists            []models.ArtistWithFullAliases
}

type InterestBucket struct {
	BucketStart time.Time `json:"bucket_start"`
	BucketEnd   time.Time `json:"bucket_end"`
	ListenCount int64     `json:"listen_count"`
}

// TagWeight is a single mood/genre folksonomy tag (e.g. from LastFM) with its
// relevance score.
type TagWeight struct {
	Tag    string
	Weight int
}
