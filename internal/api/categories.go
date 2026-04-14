package api

import (
	"context"
	"net/url"
)

type CategoryDetail struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListCategories(ctx context.Context, params url.Values) ([]CategoryDetail, *Pagination, *Response, error) {
	return ListPage[CategoryDetail](ctx, c, "/categories", params, "categories")
}
