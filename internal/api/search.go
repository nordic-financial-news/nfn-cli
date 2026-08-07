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
	// PlanLimited is hoisted from the top level of the response, where it is a
	// sibling of "results" rather than a member of it. It lives here so JSON
	// output can surface it under .data without reshaping the payload.
	PlanLimited *bool `json:"plan_limited,omitempty"`
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
	results := wrapper.Results
	results.PlanLimited = resp.PlanLimited
	return &results, resp, nil
}
