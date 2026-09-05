package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabehf/koito/engine/middleware"
	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/importprogress"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/utils"
)

// RecognizeImportFilename reports whether a filename matches a supported
// export naming convention. Implemented by internal/importer and injected
// in by the router (engine/routes.go) — internal/importer can't be
// imported directly here, since it already imports this package for the
// ListenBrainz payload type.
type RecognizeImportFilename func(name string) bool

// RunImportFile runs the importer matching filename's naming convention
// against a file already sitting in the configured import directory.
// Implemented by internal/importer.DetectAndImportFile and injected in
// for the same reason as RecognizeImportFilename above.
type RunImportFile func(ctx context.Context, filename string) error

// UploadImportHandler lets an admin upload a listening-history export
// (a single recognized file, or a .zip containing one or more of them,
// e.g. a Spotify "Extended Streaming History" export or a Google Takeout
// download) directly through the web UI instead of placing it into the
// import directory by hand. Recognized files are saved into the same
// import directory the startup scanner uses, then imported in the
// background using the same per-format importer.
func UploadImportHandler(recognize RecognizeImportFilename, runImport RunImportFile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("UploadImportHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			l.Debug().Msg("UploadImportHandler: Non-admin user attempted to upload an import file")
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("UploadImportHandler: Invalid file upload")
			utils.WriteError(w, "invalid file upload", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			l.Debug().AnErr("error", err).Msg("UploadImportHandler: Failed to read uploaded file")
			utils.WriteError(w, "failed to read uploaded file (it may be too large)", http.StatusBadRequest)
			return
		}

		importDir := path.Join(cfg.ConfigDir(), "import")
		if err := os.MkdirAll(importDir, 0744); err != nil {
			l.Err(err).Msg("UploadImportHandler: Failed to create import directory")
			utils.WriteError(w, "failed to prepare import directory", http.StatusInternalServerError)
			return
		}

		var saved []string
		if strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
			saved, err = extractRecognizedImportFiles(data, importDir, recognize)
			if err != nil {
				l.Debug().AnErr("error", err).Msg("UploadImportHandler: Failed to read uploaded zip")
				utils.WriteError(w, "failed to read zip file", http.StatusBadRequest)
				return
			}
			if len(saved) == 0 {
				utils.WriteError(w, "no recognized export files were found in the zip", http.StatusBadRequest)
				return
			}
		} else {
			base := filepath.Base(header.Filename)
			if !recognize(base) {
				utils.WriteError(w, "file not recognized as a supported export format", http.StatusBadRequest)
				return
			}
			if err := os.WriteFile(path.Join(importDir, base), data, 0644); err != nil {
				l.Err(err).Msg("UploadImportHandler: Failed to save uploaded file")
				utils.WriteError(w, "failed to save uploaded file", http.StatusInternalServerError)
				return
			}
			saved = []string{base}
		}

		l.Info().Strs("files", saved).Msg("UploadImportHandler: Starting background import of uploaded file(s)")
		importprogress.StartBatch(saved)
		go func() {
			bgCtx := logger.NewContext(l)
			for _, name := range saved {
				if err := runImport(bgCtx, name); err != nil {
					l.Warn().Err(err).Str("file", name).Msg("UploadImportHandler: Background import failed")
				}
			}
		}()

		utils.WriteJSON(w, http.StatusAccepted, map[string]any{
			"started_files": saved,
		})
	}
}

// ImportStatusHandler returns the progress of the most recent import
// batch (started either by an upload or the startup directory scan), so
// the web UI can poll it and show a progress bar.
func ImportStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}
		utils.WriteJSON(w, http.StatusOK, importprogress.Snapshot())
	}
}

// extractRecognizedImportFiles pulls only the entries from a zip archive
// whose base filename matches a known export naming convention, writing
// each one into dir. Everything else in the zip (e.g. the many unrelated
// files in a Spotify or Google Takeout export) is ignored.
func extractRecognizedImportFiles(zipData []byte, dir string, recognize RecognizeImportFilename) ([]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	var saved []string
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if !recognize(base) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		if err := os.WriteFile(path.Join(dir, base), content, 0644); err != nil {
			continue
		}
		saved = append(saved, base)
	}
	return saved, nil
}
