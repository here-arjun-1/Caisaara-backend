CREATE TABLE IF NOT EXISTS news_articles (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    link TEXT NOT NULL UNIQUE,
    summary TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    source VARCHAR(100) NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_news_articles_published_at_desc
    ON news_articles (published_at DESC, id DESC);
