-- Migration 004: Store DogeCloud player user id for SDK playback

ALTER TABLE videos
ADD COLUMN IF NOT EXISTS player_user_id VARCHAR(64) DEFAULT '';

