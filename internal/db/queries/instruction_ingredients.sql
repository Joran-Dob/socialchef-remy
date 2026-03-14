-- name: CreateInstructionIngredient :one
INSERT INTO instruction_ingredients (
    instruction_id, ingredient_id, step_quantity
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetInstructionIngredientsByInstruction :many
SELECT * FROM instruction_ingredients WHERE instruction_id = $1;

-- name: GetInstructionIngredientsByRecipe :many
SELECT ii.*
FROM instruction_ingredients ii
JOIN recipe_instructions ri ON ii.instruction_id = ri.id
WHERE ri.recipe_id = $1
ORDER BY ri.step_number;

-- name: DeleteInstructionIngredientsByInstruction :exec
DELETE FROM instruction_ingredients WHERE instruction_id = $1;
