package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Pagination holds pagination info from the API response.
type Pagination struct {
	Count      int    `json:"count"`
	NextCursor string `json:"next_cursor"`
}

// PaginatedResponse is a generic wrapper for paginated API responses.
type PaginatedResponse[T any] struct {
	Items      []T        `json:"-"`
	Pagination Pagination `json:"pagination"`
}

// ListPage fetches a single page of results. The resourceKey is the JSON key
// holding the array (e.g., "articles", "stories").
func ListPage[T any](ctx context.Context, c *Client, path string, params url.Values, resourceKey string) ([]T, *Pagination, *Response, error) {
	var raw json.RawMessage
	resp, err := c.Get(ctx, path, params, &raw)
	if err != nil {
		return nil, nil, resp, err
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, nil, resp, fmt.Errorf("decoding envelope: %w", err)
	}

	var items []T
	if data, ok := envelope[resourceKey]; ok {
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, nil, resp, fmt.Errorf("decoding %s: %w", resourceKey, err)
		}
	}

	var pagination Pagination
	if data, ok := envelope["pagination"]; ok {
		if err := json.Unmarshal(data, &pagination); err != nil {
			return nil, nil, resp, fmt.Errorf("decoding pagination: %w", err)
		}
	}

	return items, &pagination, resp, nil
}

// ListAll fetches all pages of results by following pagination cursors.
func ListAll[T any](ctx context.Context, c *Client, path string, params url.Values, resourceKey string) ([]T, *Response, error) {
	var allItems []T
	var lastResp *Response

	p := url.Values{}
	for k, v := range params {
		p[k] = v
	}

	for {
		items, pagination, resp, err := ListPage[T](ctx, c, path, p, resourceKey)
		if err != nil {
			return allItems, resp, err
		}
		lastResp = resp
		allItems = append(allItems, items...)

		if pagination.NextCursor == "" {
			break
		}
		p.Set("cursor", pagination.NextCursor)
	}

	return allItems, lastResp, nil
}
