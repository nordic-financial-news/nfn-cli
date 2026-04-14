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

	if resp.StatusCode == http.StatusTooManyRequests && retryOn429 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
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
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if json.Unmarshal(body, apiErr) != nil {
			apiErr.Detail = string(body)
		}
		return apiResp, apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return apiResp, fmt.Errorf("decoding response: %w", err)
		}
	}

	return apiResp, nil
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

const maxRetryAfter = 60 * time.Second

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 1 * time.Second
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		d := time.Duration(seconds) * time.Second
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}
	return 1 * time.Second
}
