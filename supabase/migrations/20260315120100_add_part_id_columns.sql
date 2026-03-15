-- Add part_id to recipe_ingredients
ALTER TABLE recipe_ingredients
    ADD COLUMN IF NOT EXISTS part_id UUID REFERENCES recipe_parts(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_part_id ON recipe_ingredients(part_id);

-- Add part_id to recipe_instructions
ALTER TABLE recipe_instructions
    ADD COLUMN IF NOT EXISTS part_id UUID REFERENCES recipe_parts(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_recipe_instructions_part_id ON recipe_instructions(part_id);
