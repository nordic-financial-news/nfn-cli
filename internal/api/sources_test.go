package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListSources(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sources" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/sources")
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want %q", got, "5")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sources": []map[string]interface{}{
				{"id": "src1", "name": "Dagens Industri", "domain": "di.se", "country": "SE"},
			},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("limit", "5")
	sources, pagination, _, err := c.ListSources(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("got %d sources, want 1", len(sources))
	}
	if sources[0].ID != "src1" {
		t.Errorf("ID = %q, want %q", sources[0].ID, "src1")
	}
	if sources[0].Name != "Dagens Industri" {
		t.Errorf("Name = %q, want %q", sources[0].Name, "Dagens Industri")
	}
	if sources[0].Domain != "di.se" {
		t.Errorf("Domain = %q, want %q", sources[0].Domain, "di.se")
	}
	if sources[0].Country != "SE" {
		t.Errorf("Country = %q, want %q", sources[0].Country, "SE")
	}
	if pagination.Count != 1 {
		t.Errorf("Count = %d, want 1", pagination.Count)
	}
}

func TestListAllSources(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"sources":    []map[string]string{{"id": "s1", "name": "One", "domain": "one.se"}},
				"pagination": map[string]interface{}{"count": 2, "next_cursor": "next"},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"sources":    []map[string]string{{"id": "s2", "name": "Two", "domain": "two.se"}},
				"pagination": map[string]interface{}{"count": 2},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	sources, _, err := c.ListAllSources(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 2 {
		t.Errorf("got %d sources, want 2", len(sources))
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}
