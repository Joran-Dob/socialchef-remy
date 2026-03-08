-- Migration: Add instruction_rich and instruction_rich_version columns to recipe_instructions table
-- This migration must be run on Supabase before the Go code will work
--
-- Run this in Supabase SQL Editor or via CLI:
-- supabase db push

-- Add instruction_rich column for rich text instructions
ALTER TABLE recipe_instructions ADD COLUMN IF NOT EXISTS instruction_rich TEXT DEFAULT NULL;

-- Add instruction_rich_version column for tracking version of rich text
ALTER TABLE recipe_instructions ADD COLUMN IF NOT EXISTS instruction_rich_version INTEGER DEFAULT NULL;
