package models

import "time"

// Video is a non-music YouTube video, tracked separately from the
// Artist/Album/Track catalog since it has no such structure.
type Video struct {
	ID          int32  `json:"id"`
	YoutubeID   string `json:"youtube_id"`
	Title       string `json:"title"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Thumbnail   string `json:"thumbnail"`
	Category    string `json:"category"`
	// Format is "video" (longform) or "short", based on duration at import
	// time. Never empty - defaults to "video" when unknown.
	Format string `json:"format"`
}

// VideoWatch is one watch event of a Video, the video equivalent of a Listen.
type VideoWatch struct {
	Time  time.Time `json:"time"`
	Video Video     `json:"video"`
}
