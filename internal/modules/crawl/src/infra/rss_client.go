package infra

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
)

type RSSArticle struct {
	Title       string
	URL         string
	Content     string
	PublishedAt string
	SourceName  string
}

type RssClient interface {
	FetchArticles(ctx context.Context, keyword string) ([]RSSArticle, error)
}

type rssClientImpl struct {
	Log *logrus.Logger
}

func NewRssClient(log *logrus.Logger) RssClient {
	return &rssClientImpl{Log: log}
}

func (c *rssClientImpl) FetchArticles(ctx context.Context, keyword string) ([]RSSArticle, error) {
	fp := gofeed.NewParser()
	searchURL := "https://news.google.com/rss/search?q=" + url.QueryEscape(keyword) + "&hl=id&gl=ID&ceid=ID:id"

	feed, err := fp.ParseURLWithContext(searchURL, ctx)
	if err != nil {
		c.Log.Warnf("Failed to fetch RSS for %s : %+v", keyword, err)
		return nil, err
	}

	articles := make([]RSSArticle, 0, len(feed.Items))
	for _, item := range feed.Items {
		content := cleanRSSContent(item.Description)
		if content == "" {
			content = strings.TrimSpace(item.Title)
		}

		articles = append(articles, RSSArticle{
			Title:       item.Title,
			URL:         item.Link,
			Content:     content,
			PublishedAt: item.Published,
			SourceName:  "Google News RSS",
		})
	}

	return articles, nil
}

var htmlTagRegex = regexp.MustCompile(`(?i)<[^>]*>`)

// cleanRSSContent strips HTML tags/entities from Google News RSS descriptions,
// leaving plain readable text so the LLM extraction prompt gets clean input.
func cleanRSSContent(raw string) string {
	raw = htmlTagRegex.ReplaceAllString(raw, " ")
	raw = strings.ReplaceAll(raw, "&nbsp;", " ")
	raw = strings.ReplaceAll(raw, "&amp;", "&")
	raw = strings.ReplaceAll(raw, "&quot;", `"`)
	raw = strings.ReplaceAll(raw, "&#39;", "'")
	raw = strings.ReplaceAll(raw, "&lt;", "<")
	raw = strings.ReplaceAll(raw, "&gt;", ">")
	return strings.Join(strings.Fields(raw), " ")
}
