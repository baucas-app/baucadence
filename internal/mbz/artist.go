package mbz

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/gabehf/koito/internal/logger"
	"github.com/google/uuid"
)

type MusicBrainzArtist struct {
	Name     string                   `json:"name"`
	SortName string                   `json:"sort_name"`
	Gender   string                   `json:"gender"`
	Area     MusicBrainzArea          `json:"area"`
	Aliases  []MusicBrainzArtistAlias `json:"aliases"`
}
type MusicBrainzArtistAlias struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Primary bool   `json:"primary"`
}

const artistAliasFmtStr = "%s/ws/2/artist/%s?inc=aliases"

func (c *MusicBrainzClient) getArtist(ctx context.Context, id uuid.UUID) (*MusicBrainzArtist, error) {
	mbzArtist := new(MusicBrainzArtist)
	err := c.getEntity(ctx, artistAliasFmtStr, id, mbzArtist)
	if err != nil {
		return nil, fmt.Errorf("getArtist: %w", err)
	}
	return mbzArtist, nil
}

// Returns the artist name at index 0, and all primary aliases after.
func (c *MusicBrainzClient) GetArtistPrimaryAliases(ctx context.Context, id uuid.UUID) ([]string, error) {
	l := logger.FromContext(ctx)
	artist, err := c.getArtist(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetArtistPrimaryAliases: %w", err)
	}
	if artist == nil {
		return nil, errors.New("GetArtistPrimaryAliases: artist could not be found by musicbrainz")
	}
	used := make(map[string]bool)
	ret := make([]string, 1)
	ret[0] = artist.Name
	used[artist.Name] = true
	for _, alias := range artist.Aliases {
		if alias.Primary && !slices.Contains(ret, alias.Name) {
			l.Debug().Msgf("Found primary alias '%s' for artist '%s'", alias.Name, artist.Name)
			ret = append(ret, alias.Name)
		}
	}
	return ret, nil
}
