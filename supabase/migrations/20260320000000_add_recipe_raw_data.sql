-- Migration: Add raw data storage for recipe sources
-- Created: 2026-03-20
-- Description: Store complete raw data from social media posts for comparison testing

-- Raw data storage table
CREATE TABLE IF NOT EXISTS recipe_raw_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    
    -- Source information
    origin TEXT NOT NULL, -- 'instagram', 'tiktok', 'firecrawl'
    source_url TEXT NOT NULL,
    
    -- Raw scraped data (complete payload)
    raw_data JSONB NOT NULL,
    -- Includes: caption, metadata, images array, video info, etc.
    
    -- Processed components (extracted from raw_data for easy access)
    caption TEXT,
    transcript TEXT, -- Video transcription if available
    video_url TEXT,
    thumbnail_url TEXT,
    images JSONB, -- Array of image URLs
    
    -- Metadata
    scraped_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ,
    
    -- For debugging/reproducibility
    scraper_version TEXT,
    scraper_config JSONB,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(recipe_id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_recipe_raw_data_recipe_id ON recipe_raw_data(recipe_id);
CREATE INDEX IF NOT EXISTS idx_recipe_raw_data_origin ON recipe_raw_data(origin);
CREATE INDEX IF NOT EXISTS idx_recipe_raw_data_scraped_at ON recipe_raw_data(scraped_at);

-- Function to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for updated_at
DROP TRIGGER IF EXISTS update_recipe_raw_data_updated_at ON recipe_raw_data;
CREATE TRIGGER update_recipe_raw_data_updated_at
    BEFORE UPDATE ON recipe_raw_data
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
