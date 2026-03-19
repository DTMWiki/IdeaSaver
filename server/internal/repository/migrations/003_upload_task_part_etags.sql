-- Migration 003: Persist multipart ETags for reliable CompleteMultipartUpload

ALTER TABLE upload_tasks
ADD COLUMN IF NOT EXISTS part_etags JSONB NOT NULL DEFAULT '{}'::jsonb;

