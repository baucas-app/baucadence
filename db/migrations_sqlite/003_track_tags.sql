-- +goose Up

ALTER TABLE tracks ADD COLUMN tags_synced_at INTEGER;

CREATE TABLE IF NOT EXISTS track_tags (
    track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    weight   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (track_id, tag)
);
CREATE INDEX IF NOT EXISTS idx_track_tags_track_id ON track_tags(track_id);
