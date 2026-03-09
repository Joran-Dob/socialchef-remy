-- name: GetInstructionsByRecipe :many
SELECT * FROM recipe_instructions WHERE recipe_id = $1 ORDER BY step_number;

-- name: CreateInstruction :one
INSERT INTO recipe_instructions (
    recipe_id, step_number, instruction, timer_data, instruction_rich, instruction_rich_version
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: DeleteInstructionsByRecipe :exec
DELETE FROM recipe_instructions WHERE recipe_id = $1;

-- name: UpdateInstructionRich :exec
UPDATE recipe_instructions
SET instruction_rich = $1, instruction_rich_version = $2
WHERE id = $3;
