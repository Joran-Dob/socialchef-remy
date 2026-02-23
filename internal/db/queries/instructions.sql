-- name: GetInstructionsByRecipe :many
SELECT * FROM recipe_instructions WHERE recipe_id = $1 ORDER BY step_number;

-- name: CreateInstruction :one
INSERT INTO recipe_instructions (
    recipe_id, step_number, instruction
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: DeleteInstructionsByRecipe :exec
DELETE FROM recipe_instructions WHERE recipe_id = $1;
