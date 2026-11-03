package news

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

type feedSource struct {
	Name string
	URL  string
}

var feedSources = []feedSource{
	{Name: "Chess.com", URL: "https://www.chess.com/rss/news"},
	{Name: "ChessBase", URL: "https://en.chessbase.com/feed"},
	{Name: "FIDE", URL: "https://www.fide.com/feed"},
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

const maxFeedSize = 5 * 1024 * 1024

func fetchFeed(ctx context.Context, client *http.Client, source feedSource) ([]Article, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CaisaaraNewsBot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed %s returned status %d", source.Name, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedSize))
	if err != nil {
		return nil, err
	}

	var feed rssFeed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, err
	}

	articles := []Article{}
	for _, item := range feed.Channel.Items {
		title := cleanText(item.Title)
		link := strings.TrimSpace(item.Link)

		if title == "" || link == "" {
			continue
		}

		article := Article{
			Title:       title,
			Link:        link,
			Summary:     cleanText(item.Description),
			Source:      source.Name,
			PublishedAt: parsePubDate(item.PubDate),
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func cleanText(text string) string {
	text = html.UnescapeString(text)
	text = html.UnescapeString(text)
	return strings.TrimSpace(text)
}

func parsePubDate(value string) time.Time {
	value = strings.TrimSpace(value)

	parsed, err := time.Parse(time.RFC1123Z, value)
	if err == nil {
		return parsed
	}

	parsed, err = time.Parse(time.RFC1123, value)
	if err == nil {
		return parsed
	}

	return time.Now()
}
