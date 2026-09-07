package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

// runListCommandExpectingError drives a list command that is expected to fail,
// returning the error instead of failing the test. It mirrors runListCommand.
//
// These tests must not call t.Parallel: they mutate shared cobra flag state.
func runListCommandExpectingError(t *testing.T, cmd *cobra.Command, handler http.HandlerFunc, args ...string) error {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	ctx := context.WithValue(context.Background(), formatterKey, output.NewFormatterWithWriter("json", true, &buf))
	ctx = context.WithValue(ctx, clientKey, api.NewClient("key", api.WithBaseURL(srv.URL)))

	restoreFlags(t, cmd)

	c := *cmd
	c.SetContext(ctx)
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("parsing flags: %v", err)
	}
	err := c.RunE(&c, nil)
	if err == nil {
		t.Fatal("command succeeded, want an error")
	}
	return err
}

func TestArticlesList_SemanticEndToEnd(t *testing.T) {
	var gotParams url.Values
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"articles":[
			{"id":"a1","title":"Saab lands new order","country":"SE","published_at":"2026-08-01","distance":0.1842},
			{"id":"a2","title":"Kongsberg adds capacity","country":"NO","published_at":"2026-07-28","distance":0.2311}
		]}`))
	}

	out := runListCommand(t, articlesListCmd, handler, "table",
		"--q", "defence contract wins and capacity expansion", "--semantic")

	if got := gotParams.Get("mode"); got != "semantic" {
		t.Errorf("mode = %q, want %q", got, "semantic")
	}
	if got := gotParams.Get("q"); got != "defence contract wins and capacity expansion" {
		t.Errorf("q = %q, want the query", got)
	}
	if !strings.Contains(strings.ToUpper(out), "DISTANCE") {
		t.Errorf("output = %q, want a Distance column", out)
	}
	if !strings.Contains(out, "0.1842") {
		t.Errorf("output = %q, want the distance value", out)
	}
}

// The Distance column must stay out of the way of an ordinary lexical listing.
func TestArticlesList_NoDistanceColumnWithoutSemantic(t *testing.T) {
	handler := fixedBody(`{"articles":[{"id":"a1","title":"Saab lands new order","country":"SE","published_at":"2026-08-01"}]}`)

	out := runListCommand(t, articlesListCmd, handler, "table", "--q", "Saab")

	if strings.Contains(strings.ToUpper(out), "DISTANCE") {
		t.Errorf("output = %q, want no Distance column on a lexical result", out)
	}
}

func TestArticlesList_SemanticJSONCarriesDistance(t *testing.T) {
	handler := fixedBody(`{"articles":[{"id":"a1","title":"Saab lands new order","country":"SE","published_at":"2026-08-01","distance":0.1842}]}`)

	out := runListCommand(t, articlesListCmd, handler, "json", "--q", "defence orders", "--semantic")

	var envelope struct {
		Data struct {
			Articles []struct {
				ID       string   `json:"id"`
				Distance *float64 `json:"distance"`
			} `json:"articles"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decoding output: %v\n%s", err, out)
	}
	if len(envelope.Data.Articles) != 1 {
		t.Fatalf("len(articles) = %d, want 1", len(envelope.Data.Articles))
	}
	if envelope.Data.Articles[0].Distance == nil {
		t.Fatal("distance = nil, want 0.1842")
	}
	if *envelope.Data.Articles[0].Distance != 0.1842 {
		t.Errorf("distance = %v, want 0.1842", *envelope.Data.Articles[0].Distance)
	}
}

