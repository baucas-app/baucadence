package handlers

import (
	"net/http"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
)

func GetVideosHandler(store db.VideoStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetVideosHandler: Received request to retrieve video watches")

		opts := OptsFromRequest(r)
		switch r.URL.Query().Get("format") {
		case "video", "short":
			opts.VideoFormat = r.URL.Query().Get("format")
		}
		l.Debug().Msgf("GetVideosHandler: Retrieving video watches with options: %+v", opts)

		videos, err := store.GetVideoWatchesPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetVideosHandler: Failed to retrieve video watches")
			utils.WriteError(w, "failed to get videos: "+err.Error(), http.StatusBadRequest)
			return
		}

		l.Debug().Msg("GetVideosHandler: Successfully retrieved video watches")
		utils.WriteJSON(w, http.StatusOK, videos)
	}
}
