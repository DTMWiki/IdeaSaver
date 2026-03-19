-- Migration 006: Persist transcode callback state for callback-gated playback

ALTER TABLE videos
ADD COLUMN IF NOT EXISTS transcode_status VARCHAR(32) NOT NULL DEFAULT 'pending';

ALTER TABLE videos
ADD COLUMN IF NOT EXISTS transcode_message TEXT NOT NULL DEFAULT '';

UPDATE videos
SET transcode_status = CASE
    WHEN COALESCE(play_url, '') <> '' THEN 'ready'
    ELSE 'processing'
END
WHERE COALESCE(transcode_status, '') = '';

CREATE INDEX IF NOT EXISTS idx_videos_transcode_status ON videos(transcode_status);
