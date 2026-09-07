package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
)

const defaultTopVideoChannelsLimit = 5

// GetTopVideoChannelsHandler ranks channels by watch count in the requested
// period, mirroring GetTopArtistsHandler for the Videos page.
func GetTopVideoChannelsHandler(store db.VideoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		tf := TimeframeFromRequest(r)
		channels, err := store.GetTopVideoChannels(ctx, tf, defaultTopVideoChannelsLimit)
		if err != nil {
			l.Err(err).Msg("GetTopVideoChannelsHandler: Failed to retrieve top video channels")
			utils.WriteError(w, "failed to get top video channels", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, channels)
	}
}

// GetVideoFormatSplitHandler reports how many watches in the period were
// longform videos vs Shorts.
func GetVideoFormatSplitHandler(store db.VideoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		tf := TimeframeFromRequest(r)
		split, err := store.GetVideoFormatSplit(ctx, tf)
		if err != nil {
			l.Err(err).Msg("GetVideoFormatSplitHandler: Failed to retrieve video format split")
			utils.WriteError(w, "failed to get video format split", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, split)
	}
}

// GetVideoActivityHandler returns a per-step (day/week/month/year) time
// series of longform vs Shorts watch counts, the video-page equivalent of
// GetListenActivityHandler. It reuses that handler's step bucketing helpers
// (normalizeToStep/addStep, same package) since the logic is identical -
// only the underlying counts differ.
func GetVideoActivityHandler(store db.VideoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		rangeStr := r.URL.Query().Get("range")
		_range, err := strconv.Atoi(rangeStr)
		if err != nil || _range <= 0 {
			_range = 90
		}

		var step db.StepInterval
		switch strings.ToLower(r.URL.Query().Get("step")) {
		case "week":
			step = db.StepWeek
		case "month":
			step = db.StepMonth
		case "year":
			step = db.StepYear
		default:
			step = db.StepDay
		}

		tz := parseTZ(r)
		if strings.ToLower(tz.String()) == "local" {
			tz, _ = time.LoadLocation("UTC")
		}

		from, to := db.ListenActivityOptsToTimes(db.ListenActivityOpts{
			Step:     step,
			Range:    _range,
			Timezone: tz,
		})

		counts, err := store.GetVideoDailyFormatCounts(ctx, from, to)
		if err != nil {
			l.Err(err).Msg("GetVideoActivityHandler: Failed to retrieve video activity")
			utils.WriteError(w, "failed to retrieve video activity", http.StatusInternalServerError)
			return
		}

		type bucket struct{ longform, shortform int64 }
		buckets := make(map[string]bucket)
		for _, c := range counts {
			bucketStart := normalizeToStep(c.Date, step)
			key := bucketStart.Format("2006-01-02")
			b := buckets[key]
			if c.Format == "short" {
				b.shortform += c.Count
			} else {
				b.longform += c.Count
			}
			buckets[key] = b
		}

		result := make([]db.VideoActivityItem, 0)
		for t := normalizeToStep(from, step); t.Before(to); t = addStep(t, step) {
			b := buckets[t.Format("2006-01-02")]
			result = append(result, db.VideoActivityItem{
				Start:     t,
				Longform:  b.longform,
				Shortform: b.shortform,
			})
		}

		utils.WriteJSON(w, http.StatusOK, struct {
			Activity []db.VideoActivityItem `json:"activity"`
		}{result})
	}
}
