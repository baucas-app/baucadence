package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/moodtags"
	"github.com/gabehf/koito/internal/utils"
	"github.com/go-chi/chi/v5"
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

type genreStore interface {
	db.TrackStore
	db.SettingsStore
}

func GetTopGenresHandler(store genreStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		lastfmKey, err := moodtags.ResolveApiKey(ctx, store)
		if err != nil {
			l.Err(err).Msg("GetTopGenresHandler: Failed to resolve LastFM API key")
			utils.WriteError(w, "failed to get top genres", http.StatusInternalServerError)
			return
		}
		if lastfmKey == "" {
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

// GenreTrack is a track that carries the clicked genre's tag, alongside how
// many of its listens counted toward that genre in the requested period.
type GenreTrack struct {
	models.SimpleTrack
	ListenCount int64 `json:"listen_count"`
}

// GetGenreTracksHandler lists which of the period's top tracks contributed
// to a given genre, for the "click a genre to see what's in it" expansion
// on the Top Genres card. It recomputes over the same top-tracks-for-period
// window GetTopGenresHandler uses, so results stay consistent with the
// ranking shown there.
func GetGenreTracksHandler(store genreStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		genre := chi.URLParam(r, "genre")
		if genre == "" {
			utils.WriteError(w, "genre is required", http.StatusBadRequest)
			return
		}

		tf := TimeframeFromRequest(r)
		opts := db.GetItemsOpts{Limit: maxTracksForGenreInsight, Page: 1, Timeframe: tf}

		topTracks, err := store.GetTopTracksPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetGenreTracksHandler: Failed to retrieve top tracks")
			utils.WriteError(w, "failed to get genre tracks", http.StatusBadRequest)
			return
		}

		ids := make([]int32, 0, len(topTracks.Items))
		for _, item := range topTracks.Items {
			ids = append(ids, item.Item.ID)
		}

		tagsByTrack, err := store.GetTagsForTracks(ctx, ids)
		if err != nil {
			l.Err(err).Msg("GetGenreTracksHandler: Failed to retrieve track tags")
			utils.WriteError(w, "failed to get genre tracks", http.StatusBadRequest)
			return
		}

		matches := make([]GenreTrack, 0)
		for _, item := range topTracks.Items {
			for _, tw := range tagsByTrack[item.Item.ID] {
				if strings.EqualFold(tw.Tag, genre) {
					matches = append(matches, GenreTrack{
						SimpleTrack: models.SimpleTrack{
							ID:      item.Item.ID,
							Title:   item.Item.Title,
							Artists: item.Item.Artists,
							Image:   item.Item.Image,
						},
						ListenCount: item.Item.ListenCount,
					})
					break
				}
			}
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].ListenCount > matches[j].ListenCount })

		utils.WriteJSON(w, http.StatusOK, matches)
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
