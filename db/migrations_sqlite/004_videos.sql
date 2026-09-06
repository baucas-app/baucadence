-- +goose Up

CREATE TABLE IF NOT EXISTS videos (
    id           INTEGER PRIMARY KEY,
    youtube_id   TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL,
    channel_id   TEXT NOT NULL DEFAULT '',
    channel_name TEXT NOT NULL DEFAULT '',
    thumbnail    TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS video_watches (
    video_id   INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    watched_at INTEGER NOT NULL,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client     TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (video_id, watched_at)
);
CREATE INDEX IF NOT EXISTS idx_video_watches_watched_at ON video_watches(watched_at);
