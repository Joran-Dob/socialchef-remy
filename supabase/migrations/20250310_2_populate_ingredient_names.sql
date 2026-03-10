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

UPDATE recipes r
SET ingredient_names = extract_ingredient_names(r.id)
WHERE ingredient_names IS NULL 
   OR array_length(ingredient_names, 1) IS NULL
   AND EXISTS (
    SELECT 1 FROM recipe_ingredients ri WHERE ri.recipe_id = r.id
);
