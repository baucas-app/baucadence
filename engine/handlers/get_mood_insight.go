package handlers

import (
	"net/http"
	"sort"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/moodtags"
	"github.com/gabehf/koito/internal/utils"
)

// maxTracksForMoodInsight caps how many of the period's top tracks are
// pulled in for mood aggregation. Personal listening libraries are small
// enough that this comfortably covers a full period.
const maxTracksForMoodInsight = 500

type MoodInsight struct {
	Enabled   bool                 `json:"enabled"`
	Mood      string               `json:"mood"`
	Coverage  float64              `json:"coverage"`
	TopTags   []string             `json:"top_tags"`
	TopTracks []models.SimpleTrack `json:"top_tracks"`
}

type trackMoodScore struct {
	track *models.Track
	score float64
}

func GetMoodInsightHandler(store db.TrackStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		if cfg.LastFMApiKey() == "" {
			utils.WriteJSON(w, http.StatusOK, MoodInsight{Enabled: false})
			return
		}

		tf := TimeframeFromRequest(r)
		opts := db.GetItemsOpts{Limit: maxTracksForMoodInsight, Page: 1, Timeframe: tf}

		topTracks, err := store.GetTopTracksPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetMoodInsightHandler: Failed to retrieve top tracks")
			utils.WriteError(w, "failed to get mood insight", http.StatusBadRequest)
			return
		}

		ids := make([]int32, 0, len(topTracks.Items))
		for _, item := range topTracks.Items {
			ids = append(ids, item.Item.ID)
		}

		tagsByTrack, err := store.GetTagsForTracks(ctx, ids)
		if err != nil {
			l.Err(err).Msg("GetMoodInsightHandler: Failed to retrieve track tags")
			utils.WriteError(w, "failed to get mood insight", http.StatusBadRequest)
			return
		}

		insight := computeMoodInsight(topTracks.Items, tagsByTrack)
		utils.WriteJSON(w, http.StatusOK, insight)
	}
}

func computeMoodInsight(items []db.RankedItem[*models.Track], tagsByTrack map[int32][]db.TagWeight) MoodInsight {
	var totalListens, analyzedListens int64
	categoryScore := map[string]float64{}
	categoryTags := map[string]map[string]float64{}
	categoryTracks := map[string][]trackMoodScore{}

	for _, item := range items {
		track := item.Item
		totalListens += track.ListenCount

		tags := tagsByTrack[track.ID]
		if len(tags) == 0 {
			continue
		}
		analyzedListens += track.ListenCount

		perTrackScore := map[string]float64{}
		for _, tw := range tags {
			category := moodtags.Categorize(tw.Tag)
			if category == "" {
				continue
			}
			contribution := float64(track.ListenCount) * float64(tw.Weight)
			categoryScore[category] += contribution
			perTrackScore[category] += contribution

			if categoryTags[category] == nil {
				categoryTags[category] = map[string]float64{}
			}
			categoryTags[category][tw.Tag] += contribution
		}
		for category, score := range perTrackScore {
			categoryTracks[category] = append(categoryTracks[category], trackMoodScore{track: track, score: score})
		}
	}

	insight := MoodInsight{Enabled: true, TopTags: []string{}, TopTracks: []models.SimpleTrack{}}
	if totalListens > 0 {
		insight.Coverage = float64(analyzedListens) / float64(totalListens)
	}

	winner := ""
	var winnerScore float64
	for category, score := range categoryScore {
		if score > winnerScore {
			winner = category
			winnerScore = score
		}
	}

	// Too little tagged data to make a confident call
	if winner == "" || insight.Coverage < 0.2 {
		return insight
	}

	insight.Mood = winner
	insight.TopTags = topTagNames(categoryTags[winner], 3)
	insight.TopTracks = topTracks(categoryTracks[winner], 3)

	return insight
}

func topTagNames(scores map[string]float64, limit int) []string {
	type entry struct {
		tag   string
		score float64
	}
	entries := make([]entry, 0, len(scores))
	for tag, score := range scores {
		entries = append(entries, entry{tag, score})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].score > entries[j].score })

	if len(entries) > limit {
		entries = entries[:limit]
	}
	tags := make([]string, len(entries))
	for i, e := range entries {
		tags[i] = e.tag
	}
	return tags
}

func topTracks(scores []trackMoodScore, limit int) []models.SimpleTrack {
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

	if len(scores) > limit {
		scores = scores[:limit]
	}
	tracks := make([]models.SimpleTrack, len(scores))
	for i, ts := range scores {
		tracks[i] = models.SimpleTrack{
			ID:      ts.track.ID,
			Title:   ts.track.Title,
			Artists: ts.track.Artists,
			Image:   ts.track.Image,
		}
	}
	return tracks
}
