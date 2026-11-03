package news

import "time"

type Article struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Summary     string    `json:"summary"`
	ImageURL    string    `json:"image_url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
}
