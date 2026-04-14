package api

import (
	"context"
	"net/url"
)

func (c *Client) ListSources(ctx context.Context, params url.Values) ([]Source, *Pagination, *Response, error) {
	return ListPage[Source](ctx, c, "/sources", params, "sources")
}

func (c *Client) ListAllSources(ctx context.Context, params url.Values) ([]Source, *Response, error) {
	return ListAll[Source](ctx, c, "/sources", params, "sources")
}
