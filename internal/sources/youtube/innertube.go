package youtube

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// These are not secrets: they are the same public, non-sensitive API keys
// embedded in every browser's copy of youtube.com / music.youtube.com page
// source, used only to identify which web client is calling the internal
// "InnerTube" API. Every unofficial YouTube client (yt-dlp, ytmusicapi,
// youtubei.js, ...) uses the same constants.
const (
	youtubeHost       = "https://www.youtube.com"
	youtubeAPIKey     = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	youtubeClientName = "WEB"
	youtubeClientVer  = "2.20240101.00.00"
	youtubeHistoryID  = "FEhistory"

	musicHost       = "https://music.youtube.com"
	musicAPIKey     = "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30"
	musicClientName = "WEB_REMIX"
	musicClientVer  = "1.20240101.01.00"
	musicHistoryID  = "FEmusic_history"
)

// innerTubeClient makes authenticated requests to Google's internal
// "InnerTube" API using a browser session cookie. This is not an official,
// documented API: response shapes are undocumented and may change without
// notice. Treat this client as best-effort.
type innerTubeClient struct {
	cookie     string
	httpClient *http.Client
}

func newInnerTubeClient(cookie string) *innerTubeClient {
	return &innerTubeClient{
		cookie:     cookie,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// browse calls the /youtubei/v1/browse endpoint for the given surface
// (YouTube or YouTube Music) and returns the raw decoded JSON tree.
func (c *innerTubeClient) browse(ctx context.Context, host, apiKey, clientName, clientVersion, browseID string) (map[string]any, error) {
	authHeader, err := c.sapisidHash(host)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth header: %w", err)
	}

	body := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    clientName,
				"clientVersion": clientVersion,
				"hl":            "en",
				"gl":            "US",
			},
		},
		"browseId": browseID,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/youtubei/v1/browse?key=%s", host, apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Origin", host)
	req.Header.Set("X-Origin", host)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("innertube browse returned status %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	var parsed map[string]any
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode innertube response: %w", err)
	}
	return parsed, nil
}

// sapisidHash builds the "SAPISIDHASH" Authorization header Google's web
// clients use to authenticate InnerTube requests with a plain session
// cookie (no OAuth). See: https://stackoverflow.com/a/32065323 for the
// general technique, used identically by every unofficial YouTube client.
func (c *innerTubeClient) sapisidHash(origin string) (string, error) {
	sapisid := firstCookieValue(c.cookie, "__Secure-3PAPISID", "__Secure-1PAPISID", "SAPISID")
	if sapisid == "" {
		return "", fmt.Errorf("no SAPISID-like cookie found; the stored YouTube cookie is missing or expired")
	}
	ts := time.Now().Unix()
	toHash := fmt.Sprintf("%d %s %s", ts, sapisid, origin)
	sum := sha1.Sum([]byte(toHash))
	return fmt.Sprintf("SAPISIDHASH %d_%x", ts, sum), nil
}

func firstCookieValue(cookieHeader string, names ...string) string {
	for _, name := range names {
		if v := cookieValue(cookieHeader, name); v != "" {
			return v
		}
	}
	return ""
}

func cookieValue(cookieHeader, name string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == name {
			return kv[1]
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// findRenderers walks an arbitrary decoded JSON tree (maps/slices) and
// collects every object found under a key named renderKey, regardless of
// how deeply or where it is nested. This is deliberately more tolerant of
// YouTube changing the surrounding structure than hardcoding exact paths,
// at the cost of potentially matching unrelated items with the same
// renderer name (callers should validate the fields they need are present).
func findRenderers(node any, renderKey string, out *[]map[string]any) {
	switch v := node.(type) {
	case map[string]any:
		if renderer, ok := v[renderKey]; ok {
			if m, ok := renderer.(map[string]any); ok {
				*out = append(*out, m)
			}
		}
		for _, child := range v {
			findRenderers(child, renderKey, out)
		}
	case []any:
		for _, child := range v {
			findRenderers(child, renderKey, out)
		}
	}
}

// firstRunText extracts the first text run's string from a common
// InnerTube "{ runs: [{ text: '...' }] }" or "{ simpleText: '...' }" shape
// found at path (a sequence of map keys / one array index step is not
// supported here; callers navigate arrays manually).
func firstRunText(node map[string]any, key string) string {
	val, ok := node[key]
	if !ok {
		return ""
	}
	obj, ok := val.(map[string]any)
	if !ok {
		return ""
	}
	if simple, ok := obj["simpleText"].(string); ok {
		return simple
	}
	runs, ok := obj["runs"].([]any)
	if !ok || len(runs) == 0 {
		return ""
	}
	first, ok := runs[0].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := first["text"].(string)
	return text
}
