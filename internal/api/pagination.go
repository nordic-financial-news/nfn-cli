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
	return listAll[T](ctx, c, path, params, resourceKey, nil)
}

// ListAllDeduped is ListAll for a feed whose cursor walk is at-least-once,
// collecting each id once and keeping the copy from the latest page it appeared
// on. id extracts that id.
//
// The events feed needs this: a merge takes the earlier of the two
// first_reported_at values, so under the default newest-first sort a surviving
// event only ever moves earlier, and a merge landing mid-walk can re-emit a row
// an earlier page already returned. An updated_at walk re-emits for the same
// reason, ordered ascending. Either way the repeat is caused by the record
// changing, so the later copy is the current one — notably on a sync, where an
// event merged mid-walk reappears as a tombstone that a first-wins policy would
// discard, leaving the caller's mirror with an event the API no longer serves.
//
// Completeness differs by sort: the default order can only re-emit, never skip,
// but sort=significance is a best-effort snapshot in which a mid-walk
// routine-to-notable escalation can be missed altogether, which no client-side
// dedupe can repair. The article and story feeds sort on an immutable
// published_at and so cannot drift either way.
func ListAllDeduped[T any](ctx context.Context, c *Client, path string, params url.Values, resourceKey string, id func(T) string) ([]T, *Response, error) {
	return listAll[T](ctx, c, path, params, resourceKey, id)
}

// listAll walks every page. When id is non-nil, a record repeated across pages
// is collected once, at the position it was first seen but with the content of
// the last page that carried it.
func listAll[T any](ctx context.Context, c *Client, path string, params url.Values, resourceKey string, id func(T) string) ([]T, *Response, error) {
	var allItems []T
	var lastResp *Response
	var planLimited *bool

	// Index into allItems rather than a presence flag, so a repeat can overwrite
	// the stale copy in place instead of being dropped.
	var seen map[string]int
	if id != nil {
		seen = make(map[string]int)
	}

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

		if seen == nil {
			allItems = append(allItems, items...)
		} else {
			for _, item := range items {
				key := id(item)
				if i, ok := seen[key]; ok {
					allItems[i] = item
					continue
				}
				seen[key] = len(allItems)
				allItems = append(allItems, item)
			}
		}

		// A single restricted page makes the whole aggregate incomplete, so the
		// flag is sticky once any page reports true.
		if resp != nil && resp.PlanLimited != nil && (planLimited == nil || !*planLimited) {
			planLimited = resp.PlanLimited
		}

		if pagination.NextCursor == "" {
			break
		}
		p.Set("cursor", pagination.NextCursor)
	}

	if lastResp != nil {
		lastResp.PlanLimited = planLimited
	}

	return allItems, lastResp, nil
}
