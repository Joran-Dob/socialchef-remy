-- name: GetSocialMediaOwnerByHandle :one
SELECT * FROM social_media_owners 
WHERE instagram_handle = $1 OR tiktok_id = $2;

-- name: CreateSocialMediaOwner :one
INSERT INTO social_media_owners (
    instagram_handle, tiktok_id, name, avatar_url
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetSocialMediaOwner :one
SELECT * FROM social_media_owners WHERE id = $1;
