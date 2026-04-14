package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestNewClient_Defaults(t *testing.T) {
	t.Parallel()
	c := NewClient("test-key")

	if c.baseURL != "https://nordicfinancialnews.com/api/v1" {
		t.Errorf("baseURL = %q, want default", c.baseURL)
	}
	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "test-key")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", c.httpClient.Timeout)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	t.Parallel()
	custom := &http.Client{Timeout: 5 * time.Second}
	c := NewClient("key", WithBaseURL("https://example.com"), WithHTTPClient(custom))

	if c.baseURL != "https://example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://example.com")
	}
	if c.httpClient != custom {
		t.Error("httpClient not set by WithHTTPClient")
	}
}

func TestGet_Success(t *testing.T) {
	t.Parallel()
	type result struct {
		Name string `json:"name"`
	}

	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(result{Name: "test"})
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	var got result
	resp, err := c.Get(context.Background(), "/test", nil, &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
}

func TestGet_SetsAuthHeader(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-secret-key" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer my-secret-key")
		}
		_, _ = w.Write([]byte("{}"))
	})

	c := NewClient("my-secret-key", WithBaseURL(srv.URL))
	_, _ = c.Get(context.Background(), "/test", nil, nil)
}

func TestGet_NoAuthHeader(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Authorization should be empty, got %q", auth)
		}
		_, _ = w.Write([]byte("{}"))
	})

	c := NewClient("", WithBaseURL(srv.URL))
	_, _ = c.Get(context.Background(), "/test", nil, nil)
}

func TestGet_SetsUserAgent(t *testing.T) {
	// Not parallel — mutates package-level Version
	oldVersion := Version
	Version = "1.2.3"
	defer func() { Version = oldVersion }()

	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "nfn-cli/1.2.3" {
			t.Errorf("User-Agent = %q, want %q", ua, "nfn-cli/1.2.3")
		}
		_, _ = w.Write([]byte("{}"))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, _ = c.Get(context.Background(), "/test", nil, nil)
}

func TestGet_SetsAcceptJSON(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "application/json" {
			t.Errorf("Accept = %q, want %q", accept, "application/json")
		}
		_, _ = w.Write([]byte("{}"))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, _ = c.Get(context.Background(), "/test", nil, nil)
}

func TestGet_URLEncoding(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "hello world" {
			t.Errorf("q = %q, want %q", got, "hello world")
		}
		if got := r.URL.Query().Get("category"); got != "Economic Policy" {
			t.Errorf("category = %q, want %q", got, "Economic Policy")
		}
		_, _ = w.Write([]byte("{}"))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("q", "hello world")
	params.Set("category", "Economic Policy")
	_, _ = c.Get(context.Background(), "/test", params, nil)
}

func TestGet_ErrorResponse_JSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantMsg    string
	}{
		{
			name:       "400 with detail",
			statusCode: 400,
			body:       `{"title":"Bad Request","detail":"invalid parameter"}`,
			wantMsg:    "invalid parameter",
		},
		{
			name:       "404 with title only",
			statusCode: 404,
			body:       `{"title":"Not Found"}`,
			wantMsg:    "Not Found",
		},
		{
			name:       "500 with no body",
			statusCode: 500,
			body:       "",
			wantMsg:    "API error: HTTP 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			})

			c := NewClient("key", WithBaseURL(srv.URL))
			_, err := c.Get(context.Background(), "/test", nil, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if apiErr.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", apiErr.Error(), tt.wantMsg)
			}
		})
	}
}

func TestGet_ErrorResponse_NonJSON(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
		_, _ = w.Write([]byte("<html>Bad Gateway</html>"))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/test", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Detail != "<html>Bad Gateway</html>" {
		t.Errorf("Detail = %q, want HTML body", apiErr.Detail)
	}
}

func TestGet_ErrorResponse_LimitedRead(t *testing.T) {
	t.Parallel()
	// Server sends a body larger than 4096 bytes
	bigBody := strings.Repeat("x", 8192)
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(bigBody))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/test", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if len(apiErr.Detail) > 4096 {
		t.Errorf("Detail length = %d, want <= 4096", len(apiErr.Detail))
	}
}

func TestGet_RateLimited_Retry(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"title":"Rate limited"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestGet_RateLimited_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"title":"Rate limited"}`))
	})

	// Timeout long enough for the first request (localhost) but shorter than
	// the 30s retry wait, so the select picks ctx.Done() during the wait.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(ctx, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestGet_RateLimited_NoInfiniteRetry(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"title":"Rate limited"}`))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error on second 429")
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (initial + one retry)", calls)
	}
}

func TestGet_Unauthorized(t *testing.T) {
	t.Parallel()
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"title":"Unauthorized","detail":"invalid API key"}`))
	})

	c := NewClient("bad-key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/test", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsUnauthorized() {
		t.Error("expected IsUnauthorized() == true")
	}
}

func TestParseRateLimitHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		headers       map[string]string
		wantNil       bool
		wantLimit     int
		wantRemaining int
	}{
		{
			name:    "no headers",
			headers: map[string]string{},
			wantNil: true,
		},
		{
			name: "all headers present",
			headers: map[string]string{
				"X-Ratelimit-Limit":     "100",
				"X-Ratelimit-Remaining": "42",
				"X-Ratelimit-Reset":     "1700000000",
			},
			wantLimit:     100,
			wantRemaining: 42,
		},
		{
			name: "partial headers",
			headers: map[string]string{
				"X-Ratelimit-Limit": "50",
			},
			wantLimit:     50,
			wantRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := http.Header{}
			for k, v := range tt.headers {
				h.Set(k, v)
			}
			info := parseRateLimitHeaders(h)
			if tt.wantNil {
				if info != nil {
					t.Errorf("expected nil, got %+v", info)
				}
				return
			}
			if info == nil {
				t.Fatal("expected non-nil RateLimitInfo")
			}
			if info.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", info.Limit, tt.wantLimit)
			}
			if info.Remaining != tt.wantRemaining {
				t.Errorf("Remaining = %d, want %d", info.Remaining, tt.wantRemaining)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "empty", value: "", want: 1 * time.Second},
		{name: "zero", value: "0", want: 0},
		{name: "five", value: "5", want: 5 * time.Second},
		{name: "invalid", value: "abc", want: 1 * time.Second},
		{name: "capped at 60s", value: "999999", want: 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseRetryAfter(tt.value)
			if got != tt.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
