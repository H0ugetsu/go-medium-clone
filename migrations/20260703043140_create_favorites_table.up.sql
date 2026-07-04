CREATE TABLE favorites (
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    article_id BIGINT NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, article_id)
);

CREATE INDEX idx_favorites_article_id ON favorites (article_id);
