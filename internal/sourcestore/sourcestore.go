// Package sourcestore persists small pieces of per-source state (OAuth
// tokens, session cookies, "last seen" cursors) to disk as encrypted JSON,
// one file per source, inside the app's config directory.
package sourcestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gabehf/koito/internal/secure"
)

// Store reads and writes one encrypted JSON document per source name.
type Store struct {
	dir string
	box *secure.Box
	mu  sync.Mutex
}

// New creates a Store rooted at dir (created if missing), encrypting all
// documents with box.
func New(dir string, box *secure.Box) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("sourcestore.New: failed to create directory: %w", err)
	}
	return &Store{dir: dir, box: box}, nil
}

func (s *Store) path(name string) string {
	return filepath.Join(s.dir, name+".enc.json")
}

// Load decodes the stored document for name into dst (a pointer). If no
// document has been saved yet, it returns ErrNotFound.
func (s *Store) Load(name string, dst any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("sourcestore.Load: failed to read file: %w", err)
	}
	plaintext, err := s.box.Decrypt(string(raw))
	if err != nil {
		return fmt.Errorf("sourcestore.Load: failed to decrypt: %w", err)
	}
	if err := json.Unmarshal(plaintext, dst); err != nil {
		return fmt.Errorf("sourcestore.Load: failed to unmarshal: %w", err)
	}
	return nil
}

// Save encrypts and writes src (any JSON-serializable value) as the
// document for name, replacing any previous value atomically.
func (s *Store) Save(name string, src any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plaintext, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("sourcestore.Save: failed to marshal: %w", err)
	}
	ciphertext, err := s.box.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("sourcestore.Save: failed to encrypt: %w", err)
	}
	tmp := s.path(name) + ".tmp"
	if err := os.WriteFile(tmp, []byte(ciphertext), 0o600); err != nil {
		return fmt.Errorf("sourcestore.Save: failed to write temp file: %w", err)
	}
	if err := os.Rename(tmp, s.path(name)); err != nil {
		return fmt.Errorf("sourcestore.Save: failed to finalize file: %w", err)
	}
	return nil
}

// Delete removes the stored document for name, if any.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path(name))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("sourcestore.Delete: %w", err)
	}
	return nil
}

// errNotFound is a sentinel returned by Load when no document exists yet.
type notFoundError struct{}

func (notFoundError) Error() string { return "sourcestore: no document saved yet" }

// ErrNotFound is returned by Load when the requested document has not been
// saved yet (e.g. a source has never been authenticated).
var ErrNotFound error = notFoundError{}