func TestCompaniesList_SemanticEndToEnd(t *testing.T) {
	var gotParams url.Values
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"companies":[
			{"id":"c1","name":"Vestas Wind Systems","ticker":"VWS","is_active":true,"distance":0.0912}
		]}`))
	}

	out := runListCommand(t, companiesListCmd, handler, "table",
		"--q", "offshore wind turbine installation", "--semantic", "--market", "DK")

	if got := gotParams.Get("mode"); got != "semantic" {
		t.Errorf("mode = %q, want %q", got, "semantic")
	}
	// Filters still apply as hard filters alongside semantic matching.
	if got := gotParams.Get("market"); got != "DK" {
		t.Errorf("market = %q, want %q", got, "DK")
	}
	if !strings.Contains(out, "0.0912") {
		t.Errorf("output = %q, want the distance value", out)
	}
}

func TestStoriesList_SemanticEndToEnd(t *testing.T) {
	var gotParams url.Values
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stories":[
			{"id":"s1","title":"Port congestion spreads","country":"SE","article_count":4,"published_at":"2026-08-01","distance":0.2044}
		]}`))
	}

	out := runListCommand(t, storiesListCmd, handler, "table",
		"--q", "supply chain disruption", "--semantic")

	if got := gotParams.Get("mode"); got != "semantic" {
		t.Errorf("mode = %q, want %q", got, "semantic")
	}
	if got := gotParams.Get("q"); got != "supply chain disruption" {
		t.Errorf("q = %q, want the query", got)
	}
	if !strings.Contains(out, "0.2044") {
		t.Errorf("output = %q, want the distance value", out)
	}
}

// A semantic request that violates the API's constraints must fail before any
// HTTP call is made, so the hourly semantic budget is not spent on it.
func TestSemanticValidationSkipsTheRequest(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "no query",
			args:    []string{"--semantic"},
			wantErr: "needs a query",
		},
		{
			name:    "cursor",
			args:    []string{"--semantic", "--q", "battery plants", "--cursor", "abc"},
			wantErr: "--cursor cannot be combined with --semantic",
		},
		{
			name:    "all",
			args:    []string{"--semantic", "--q", "battery plants", "--all"},
			wantErr: "--all cannot be combined with --semantic",
		},
		{
			// The API answers this pair with a message naming mode=semantic, a
			// parameter the caller never typed. Catching it here names --sort.
			name:    "sort",
			args:    []string{"--semantic", "--q", "battery plants", "--sort", "latest"},
			wantErr: "--sort cannot be combined with --semantic",
		},
		{
			name:    "updated-after",
			args:    []string{"--semantic", "--q", "battery plants", "--updated-after", "2026-01-01T00:00:00Z"},
			wantErr: "--updated-after cannot be combined with --semantic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			handler := func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"articles":[]}`))
			}

			err := runListCommandExpectingError(t, articlesListCmd, handler, tt.args...)
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
			if calls != 0 {
				t.Errorf("calls = %d, want 0 (validation must precede the request)", calls)
			}
		})
	}
}

func TestArticlesList_SemanticUnavailableEndToEnd(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"type":"https://nordicfinancialnews.com/problems/semantic-unavailable","title":"Semantic search unavailable"}`))
	}

	err := runListCommandExpectingError(t, articlesListCmd, handler, "--q", "defence orders", "--semantic")

	if !strings.Contains(err.Error(), "temporarily unavailable") {
		t.Errorf("err = %q, want it to explain the outage", err.Error())
	}
	if !strings.Contains(err.Error(), "drop --semantic") {
		t.Errorf("err = %q, want it to suggest the lexical fallback", err.Error())
	}
}

func TestArticlesList_SemanticBudgetExhaustedEndToEnd(t *testing.T) {
	calls := 0
	handler := func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "1800")
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"title":"Too many requests"}`))
	}

	err := runListCommandExpectingError(t, articlesListCmd, handler, "--q", "defence orders", "--semantic")

	if !strings.Contains(err.Error(), "120 per hour") {
		t.Errorf("err = %q, want it to mention the semantic budget", err.Error())
	}
	if !strings.Contains(err.Error(), "30m0s") {
		t.Errorf("err = %q, want it to report the retry window", err.Error())
	}
	// A half-hour window is far past what the client will wait out, so it must
	// report rather than sleep and retry.
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on a long window)", calls)
	}
}
