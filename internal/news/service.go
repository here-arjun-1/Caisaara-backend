package news

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const newsPageSize = 20

type Service struct {
	Repository *Repository
	Client     *http.Client
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *Service) RefreshNews(ctx context.Context) error {
	for _, source := range feedSources {
		articles, err := fetchFeed(ctx, s.Client, source)
		if err != nil {
			slog.WarnContext(ctx, "fetch news feed failed", "source", source.Name, "error", err)
			continue
		}

		for _, article := range articles {
			err := s.Repository.Save(ctx, article)
			if err != nil {
				return err
			}
		}

		slog.InfoContext(ctx, "news feed fetched", "source", source.Name, "count", len(articles))
	}

	return nil
}

func (s *Service) StartNewsFetcher(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		s.refreshAndLog(ctx)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.refreshAndLog(ctx)
			}
		}
	}()
}

func (s *Service) refreshAndLog(ctx context.Context) {
	err := s.RefreshNews(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "refresh news failed", "error", err)
	}
}

func (s *Service) GetNews(ctx context.Context, page int) ([]Article, error) {
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * newsPageSize
	return s.Repository.List(ctx, newsPageSize, offset)
}
