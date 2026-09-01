package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListArticles_SemanticDistance(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
			"articles": [
				{"id": "a1", "title": "Saab wins order", "distance": 0.1842},
				{"id": "a2", "title": "Kongsberg expands", "distance": 0}
			]
		}`))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	params := map[string][]string{"q": {"defence contracts"}, "mode": {"semantic"}}
	articles, _, _, err := c.ListArticles(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery == "" {
		t.Fatal("expected query parameters to be sent")
	}

	if len(articles) != 2 {
		t.Fatalf("len(articles) = %d, want 2", len(articles))
	}
	if articles[0].Distance == nil {
		t.Fatal("articles[0].Distance = nil, want 0.1842")
	}
	if *articles[0].Distance != 0.1842 {
		t.Errorf("articles[0].Distance = %v, want 0.1842", *articles[0].Distance)
	}
	// A distance of exactly zero is a real value, not an absent one — this is
	// why the field is a pointer.
	if articles[1].Distance == nil {
		t.Fatal("articles[1].Distance = nil, want 0")
	}
	if *articles[1].Distance != 0 {
		t.Errorf("articles[1].Distance = %v, want 0", *articles[1].Distance)
	}
}

func TestListArticles_NoDistanceWithoutSemantic(t *testing.T) {
	t.Parallel()

	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"articles": [{"id": "a1", "title": "Saab wins order"}]}`))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	articles, _, _, err := c.ListArticles(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if articles[0].Distance != nil {
		t.Errorf("Distance = %v, want nil on a lexical result", *articles[0].Distance)
	}
}

func TestDistanceOmittedFromJSONWhenAbsent(t *testing.T) {
	t.Parallel()

	b, err := json.Marshal(Article{ID: "a1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(b); strings.Contains(got, "distance") {
		t.Errorf("marshalled article = %s, want no distance key", got)
	}

	d := 0.0
	b, err = json.Marshal(Article{ID: "a1", Distance: &d})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(b); !strings.Contains(got, `"distance":0`) {
		t.Errorf("marshalled article = %s, want a distance of 0", got)
	}
}

func TestAPIError_IsSemanticUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		typ    string
		want   bool
	}{
		{
			name:   "503 with the semantic problem type",
			status: 503,
			typ:    "https://nordicfinancialnews.com/problems/semantic-unavailable",
			want:   true,
		},
		{name: "bare problem type", status: 503, typ: "semantic-unavailable", want: true},
		{name: "503 from something else", status: 503, typ: "service-unavailable", want: false},
		{name: "right type, wrong status", status: 500, typ: "semantic-unavailable", want: false},
		{name: "no type at all", status: 503, typ: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &APIError{StatusCode: tt.status, Type: tt.typ}
			if got := e.IsSemanticUnavailable(); got != tt.want {
				t.Errorf("IsSemanticUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_IsUnknownParameter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		title  string
		detail string
		want   bool
	}{
		{
			name:   "the rollout rejection the changelog quotes",
			status: 400,
			detail: "Unknown parameter 'mode'",
			want:   true,
		},
		{name: "422 variant", status: 422, detail: "Unrecognized parameter: mode", want: true},
		{name: "wording in the title", status: 400, title: "Invalid parameter 'mode'", want: true},
		{name: "case insensitive", status: 400, detail: "UNKNOWN PARAMETER 'MODE'", want: true},
		// Must not swallow unrelated 400s: a bad value for a parameter the API
		// does know about is a different problem with a different fix.
		{name: "bad value for a known parameter", status: 400, detail: "q must be between 2 and 200 characters", want: false},
		{name: "a different unknown parameter", status: 400, detail: "Unknown parameter 'sort'", want: false},
		{name: "not a parameter problem at all", status: 400, detail: "Malformed request", want: false},
		{name: "wrong status", status: 500, detail: "Unknown parameter 'mode'", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &APIError{StatusCode: tt.status, Title: tt.title, Detail: tt.detail}
			if got := e.IsUnknownParameter("mode"); got != tt.want {
				t.Errorf("IsUnknownParameter(\"mode\") = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGet_SemanticUnavailable(t *testing.T) {
	t.Parallel()

	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"type":"https://nordicfinancialnews.com/problems/semantic-unavailable","title":"Semantic search unavailable"}`))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/articles", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err = %T, want *APIError", err)
	}
	if !apiErr.IsSemanticUnavailable() {
		t.Errorf("IsSemanticUnavailable() = false, want true")
	}
	// The client must not retry a 503 into a lexical fallback; a semantic 200
	// is the only thing that means a semantic result.
	if apiErr.StatusCode != 503 {
		t.Errorf("StatusCode = %d, want 503", apiErr.StatusCode)
	}
}

func TestAPIError_RetryAfterPopulated(t *testing.T) {
	t.Parallel()

	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1800")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"title":"Semantic budget exhausted"}`))
	})

	c := NewClient("key", WithBaseURL(srv.URL))
	_, err := c.Get(context.Background(), "/articles", nil, nil)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("err = %T, want *APIError", err)
	}
	if apiErr.RetryAfter != 30*time.Minute {
		t.Errorf("RetryAfter = %v, want 30m0s", apiErr.RetryAfter)
	}
}
