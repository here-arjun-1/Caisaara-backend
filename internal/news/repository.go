package news

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) Save(ctx context.Context, article Article) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO news_articles (title, link, summary, image_url, source, published_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (link) DO NOTHING`,
		article.Title, article.Link, article.Summary, article.ImageURL, article.Source, article.PublishedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "save news article failed", "link", article.Link, "error", err)
		return err
	}
	return nil
}

func (r *Repository) List(ctx context.Context, limit int, offset int) ([]Article, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT id, title, link, summary, image_url, source, published_at
		FROM news_articles
		ORDER BY published_at DESC, id DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "list news articles failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	articles := []Article{}
	for rows.Next() {
		var article Article
		err := rows.Scan(&article.ID, &article.Title, &article.Link, &article.Summary, &article.ImageURL, &article.Source, &article.PublishedAt)
		if err != nil {
			slog.ErrorContext(ctx, "scan news article row failed", "error", err)
			return nil, err
		}
		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "news articles rows iteration error", "error", err)
		return nil, err
	}

	return articles, nil
}
