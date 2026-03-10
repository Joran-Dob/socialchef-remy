-- Add ingredient_names array column
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS ingredient_names text[];

-- Create function to extract ingredient names from recipe_ingredients table
CREATE OR REPLACE FUNCTION extract_ingredient_names(p_recipe_id UUID)
RETURNS text[]
LANGUAGE plpgsql
AS $$
DECLARE
    names text[];
BEGIN
    SELECT array_agg(DISTINCT name)
    INTO names
    FROM recipe_ingredients
    WHERE recipe_id = p_recipe_id;

    RETURN names;
END;
$$;

-- Update existing recipes to populate ingredient_names from recipe_ingredients table
UPDATE recipes r
SET ingredient_names = extract_ingredient_names(r.id)
WHERE EXISTS (
    SELECT 1 FROM recipe_ingredients ri WHERE ri.recipe_id = r.id
);

-- Create index for ingredient search
CREATE INDEX IF NOT EXISTS idx_recipes_ingredients ON recipes USING gin(ingredient_names);

-- Update search_vector to include ingredients
UPDATE recipes
SET search_vector =
    setweight(to_tsvector('english', COALESCE(recipe_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(array_to_string(ingredient_names, ' '), '')), 'C');
