package moodtags

import (
	"context"

	"github.com/gabehf/koito/internal/cfg"
	"github.com/gabehf/koito/internal/db"
)

// SettingKeyApiKey is the app_settings row key for a LastFM API key entered
// through the Settings UI.
const SettingKeyApiKey = "lastfm_api_key"

// ResolveApiKey returns the LastFM API key to use: a value saved through
// the Settings UI takes priority (no restart needed to pick it up), falling
// back to the KOITO_LASTFM_API_KEY environment variable for admins who
// prefer configuring it that way.
func ResolveApiKey(ctx context.Context, store db.SettingsStore) (string, error) {
	dbKey, err := store.GetSetting(ctx, SettingKeyApiKey)
	if err != nil {
		return "", err
	}
	if dbKey != "" {
		return dbKey, nil
	}
	return cfg.LastFMApiKey(), nil
}
