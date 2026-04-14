package api

import (
	"context"
	"net/url"
)

type Article struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	SourceTitle string    `json:"source_title,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	ArticleURL  string    `json:"article_url"`
	ContentType string    `json:"content_type"`
	PublishedAt string    `json:"published_at"`
	UpdatedAt   string    `json:"updated_at,omitempty"`
	StoryIDs    []string  `json:"story_ids,omitempty"`
	Category    *Category `json:"category"`
	Source      *Source   `json:"source"`
	Companies   []Company `json:"companies,omitempty"`
	CompanyIDs  []string  `json:"company_ids,omitempty"`
	Country     string    `json:"country"`
	KeyPoints   []string  `json:"key_points,omitempty"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Source struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Domain  string `json:"domain"`
	Country string `json:"country,omitempty"`
}

func (c *Client) ListArticles(ctx context.Context, params url.Values) ([]Article, *Pagination, *Response, error) {
	return ListPage[Article](ctx, c, "/articles", params, "articles")
}

func (c *Client) ListAllArticles(ctx context.Context, params url.Values) ([]Article, *Response, error) {
	return ListAll[Article](ctx, c, "/articles", params, "articles")
}

func (c *Client) GetArticle(ctx context.Context, id string) (*Article, *Response, error) {
	var wrapper struct {
		Article Article `json:"article"`
	}
	resp, err := c.Get(ctx, "/articles/"+url.PathEscape(id), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Article, resp, nil
}
