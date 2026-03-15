-- Migration: Add recipe_parts table
-- Created: 2026-03-15
-- Description: Table for organizing recipes into parts/chapters (e.g., "Marinade", "Glaze", "Assembly")

-- Recipe parts table for multi-section recipes
CREATE TABLE IF NOT EXISTS recipe_parts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_optional BOOLEAN NOT NULL DEFAULT false,
    prep_time INTEGER,
    cooking_time INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(recipe_id, display_order)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_recipe_parts_recipe_id ON recipe_parts(recipe_id);
CREATE INDEX IF NOT EXISTS idx_recipe_parts_display_order ON recipe_parts(display_order);
