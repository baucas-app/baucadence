// Package ytdata is a thin client for the public YouTube Data API v3
// "videos" endpoint. It is used only to enrich a video ID with metadata
// (title, channel, thumbnail, category) via a plain API key - this is
// public metadata and has always been available via the API.
//
// This is deliberately unrelated to a user's personal watch history: the
// Data API has had no endpoint for that since ~2016 (the watchHistory
// playlist has returned an empty placeholder since then, and there is no
// OAuth scope for it). Watch history itself only ever comes from a Google
// Takeout export; this package just fills in the details for videos found
// in that export.
package ytdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const videosEndpoint = "https://www.googleapis.com/youtube/v3/videos"

// maxIDsPerRequest is the batch size limit the videos.list endpoint enforces.
const maxIDsPerRequest = 50

// categoryNames maps YouTube's standard video category IDs to a human
// readable name. Category 10 is "Music" and is the only one this package's
// caller currently branches on, but the rest are included so imported
// videos show a meaningful category rather than a bare numeric ID.
var categoryNames = map[string]string{
	"1": "Film & Animation", "2": "Autos & Vehicles", "10": "Music",
	"15": "Pets & Animals", "17": "Sports", "18": "Short Movies",
	"19": "Travel & Events", "20": "Gaming", "21": "Videoblogging",
	"22": "People & Blogs", "23": "Comedy", "24": "Entertainment",
	"25": "News & Politics", "26": "Howto & Style", "27": "Education",
	"28": "Science & Technology", "29": "Nonprofits & Activism", "30": "Movies",
}

// MusicCategoryID is YouTube's video category ID for "Music".
const MusicCategoryID = "10"

// shortMaxDurationSeconds is the classic Shorts duration cutoff. YouTube
// has since raised the *eligibility* limit for creators to up to 3
// minutes, but 60s is still the reliable signal for "this was almost
// certainly watched as a Short" without a false-positive risk on regular
// short-form videos.
const shortMaxDurationSeconds = 60

type VideoInfo struct {
	ID              string
	Title           string
	ChannelID       string
	ChannelName     string
	Thumbnail       string
	CategoryID      string
	Category        string
	DurationSeconds int
}

func (v VideoInfo) IsMusic() bool {
	return v.CategoryID == MusicCategoryID
}

// IsShort reports whether this video was almost certainly a YouTube
// Short, based on duration. 0 duration (unknown/live) is never a Short.
func (v VideoInfo) IsShort() bool {
	return v.DurationSeconds > 0 && v.DurationSeconds <= shortMaxDurationSeconds
}

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey, httpClient: &http.Client{}}
}

// GetVideos looks up metadata for the given YouTube video IDs, batching
// requests in groups of up to 50 (the API's limit per call). Videos that
// no longer exist (deleted/private) are simply absent from the result.
func (c *Client) GetVideos(ctx context.Context, ids []string) (map[string]VideoInfo, error) {
	result := make(map[string]VideoInfo, len(ids))
	for i := 0; i < len(ids); i += maxIDsPerRequest {
		end := min(i+maxIDsPerRequest, len(ids))
		batch, err := c.getVideosBatch(ctx, ids[i:end])
		if err != nil {
			return nil, err
		}
		for id, info := range batch {
			result[id] = info
		}
	}
	return result, nil
}

func (c *Client) getVideosBatch(ctx context.Context, ids []string) (map[string]VideoInfo, error) {
	q := url.Values{}
	q.Set("part", "snippet,contentDetails")
	q.Set("id", strings.Join(ids, ","))
	q.Set("key", c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videosEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("ytdata: GetVideos: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ytdata: GetVideos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ytdata: GetVideos: unexpected status %d", resp.StatusCode)
	}

	var parsed struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelID    string `json:"channelId"`
				ChannelTitle string `json:"channelTitle"`
				CategoryID   string `json:"categoryId"`
				Thumbnails   struct {
					Medium struct {
						URL string `json:"url"`
					} `json:"medium"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("ytdata: GetVideos: %w", err)
	}

	out := make(map[string]VideoInfo, len(parsed.Items))
	for _, item := range parsed.Items {
		out[item.ID] = VideoInfo{
			ID:              item.ID,
			Title:           item.Snippet.Title,
			ChannelID:       item.Snippet.ChannelID,
			ChannelName:     item.Snippet.ChannelTitle,
			Thumbnail:       item.Snippet.Thumbnails.Medium.URL,
			CategoryID:      item.Snippet.CategoryID,
			Category:        categoryName(item.Snippet.CategoryID),
			DurationSeconds: parseISO8601DurationSeconds(item.ContentDetails.Duration),
		}
	}
	return out, nil
}

func categoryName(id string) string {
	if name, ok := categoryNames[id]; ok {
		return name
	}
	return id
}

var iso8601DurationRe = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseISO8601DurationSeconds parses the subset of ISO 8601 durations the
// Data API actually returns for videos (PT#H#M#S, time-only - videos don't
// have a day/month/year component). Returns 0 for anything it can't parse,
// e.g. a livestream's "P0D".
func parseISO8601DurationSeconds(d string) int {
	m := iso8601DurationRe.FindStringSubmatch(d)
	if m == nil {
		return 0
	}
	hours, _ := strconv.Atoi(m[1])
	minutes, _ := strconv.Atoi(m[2])
	seconds, _ := strconv.Atoi(m[3])
	return hours*3600 + minutes*60 + seconds
}

// VideoIDFromURL extracts the "v" video ID from a standard YouTube watch
// URL or a youtu.be short link, as found in a Takeout "titleUrl" field.
func VideoIDFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(u.Hostname(), "youtu.be") {
		return strings.Trim(u.Path, "/")
	}
	return u.Query().Get("v")
}
