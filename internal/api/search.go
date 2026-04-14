package api

import (
	"context"
	"net/url"
)

type SearchResults struct {
	Articles  []Article  `json:"articles"`
	Stories   []Story    `json:"stories"`
	Companies []Company  `json:"companies"`
	Countries []Country  `json:"countries"`
	Exchanges []Exchange `json:"exchanges"`
}

type Country struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (c *Client) Search(ctx context.Context, params url.Values) (*SearchResults, *Response, error) {
	var wrapper struct {
		Results SearchResults `json:"results"`
	}
	resp, err := c.Get(ctx, "/search", params, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.Results, resp, nil
}
