UPDATE recipes
SET search_vector =
    setweight(to_tsvector('english', COALESCE(recipe_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(array_to_string(ingredient_names, ' '), '')), 'C')
WHERE search_vector IS NULL 
   OR search_vector = '';

CREATE INDEX IF NOT EXISTS recipe_search_idx ON recipes USING GiST (search_vector);
