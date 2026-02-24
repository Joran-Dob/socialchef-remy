-- name: GetSocialMediaOwnerByOrigin :one
SELECT * FROM social_media_owners 
WHERE origin_id = $1 AND platform = $2;

-- name: CreateSocialMediaOwner :one
INSERT INTO social_media_owners (
    username, profile_pic_stored_image_id, origin_id, platform
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetSocialMediaOwner :one
SELECT * FROM social_media_owners WHERE id = $1;

-- name: GetSocialMediaOwnerByUsername :one
SELECT * FROM social_media_owners WHERE username = $1;
