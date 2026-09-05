// Package secure provides authenticated encryption for credentials (OAuth
// tokens, session cookies, etc.) that source connectors persist to disk.
// It is not used for listen/track data, which is stored in plain SQLite
// like the rest of the catalog.
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// Box encrypts and decrypts small blobs of data (credentials) using
// AES-256-GCM. The key is derived from the raw secret via SHA-256 so any
// length of input secret produces a valid 32-byte AES-256 key.
type Box struct {
	gcm cipher.AEAD
}

// NewBox builds a Box from a raw secret (e.g. the value of an environment
// variable). The secret must not be empty.
func NewBox(secret string) (*Box, error) {
	if secret == "" {
		return nil, errors.New("secure.NewBox: secret must not be empty")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("secure.NewBox: failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secure.NewBox: failed to create GCM: %w", err)
	}
	return &Box{gcm: gcm}, nil
}

// Encrypt returns a base64-encoded, self-contained ciphertext (nonce is
// prepended) safe to write to disk or a database column.
func (b *Box) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secure.Encrypt: failed to generate nonce: %w", err)
	}
	ciphertext := b.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt. It returns an error if the ciphertext was
// tampered with or the key is wrong (authentication failure).
func (b *Box) Decrypt(encoded string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("secure.Decrypt: invalid base64: %w", err)
	}
	nonceSize := b.gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("secure.Decrypt: ciphertext too short")
	}
	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := b.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("secure.Decrypt: authentication failed (wrong key or corrupted data): %w", err)
	}
	return plaintext, nil
}
