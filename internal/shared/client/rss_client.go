package client

import (
	"context"
	"net/url"

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
	
	// Format untuk Google News RSS bahasa Indonesia
	searchURL := "https://news.google.com/rss/search?q=" + url.QueryEscape(keyword) + "&hl=id&gl=ID&ceid=ID:id"
	
	feed, err := fp.ParseURLWithContext(searchURL, ctx)
	if err != nil {
		c.Log.Warnf("Failed to fetch RSS for %s : %+v", keyword, err)
		return nil, err
	}

	var articles []RSSArticle
	for _, item := range feed.Items {
		content := item.Description
		if content == "" {
			content = item.Title
		}
		
		source := "Google News RSS"
		
		articles = append(articles, RSSArticle{
			Title:       item.Title,
			URL:         item.Link,
			Content:     content,
			PublishedAt: item.Published,
			SourceName:  source,
		})
	}

	return articles, nil
}
