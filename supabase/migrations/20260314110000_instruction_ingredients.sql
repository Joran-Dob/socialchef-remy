-- Migration: Add instruction_ingredients junction table
-- Created: 2026-03-14
-- Description: Junction table linking recipe instructions to ingredients with optional per-step quantities

-- Junction table for instruction-ingredient relationships
CREATE TABLE IF NOT EXISTS instruction_ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instruction_id UUID NOT NULL REFERENCES recipe_instructions(id) ON DELETE CASCADE,
    ingredient_id UUID NOT NULL REFERENCES recipe_ingredients(id) ON DELETE CASCADE,
    step_quantity TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(instruction_id, ingredient_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_instruction_ingredients_instruction_id ON instruction_ingredients(instruction_id);
CREATE INDEX IF NOT EXISTS idx_instruction_ingredients_ingredient_id ON instruction_ingredients(ingredient_id);
