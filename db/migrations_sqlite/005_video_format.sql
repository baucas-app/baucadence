-- +goose Up

ALTER TABLE videos ADD COLUMN format TEXT NOT NULL DEFAULT 'video';
