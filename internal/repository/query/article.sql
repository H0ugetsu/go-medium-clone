-- name: CreateArticle :one
INSERT INTO articles (
    slug, title, description, body, author_id
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, slug, title, description, body, author_id, created_at, updated_at;

-- name: UpsertTag :one
INSERT INTO tags (
    name
) VALUES (
    $1
)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: AddArticleTag :exec
INSERT INTO article_tags (
    article_id, tag_id
) VALUES (
    $1, $2
)
ON CONFLICT DO NOTHING;

-- name: GetArticleBySlug :one
SELECT
    a.slug,
    a.title,
    a.description,
    a.body,
    a.created_at,
    a.updated_at,
    u.username AS author_username,
    u.bio AS author_bio,
    u.image AS author_image,
    EXISTS (
        SELECT 1
        FROM follows f
        WHERE f.follower_id = sqlc.narg('current_user_id')
          AND f.followee_id = u.id
    ) AS author_following,
    EXISTS (
        SELECT 1
        FROM favorites fav
        WHERE fav.user_id = sqlc.narg('current_user_id')
          AND fav.article_id = a.id
    ) AS favorited,
    (
        SELECT COUNT(*)
        FROM favorites fav
        WHERE fav.article_id = a.id
    ) AS favorites_count,
    COALESCE(
        ARRAY_AGG(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
        '{}'
    )::TEXT[] AS tag_list
FROM
    articles a
    JOIN users u ON u.id = a.author_id
    LEFT JOIN article_tags at2 ON at2.article_id = a.id
    LEFT JOIN tags t ON t.id = at2.tag_id
WHERE
    a.slug = sqlc.arg('slug')
GROUP BY
    a.id, u.id, u.username, u.bio, u.image;

-- name: ListArticles :many
SELECT
    a.slug,
    a.title,
    a.description,
    a.created_at,
    a.updated_at,
    u.username AS author_username,
    u.bio AS author_bio,
    u.image AS author_image,
    EXISTS (
        SELECT 1
        FROM follows f
        WHERE f.follower_id = sqlc.narg('current_user_id')
          AND f.followee_id = u.id
    ) AS author_following,
    EXISTS (
        SELECT 1
        FROM favorites fav
        WHERE fav.user_id = sqlc.narg('current_user_id')
          AND fav.article_id = a.id
    ) AS favorited,
    (
        SELECT COUNT(*)
        FROM favorites fav
        WHERE fav.article_id = a.id
    ) AS favorites_count,
    COALESCE(
        ARRAY_AGG(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
        '{}'
    )::TEXT[] AS tag_list,
    COUNT(*) OVER ()::BIGINT AS total_count
FROM
    articles a
    JOIN users u ON u.id = a.author_id
    LEFT JOIN article_tags at2 ON at2.article_id = a.id
    LEFT JOIN tags t ON t.id = at2.tag_id
WHERE
    (
        sqlc.narg('tag')::TEXT IS NULL
        OR EXISTS (
            SELECT 1
            FROM article_tags at3
                JOIN tags t3 ON t3.id = at3.tag_id
            WHERE at3.article_id = a.id
              AND t3.name = sqlc.narg('tag')
        )
    )
    AND (
        sqlc.narg('author')::TEXT IS NULL
        OR u.username = sqlc.narg('author')
    )
    AND (
        sqlc.narg('favorited_by')::TEXT IS NULL
        OR EXISTS (
            SELECT 1
            FROM favorites fav2
                JOIN users u2 ON u2.id = fav2.user_id
            WHERE fav2.article_id = a.id
              AND u2.username = sqlc.narg('favorited_by')
        )
    )
GROUP BY
    a.id, u.id, u.username, u.bio, u.image
ORDER BY
    a.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: ListFeedArticles :many
SELECT
    a.slug,
    a.title,
    a.description,
    a.created_at,
    a.updated_at,
    u.username AS author_username,
    u.bio AS author_bio,
    u.image AS author_image,
    TRUE::BOOLEAN AS author_following,
    EXISTS (
        SELECT 1
        FROM favorites fav
        WHERE fav.user_id = sqlc.arg('current_user_id')
          AND fav.article_id = a.id
    ) AS favorited,
    (
        SELECT COUNT(*)
        FROM favorites fav
        WHERE fav.article_id = a.id
    ) AS favorites_count,
    COALESCE(
        ARRAY_AGG(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
        '{}'
    )::TEXT[] AS tag_list,
    COUNT(*) OVER ()::BIGINT AS total_count
FROM
    articles a
    JOIN users u ON u.id = a.author_id
    JOIN follows f ON f.followee_id = a.author_id
        AND f.follower_id = sqlc.arg('current_user_id')
    LEFT JOIN article_tags at2 ON at2.article_id = a.id
    LEFT JOIN tags t ON t.id = at2.tag_id
GROUP BY
    a.id, u.id, u.username, u.bio, u.image
ORDER BY
    a.created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: ListTags :many
SELECT DISTINCT
    name
FROM
    tags
ORDER BY
    name;

-- name: FindArticleRefBySlug :one
SELECT
    id, author_id
FROM
    articles
WHERE
    slug = $1;

-- name: UpdateArticle :one
UPDATE articles
SET
    title       = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    body        = COALESCE(sqlc.narg('body'), body),
    updated_at  = now()
WHERE
    slug = sqlc.arg('slug')
RETURNING id, slug, title, description, body, author_id, created_at, updated_at;

-- name: ClearArticleTags :exec
DELETE FROM article_tags
WHERE
    article_id = $1;

-- name: DeleteArticle :exec
DELETE FROM articles
WHERE
    slug = $1;

-- name: FavoriteArticle :exec
INSERT INTO favorites (
    user_id, article_id
) VALUES (
    $1, $2
)
ON CONFLICT DO NOTHING;

-- name: UnfavoriteArticle :exec
DELETE FROM favorites
WHERE
    user_id = $1
    AND article_id = $2;

-- name: CreateComment :one
INSERT INTO comments (
    article_id, author_id, body
) VALUES (
    $1, $2, $3
)
RETURNING id, article_id, author_id, body, created_at, updated_at;

-- name: GetCommentByID :one
SELECT
    c.id,
    c.body,
    c.created_at,
    c.updated_at,
    u.username AS author_username,
    u.bio AS author_bio,
    u.image AS author_image,
    EXISTS (
        SELECT 1
        FROM follows f
        WHERE f.follower_id = sqlc.narg('current_user_id')
          AND f.followee_id = u.id
    ) AS author_following
FROM
    comments c
    JOIN users u ON u.id = c.author_id
WHERE
    c.id = sqlc.arg('id');

-- name: ListCommentsByArticleID :many
SELECT
    c.id,
    c.body,
    c.created_at,
    c.updated_at,
    u.username AS author_username,
    u.bio AS author_bio,
    u.image AS author_image,
    EXISTS (
        SELECT 1
        FROM follows f
        WHERE f.follower_id = sqlc.narg('current_user_id')
          AND f.followee_id = u.id
    ) AS author_following
FROM
    comments c
    JOIN users u ON u.id = c.author_id
WHERE
    c.article_id = sqlc.arg('article_id')
ORDER BY
    c.created_at ASC;

-- name: FindCommentRefByID :one
SELECT
    article_id, author_id
FROM
    comments
WHERE
    id = $1;

-- name: DeleteComment :exec
DELETE FROM comments
WHERE
    id = $1;
