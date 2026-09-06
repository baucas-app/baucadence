package handlers

import (
	"net/http"

	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/utils"
)

func GetVideoCategoriesHandler(store db.VideoStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetVideoCategoriesHandler: Received request to retrieve video category counts")

		tf := TimeframeFromRequest(r)

		counts, err := store.GetVideoCategoryCounts(ctx, tf)
		if err != nil {
			l.Err(err).Msg("GetVideoCategoriesHandler: Failed to retrieve video category counts")
			utils.WriteError(w, "failed to get video categories: "+err.Error(), http.StatusBadRequest)
			return
		}

		l.Debug().Msg("GetVideoCategoriesHandler: Successfully retrieved video category counts")
		utils.WriteJSON(w, http.StatusOK, counts)
	}
}
