-- Migration: Add 'youtube' to recipe origins enum
-- Created: 2026-04-10
-- Description: Add YouTube as a supported recipe origin source

-- Add 'youtube' to the recipe_origin enum in an idempotent way
DO $$
BEGIN
    -- Check if 'youtube' is already in the enum
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum
        WHERE enumlabel = 'youtube'
        AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'recipe_origin')
    ) THEN
        -- Add the new value
        ALTER TYPE recipe_origin ADD VALUE 'youtube';
        RAISE NOTICE 'Added youtube to recipe_origin enum';
    ELSE
        RAISE NOTICE 'youtube already exists in recipe_origin enum';
    END IF;
END $$;
