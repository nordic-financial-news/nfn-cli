package api

import (
	"context"
	"net/url"
)

type IndexSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type StockIndex struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Symbol         string    `json:"symbol"`
	PanNordic      bool      `json:"pan_nordic"`
	Exchange       *Exchange `json:"exchange"`
	Description    string    `json:"description,omitempty"`
	CompaniesCount int       `json:"companies_count,omitempty"`
}

func (c *Client) ListIndices(ctx context.Context, params url.Values) ([]StockIndex, *Pagination, *Response, error) {
	return ListPage[StockIndex](ctx, c, "/indices", params, "indices")
}

func (c *Client) ListAllIndices(ctx context.Context, params url.Values) ([]StockIndex, *Response, error) {
	return ListAll[StockIndex](ctx, c, "/indices", params, "indices")
}

func (c *Client) GetIndex(ctx context.Context, idOrSymbol string) (*StockIndex, *Response, error) {
	var wrapper struct {
		StockIndex StockIndex `json:"index"`
	}
	resp, err := c.Get(ctx, "/indices/"+url.PathEscape(idOrSymbol), nil, &wrapper)
	if err != nil {
		return nil, resp, err
	}
	return &wrapper.StockIndex, resp, nil
}

func (c *Client) ListIndexCompanies(ctx context.Context, idOrSymbol string, params url.Values) ([]Company, *Pagination, *Response, error) {
	return ListPage[Company](ctx, c, "/indices/"+url.PathEscape(idOrSymbol)+"/companies", params, "companies")
}

func (c *Client) ListAllIndexCompanies(ctx context.Context, idOrSymbol string, params url.Values) ([]Company, *Response, error) {
	return ListAll[Company](ctx, c, "/indices/"+url.PathEscape(idOrSymbol)+"/companies", params, "companies")
}
