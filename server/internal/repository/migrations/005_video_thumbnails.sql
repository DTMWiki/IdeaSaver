-- Migration 005: Store DogeCloud video thumbnails for list rendering

ALTER TABLE videos
ADD COLUMN IF NOT EXISTS thumbnail_url TEXT DEFAULT '';

ALTER TABLE videos
ADD COLUMN IF NOT EXISTS thumbnail_small_url TEXT DEFAULT '';
