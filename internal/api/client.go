package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var Version = "dev"

// RateLimitInfo holds rate limit information from response headers.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// Response wraps the API response with metadata.
type Response struct {
	StatusCode    int
	RateLimitInfo *RateLimitInfo
	// PlanLimited carries the top-level plan_limited flag when the API sends
	// one. It is a pointer because the API omits the key entirely when no plan
	// restriction applied, so nil ("the API said nothing") stays distinct from
	// an explicit false.
	PlanLimited *bool
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// Client is the API client for Nordic Financial News.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    "https://nordicfinancialnews.com/api/v1",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Get performs a GET request and decodes the response into result.
func (c *Client) Get(ctx context.Context, path string, params url.Values, result interface{}) (*Response, error) {
	return c.doGet(ctx, path, params, result, true)
}

func (c *Client) doGet(ctx context.Context, path string, params url.Values, result interface{}, retryOn429 bool) (*Response, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("constructing URL: %w", err)
	}
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("User-Agent", "nfn-cli/"+Version)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rateLimitInfo := parseRateLimitHeaders(resp.Header)
	apiResp := &Response{
		StatusCode:    resp.StatusCode,
		RateLimitInfo: rateLimitInfo,
	}

	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))

	// Absorb a 429 by waiting it out, but only when the wait is short. Semantic
	// search draws on an hourly budget, so its Retry-After routinely runs into
	// the thousands of seconds; sleeping a capped 60s there would just delay
	// the same failure. A long window is reported to the caller instead.
	if resp.StatusCode == http.StatusTooManyRequests && retryOn429 && retryAfter <= maxRetryAfter {
		timer := time.NewTimer(retryAfter)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
		return c.doGet(ctx, path, params, result, false)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		apiErr := &APIError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
		if json.Unmarshal(body, apiErr) != nil {
			apiErr.Detail = string(body)
		}
		return apiResp, apiErr
	}

	// Read the body up front so top-level metadata can be extracted alongside
	// the caller's decode. Typed result structs silently discard JSON keys they
	// don't declare, which would otherwise drop response-level fields.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResp, fmt.Errorf("reading response: %w", err)
	}

	apiResp.PlanLimited = extractPlanLimited(body)

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return apiResp, fmt.Errorf("decoding response: %w", err)
		}
	}

	return apiResp, nil
}

// extractPlanLimited pulls the top-level plan_limited flag out of a response
// body. The API sets it on list endpoints when the caller's plan hid records
// from the result, and omits it otherwise. Bodies that aren't JSON objects
// yield nil rather than an error — this is metadata, not the payload.
func extractPlanLimited(body []byte) *bool {
	var meta struct {
		PlanLimited *bool `json:"plan_limited"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil
	}
	return meta.PlanLimited
}

func parseRateLimitHeaders(h http.Header) *RateLimitInfo {
	limit := h.Get("X-Ratelimit-Limit")
	remaining := h.Get("X-Ratelimit-Remaining")
	if limit == "" && remaining == "" {
		return nil
	}

	info := &RateLimitInfo{}
	info.Limit, _ = strconv.Atoi(limit)
	info.Remaining, _ = strconv.Atoi(remaining)

	if reset := h.Get("X-Ratelimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			info.Reset = time.Unix(ts, 0)
		}
	}

	return info
}

// maxRetryAfter is the longest wait the client will absorb on a 429 before
// handing the error back rather than blocking the caller.
const maxRetryAfter = 60 * time.Second

// parseRetryAfter reads a Retry-After header as a duration, uncapped. A missing
// or unparseable value falls back to a second, matching the API's standard
// rate-limit window.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 1 * time.Second
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return 1 * time.Second
}
