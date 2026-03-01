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
    focused_diet TEXT,
    estimated_calories INTEGER,
    origin recipe_origin NOT NULL,
    url TEXT NOT NULL,
    language TEXT DEFAULT 'en',
    created_by UUID NOT NULL REFERENCES auth.users(id),
    owner_id UUID REFERENCES social_media_owners(id),
    thumbnail_id UUID REFERENCES recipe_images(id),
    embedding vector(1536),
    search_vector tsvector,
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


-- Enable pg_trgm for fuzzy text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Category Tables
CREATE TABLE cuisine_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE meal_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE occasions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE dietary_restrictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE equipment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Category Junction Tables
CREATE TABLE recipe_cuisine_categories (
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    cuisine_category_id UUID REFERENCES cuisine_categories(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (recipe_id, cuisine_category_id)
);

CREATE TABLE recipe_meal_types (
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    meal_type_id UUID REFERENCES meal_types(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (recipe_id, meal_type_id)
);

CREATE TABLE recipe_occasions (
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    occasion_id UUID REFERENCES occasions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (recipe_id, occasion_id)
);

CREATE TABLE recipe_dietary_restrictions (
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    dietary_restriction_id UUID REFERENCES dietary_restrictions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (recipe_id, dietary_restriction_id)
);

CREATE TABLE recipe_equipment (
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    equipment_id UUID REFERENCES equipment(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (recipe_id, equipment_id)
);

-- Category Indexes
CREATE INDEX idx_recipe_cuisine_categories_recipe_id ON recipe_cuisine_categories(recipe_id);
CREATE INDEX idx_recipe_meal_types_recipe_id ON recipe_meal_types(recipe_id);
CREATE INDEX idx_recipe_occasions_recipe_id ON recipe_occasions(recipe_id);
CREATE INDEX idx_recipe_dietary_restrictions_recipe_id ON recipe_dietary_restrictions(recipe_id);
CREATE INDEX idx_recipe_equipment_recipe_id ON recipe_equipment(recipe_id);

-- Search Indexes
CREATE INDEX IF NOT EXISTS recipe_search_idx ON recipes USING GiST (search_vector);
CREATE INDEX IF NOT EXISTS recipe_embedding_idx ON recipes USING hnsw (embedding vector_cosine_ops);
