// package moodtags fetches mood/genre folksonomy tags for tracks from the
// Last.fm API and classifies them into a small set of mood categories.
package moodtags

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/queue"
)

const lastFMApiBaseUrl = "http://ws.audioscrobbler.com/2.0/"

type Client struct {
	apiKey       string
	baseUrl      string
	userAgent    string
	requestQueue *queue.RequestQueue
}

// NewClient builds a LastFM client for the given API key. The key is
// passed in explicitly (see ResolveApiKey) rather than read from cfg here,
// since it may come from the Settings UI instead of the environment.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:       apiKey,
		baseUrl:      lastFMApiBaseUrl,
		userAgent:    cfg.UserAgent(),
		requestQueue: queue.NewRequestQueue(5, 5),
	}
}

func (c *Client) Shutdown() {
	c.requestQueue.Shutdown()
}

// LastFM JSON wraps the tag list under "toptags". The "count" field is
// documented as a string but LastFM's API inconsistently returns it as a
// bare JSON number for some tracks, so it's parsed through flexibleCount
// rather than a plain string to tolerate both.
type lastFMTag struct {
	Name  string        `json:"name"`
	Count flexibleCount `json:"count"`
}

// flexibleCount unmarshals a LastFM count field that may arrive as either
// a JSON string ("123") or a bare JSON number (123).
type flexibleCount int

func (f *flexibleCount) UnmarshalJSON(data []byte) error {
	var asInt int
	if err := json.Unmarshal(data, &asInt); err == nil {
		*f = flexibleCount(asInt)
		return nil
	}
	var asStr string
	if err := json.Unmarshal(data, &asStr); err != nil {
		return fmt.Errorf("flexibleCount: %w", err)
	}
	n, err := strconv.Atoi(asStr)
	if err != nil {
		*f = 0
		return nil
	}
	*f = flexibleCount(n)
	return nil
}

type lastFMTopTagsResponse struct {
	Toptags struct {
		Tag []lastFMTag `json:"tag"`
	} `json:"toptags"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

func (c *Client) queue(ctx context.Context, req *http.Request) ([]byte, error) {
	l := logger.FromContext(ctx)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resultChan := c.requestQueue.Enqueue(func(client *http.Client, done chan<- queue.RequestResult) {
		resp, err := client.Do(req)
		if err != nil {
			l.Debug().Err(err).Str("url", req.URL.String()).Msg("Failed to contact LastFM")
			done <- queue.RequestResult{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			done <- queue.RequestResult{Err: fmt.Errorf("received server error from LastFM: %s", resp.Status)}
			return
		}

		body, err := io.ReadAll(resp.Body)
		done <- queue.RequestResult{Body: body, Err: err}
	})

	result := <-resultChan
	return result.Body, result.Err
}

func (c *Client) getTopTags(ctx context.Context, method string, params url.Values) ([]db.TagWeight, error) {
	params.Set("method", method)
	params.Set("api_key", c.apiKey)
	params.Set("format", "json")

	reqUrl, _ := url.Parse(c.baseUrl)
	reqUrl.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("getTopTags: %w", err)
	}

	body, err := c.queue(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("getTopTags: %w", err)
	}

	var resp lastFMTopTagsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("getTopTags: unmarshal: %w", err)
	}
	if resp.Error != 0 {
		// Not found / no tags is a normal outcome (e.g. error 6), not a hard failure
		return nil, nil
	}

	tags := make([]db.TagWeight, 0, len(resp.Toptags.Tag))
	for _, t := range resp.Toptags.Tag {
		weight := int(t.Count)
		if t.Name == "" || weight <= 0 {
			continue
		}
		tags = append(tags, db.TagWeight{Tag: t.Name, Weight: weight})
	}
	return tags, nil
}

// GetTrackTopTags fetches community tags for a specific track.
func (c *Client) GetTrackTopTags(ctx context.Context, artist, title string) ([]db.TagWeight, error) {
	params := url.Values{}
	params.Set("artist", artist)
	params.Set("track", title)
	params.Set("autocorrect", "1")
	return c.getTopTags(ctx, "track.getTopTags", params)
}

// GetArtistTopTags fetches community tags for an artist, used as a fallback
// when a track has no tags of its own.
func (c *Client) GetArtistTopTags(ctx context.Context, artist string) ([]db.TagWeight, error) {
	params := url.Values{}
	params.Set("artist", artist)
	params.Set("autocorrect", "1")
	return c.getTopTags(ctx, "artist.getTopTags", params)
}

// GetTags returns tags for a track, falling back to the artist's tags if the
// track itself has none.
func (c *Client) GetTags(ctx context.Context, artist, title string) ([]db.TagWeight, error) {
	tags, err := c.GetTrackTopTags(ctx, artist, title)
	if err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		return tags, nil
	}
	return c.GetArtistTopTags(ctx, artist)
}
