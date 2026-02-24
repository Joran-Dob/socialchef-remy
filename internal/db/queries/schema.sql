-- schema.sql
-- Generated from Supabase TypeScript migrations
-- DO NOT EDIT MANUALLY - use make sync-schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- Enums
CREATE TYPE recipe_origin AS ENUM ('instagram', 'tiktok');
CREATE TYPE social_media_platform AS ENUM ('instagram', 'tiktok');
CREATE TYPE measurement_unit AS ENUM ('metric', 'imperial');

-- Profiles table
CREATE TABLE profiles (
    id UUID PRIMARY KEY REFERENCES auth.users(id),
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    username TEXT UNIQUE,
    avatar_url TEXT,
    measurement_unit measurement_unit DEFAULT 'metric',
    full_name TEXT GENERATED ALWAYS AS (COALESCE(NULLIF(CONCAT(first_name, last_name), ''), NULLIF(first_name, ''), NULLIF(last_name, ''), NULL)) STORED,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Recipes table
CREATE TABLE recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_name TEXT NOT NULL,
    description TEXT,
    prep_time INTEGER,
    cooking_time INTEGER,
    total_time INTEGER,
    original_serving_size INTEGER,
    difficulty_rating SMALLINT CHECK (difficulty_rating BETWEEN 1 AND 5),
    origin recipe_origin NOT NULL,
    url TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES auth.users(id),
    owner_id UUID REFERENCES social_media_owners(id),
    thumbnail_id UUID REFERENCES recipe_images(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
-- Recipe ingredients table
CREATE TABLE recipe_ingredients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    quantity TEXT,
    unit TEXT,
    original_quantity TEXT,
    original_unit TEXT,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Recipe instructions table
CREATE TABLE recipe_instructions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    instruction TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Recipe nutrition table
CREATE TABLE recipe_nutrition (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE UNIQUE,
    protein DECIMAL,
    carbs DECIMAL,
    fat DECIMAL,
    fiber DECIMAL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Social media owners table
CREATE TABLE social_media_owners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    profile_pic_stored_image_id TEXT,
    origin_id TEXT NOT NULL,
    platform social_media_platform NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(origin_id, platform)
);

-- Stored images table
CREATE TABLE stored_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_path TEXT NOT NULL,
    source_url TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    mime_type TEXT,
    width INTEGER,
    height INTEGER,
    file_size BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(content_hash)
);

-- Recipe images table
CREATE TABLE recipe_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    stored_image_id UUID NOT NULL REFERENCES stored_images(id) ON DELETE RESTRICT,
    image_type TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Recipe import jobs table
CREATE TABLE recipe_import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('instagram', 'tiktok')),
    status TEXT NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED', 'EXECUTING', 'COMPLETED', 'FAILED', 'CRASHED', 'TIMED_OUT', 'CANCELED')),
    progress_step TEXT,
    progress_message TEXT,
    result JSONB,
    error JSONB,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_recipes_origin ON recipes(origin);
CREATE INDEX idx_recipes_recipe_name ON recipes(recipe_name);
CREATE INDEX idx_recipe_ingredients_recipe_id ON recipe_ingredients(recipe_id);
CREATE INDEX idx_recipe_instructions_recipe_id ON recipe_instructions(recipe_id);
CREATE INDEX idx_social_media_owners_platform ON social_media_owners(platform);
CREATE INDEX idx_recipe_images_recipe_id ON recipe_images(recipe_id);
CREATE INDEX idx_stored_images_content_hash ON stored_images(content_hash);
CREATE INDEX idx_recipe_images_stored_image_id ON recipe_images(stored_image_id);
CREATE INDEX idx_recipe_import_jobs_user_id ON recipe_import_jobs(user_id);
CREATE INDEX idx_recipe_import_jobs_job_id ON recipe_import_jobs(job_id);
CREATE INDEX idx_recipe_import_jobs_created_at ON recipe_import_jobs(created_at);
CREATE INDEX idx_recipe_import_jobs_status ON recipe_import_jobs(status);
CREATE INDEX idx_recipe_import_jobs_user_status ON recipe_import_jobs(user_id, status);
