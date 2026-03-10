ALTER TABLE recipes ADD COLUMN IF NOT EXISTS ingredient_names text[];

CREATE INDEX IF NOT EXISTS idx_recipes_ingredients ON recipes USING gin(ingredient_names);
