// Package importprogress tracks the progress of listening-history import
// jobs so the web UI can poll it and show a percentage. It has no
// dependency on internal/importer or engine/handlers (both of which
// already depend on each other in places) precisely so both can import
// it without creating a cycle.
package importprogress

import "sync"

type FileProgress struct {
	Filename string `json:"filename"`
	// Total is the number of items to process, or -1 if that isn't known
	// upfront for this format (the UI should show an indeterminate state).
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
}

var (
	mu    sync.RWMutex
	batch []*FileProgress
)

// StartBatch resets progress tracking for a new set of files about to be
// imported, in order. Only one batch is tracked at a time, which is fine
// for a single-user self-hosted instance running one import job at once.
func StartBatch(filenames []string) {
	mu.Lock()
	defer mu.Unlock()
	batch = make([]*FileProgress, len(filenames))
	for i, f := range filenames {
		batch[i] = &FileProgress{Filename: f, Total: -1}
	}
}

// SetTotal records how many items filename's import will process, once
// known (typically right after decoding its export data).
func SetTotal(filename string, total int) {
	mu.Lock()
	defer mu.Unlock()
	if p := find(filename); p != nil {
		p.Total = total
	}
}

// Advance records that one more item of filename has been processed.
func Advance(filename string) {
	mu.Lock()
	defer mu.Unlock()
	if p := find(filename); p != nil {
		p.Processed++
	}
}

// Finish marks filename's import as complete, successfully or not.
func Finish(filename string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if p := find(filename); p != nil {
		p.Done = true
		if err != nil {
			p.Error = err.Error()
		}
	}
}

// find must be called with mu held.
func find(filename string) *FileProgress {
	for _, p := range batch {
		if p.Filename == filename {
			return p
		}
	}
	return nil
}

// Snapshot returns the current progress of the most recent import batch.
func Snapshot() []FileProgress {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]FileProgress, len(batch))
	for i, p := range batch {
		out[i] = *p
	}
	return out
}
