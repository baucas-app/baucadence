package images

import (
	"context"
	"errors"
)

type MockFinder struct{}

func (m *MockFinder) GetArtistImage(ctx context.Context, opts ArtistImageOpts) (string, error) {
	return "", nil
}

func (m *MockFinder) GetAlbumImage(ctx context.Context, opts AlbumImageOpts) (string, error) {
	return "", nil
}
func (m *MockFinder) Shutdown() {}

type ErrorFinder struct{}

func (m *ErrorFinder) GetArtistImage(ctx context.Context, opts ArtistImageOpts) (string, error) {
	return "", errors.New("mock error")
}

func (m *ErrorFinder) GetAlbumImage(ctx context.Context, opts AlbumImageOpts) (string, error) {
	return "", errors.New("mock error")
}
func (m *ErrorFinder) Shutdown() {}
