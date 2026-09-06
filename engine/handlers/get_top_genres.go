package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/utils"
)

// maxTracksForGenreInsight caps how many of the period's top tracks are
// pulled in for genre aggregation, mirroring the mood insight tradeoff:
// personal listening libraries are small enough that this comfortably
// covers a full period.
const maxTracksForGenreInsight = 500

const defaultTopGenresLimit = 5
const maxTopGenresLimit = 20

type TopGenresResponse struct {
	Enabled bool        `json:"enabled"`
	Genres  []GenreRank `json:"genres"`
}

type GenreRank struct {
	Rank        int    `json:"rank"`
	Name        string `json:"name"`
	ListenCount int64  `json:"listen_count"`
}

type genreAgg struct {
	score       float64
	listenCount int64
}

func GetTopGenresHandler(store db.TrackStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		if cfg.LastFMApiKey() == "" {
			utils.WriteJSON(w, http.StatusOK, TopGenresResponse{Enabled: false, Genres: []GenreRank{}})
			return
		}

		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			limit = defaultTopGenresLimit
		}
		if limit > maxTopGenresLimit {
			limit = maxTopGenresLimit
		}

		tf := TimeframeFromRequest(r)
		opts := db.GetItemsOpts{Limit: maxTracksForGenreInsight, Page: 1, Timeframe: tf}

		topTracks, err := store.GetTopTracksPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetTopGenresHandler: Failed to retrieve top tracks")
			utils.WriteError(w, "failed to get top genres", http.StatusBadRequest)
			return
		}

		ids := make([]int32, 0, len(topTracks.Items))
		for _, item := range topTracks.Items {
			ids = append(ids, item.Item.ID)
		}

		tagsByTrack, err := store.GetTagsForTracks(ctx, ids)
		if err != nil {
			l.Err(err).Msg("GetTopGenresHandler: Failed to retrieve track tags")
			utils.WriteError(w, "failed to get top genres", http.StatusBadRequest)
			return
		}

		genres := computeTopGenres(topTracks.Items, tagsByTrack, limit)
		utils.WriteJSON(w, http.StatusOK, TopGenresResponse{Enabled: true, Genres: genres})
	}
}

func computeTopGenres(items []db.RankedItem[*models.Track], tagsByTrack map[int32][]db.TagWeight, limit int) []GenreRank {
	aggs := map[string]genreAgg{}

	for _, item := range items {
		track := item.Item
		for _, tw := range tagsByTrack[track.ID] {
			agg := aggs[tw.Tag]
			agg.score += float64(track.ListenCount) * float64(tw.Weight)
			agg.listenCount += track.ListenCount
			aggs[tw.Tag] = agg
		}
	}

	type entry struct {
		tag string
		genreAgg
	}
	entries := make([]entry, 0, len(aggs))
	for tag, agg := range aggs {
		entries = append(entries, entry{tag, agg})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].score > entries[j].score })

	if len(entries) > limit {
		entries = entries[:limit]
	}

	genres := make([]GenreRank, len(entries))
	for i, e := range entries {
		genres[i] = GenreRank{Rank: i + 1, Name: e.tag, ListenCount: e.listenCount}
	}
	return genres
}
