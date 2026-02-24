-- name: GetProfile :one
SELECT * FROM profiles WHERE id = $1;

-- name: UpdateProfile :one
UPDATE profiles 
SET 
    username = $2, 
    avatar_url = $3, 
    measurement_unit = $4,
    updated_at = NOW()
WHERE id = $1 
RETURNING *;
