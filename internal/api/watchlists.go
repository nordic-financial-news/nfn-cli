package api

import (
	"context"
	"net/url"
)

type Watchlist struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Position     int    `json:"position"`
	CompanyCount int    `json:"company_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (c *Client) ListWatchlists(ctx context.Context) ([]Watchlist, *Response, error) {
	var wrapper struct {
		Watchlists []Watchlist `json:"watchlists"`
	}
	resp, err := c.Get(ctx, "/watchlists", nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return wrapper.Watchlists, resp, nil
}

func (c *Client) GetWatchlist(ctx context.Context, id string, params url.Values) (*Watchlist, []Company, *Pagination, *Response, error) {
	var wrapper struct {
		Watchlist  Watchlist  `json:"watchlist"`
		Companies  []Company  `json:"companies"`
		Pagination Pagination `json:"pagination"`
	}
	resp, err := c.Get(ctx, "/watchlists/"+url.PathEscape(id), params, &wrapper)
	if err != nil {
		return nil, nil, nil, resp, err
	}
	return &wrapper.Watchlist, wrapper.Companies, &wrapper.Pagination, resp, nil
}
