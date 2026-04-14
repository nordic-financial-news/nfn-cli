package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type testItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestListPage_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []testItem{
				{ID: "1", Name: "First"},
				{ID: "2", Name: "Second"},
			},
			"pagination": map[string]interface{}{
				"count":       2,
				"next_cursor": "abc123",
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	items, pagination, _, err := ListPage[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
	if items[0].Name != "First" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "First")
	}
	if pagination.Count != 2 {
		t.Errorf("Count = %d, want 2", pagination.Count)
	}
	if pagination.NextCursor != "abc123" {
		t.Errorf("NextCursor = %q, want %q", pagination.NextCursor, "abc123")
	}
}

func TestListPage_EmptyResults(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items":      []testItem{},
			"pagination": map[string]interface{}{"count": 0},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	items, pagination, _, err := ListPage[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
	if pagination.Count != 0 {
		t.Errorf("Count = %d, want 0", pagination.Count)
	}
}

func TestListPage_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "unauthorized"})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("bad-key", WithBaseURL(srv.URL))
	_, _, _, err := ListPage[testItem](context.Background(), c, "/items", nil, "items")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListAll_SinglePage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []testItem{
				{ID: "1", Name: "Only"},
			},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	items, _, err := ListAll[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("got %d items, want 1", len(items))
	}
}

func TestListAll_MultiplePages(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cursor := r.URL.Query().Get("cursor")
		switch cursor {
		case "":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items":      []testItem{{ID: "1", Name: "First"}},
				"pagination": map[string]interface{}{"count": 2, "next_cursor": "page2"},
			})
		case "page2":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items":      []testItem{{ID: "2", Name: "Second"}},
				"pagination": map[string]interface{}{"count": 2},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	items, _, err := ListAll[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestListAll_PreservesParams(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("country"); got != "SE" {
			t.Errorf("country = %q, want %q", got, "SE")
		}

		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items":      []testItem{{ID: "1"}},
				"pagination": map[string]interface{}{"count": 2, "next_cursor": "page2"},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items":      []testItem{{ID: "2"}},
				"pagination": map[string]interface{}{"count": 2},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("country", "SE")
	items, _, err := ListAll[testItem](context.Background(), c, "/items", params, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}

	// Verify original params not mutated
	if params.Get("cursor") != "" {
		t.Error("original params should not have cursor set")
	}
}
