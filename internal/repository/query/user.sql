-- name: CreateUser :one
INSERT INTO users (
    username, email, password
) VALUES (
    $1, $2, $3
) 
RETURNING id, username, email, bio, image, created_at, updated_at;

-- name: FindByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1;

-- name: FindByID :one
SELECT
    id, username, email, bio, image, created_at, updated_at
FROM
    users
WHERE
    id = $1;

-- name: FindProfileByUsername :one
SELECT
    u.username,
    u.bio,
    u.image,
    EXISTS (
        SELECT 1
        FROM follows f
        WHERE f.follower_id = sqlc.narg('current_user_id')
          AND f.followee_id = u.id
    ) AS following
FROM
    users u
WHERE
    u.username = $1;

-- name: FindIDByUsername :one
SELECT
    id
FROM
    users
WHERE
    username = $1;

-- name: Follow :exec
INSERT INTO follows (
    follower_id, followee_id
) VALUES (
    $1, $2
)
ON CONFLICT DO NOTHING;

-- name: Unfollow :exec
DELETE FROM follows
WHERE
    follower_id = $1
    AND followee_id = $2;

-- name: UpdateUser :one
UPDATE users
SET
    username   = COALESCE(sqlc.narg('username'), username),
    email      = COALESCE(sqlc.narg('email'), email),
    password   = COALESCE(sqlc.narg('password'), password),
    bio        = CASE WHEN sqlc.arg('bio_set')::bool THEN sqlc.narg('bio') ELSE bio END,
    image      = CASE WHEN sqlc.arg('image_set')::bool THEN sqlc.narg('image') ELSE image END,
    updated_at = now()
WHERE
    id = sqlc.arg('id')
RETURNING id, username, email, bio, image, created_at, updated_at;