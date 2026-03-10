-- Migration: Add bulk import jobs table
-- Created: 2026-03-10

-- Bulk import jobs table
CREATE TABLE IF NOT EXISTS bulk_import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    total_urls INTEGER NOT NULL,
    processed_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED', 'EXECUTING', 'COMPLETED', 'FAILED', 'CANCELED')),
    summary JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add bulk_job_id to recipe_import_jobs for linking to bulk jobs
ALTER TABLE recipe_import_jobs 
    ADD COLUMN IF NOT EXISTS bulk_job_id TEXT REFERENCES bulk_import_jobs(job_id) ON DELETE CASCADE;

-- Indexes for bulk_import_jobs
CREATE INDEX IF NOT EXISTS idx_bulk_import_jobs_user_id ON bulk_import_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_bulk_import_jobs_job_id ON bulk_import_jobs(job_id);
CREATE INDEX IF NOT EXISTS idx_bulk_import_jobs_status ON bulk_import_jobs(status);
CREATE INDEX IF NOT EXISTS idx_bulk_import_jobs_user_status ON bulk_import_jobs(user_id, status);

-- Index for linking individual jobs to bulk jobs
CREATE INDEX IF NOT EXISTS idx_recipe_import_jobs_bulk_job_id ON recipe_import_jobs(bulk_job_id);
