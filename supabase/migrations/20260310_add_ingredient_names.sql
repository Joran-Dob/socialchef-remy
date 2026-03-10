-- Add ingredient_names array column
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS ingredient_names text[];

-- Create function to extract ingredient names from JSONB
CREATE OR REPLACE FUNCTION extract_ingredient_names(ingredients jsonb)
RETURNS text[]
LANGUAGE plpgsql
AS $$
DECLARE
    names text[];
BEGIN
    SELECT array_agg(DISTINCT elem->>'name')
    INTO names
    FROM jsonb_array_elements(ingredients) AS elem
    WHERE elem->>'name' IS NOT NULL;

    RETURN names;
END;
$$;

-- Update existing recipes to populate ingredient_names
UPDATE recipes
SET ingredient_names = extract_ingredient_names(ingredients)
WHERE ingredients IS NOT NULL;

-- Create index for ingredient search
CREATE INDEX IF NOT EXISTS idx_recipes_ingredients ON recipes USING gin(ingredient_names);

-- Update search_vector to include ingredients
UPDATE recipes
SET search_vector =
    setweight(to_tsvector('english', COALESCE(recipe_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(array_to_string(ingredient_names, ' '), '')), 'C');
