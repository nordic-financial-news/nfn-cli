package api

import (
	"context"
	"net/url"
)

type Story struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	Content      string    `json:"content,omitempty"`
	PublishedAt  string    `json:"published_at"`
	UpdatedAt    string    `json:"updated_at,omitempty"`
	ArticleCount int       `json:"article_count"`
	ArticleIDs   []string  `json:"article_ids"`
	Country      string    `json:"country"`
	CompanyIDs   []string  `json:"company_ids,omitempty"`
	Companies    []Company `json:"companies,omitempty"`
	Category     *Category `json:"category"`
}

func (c *Client) ListStories(ctx context.Context, params url.Values) ([]Story, *Pagination, *Response, error) {
	return ListPage[Story](ctx, c, "/stories", params, "stories")
}

func (c *Client) ListAllStories(ctx context.Context, params url.Values) ([]Story, *Response, error) {
	return ListAll[Story](ctx, c, "/stories", params, "stories")
}

func (c *Client) GetStory(ctx context.Context, id string) (*Story, *Response, error) {
	var wrapper struct {
		Story Story `json:"story"`
	}
	resp, err := c.Get(ctx, "/stories/"+url.PathEscape(id), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Story, resp, nil
}
