package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListArticles(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/articles" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/articles")
		}
		if got := r.URL.Query().Get("country"); got != "SE" {
			t.Errorf("country = %q, want %q", got, "SE")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"articles": []map[string]interface{}{
				{"id": "art1", "title": "Test Article", "country": "SE"},
			},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("country", "SE")
	articles, pagination, _, err := c.ListArticles(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("got %d articles, want 1", len(articles))
	}
	if articles[0].ID != "art1" {
		t.Errorf("ID = %q, want %q", articles[0].ID, "art1")
	}
	if pagination.Count != 1 {
		t.Errorf("Count = %d, want 1", pagination.Count)
	}
}

func TestListAllArticles(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"articles":   []map[string]string{{"id": "a1", "title": "One"}},
				"pagination": map[string]interface{}{"count": 2, "next_cursor": "next"},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"articles":   []map[string]string{{"id": "a2", "title": "Two"}},
				"pagination": map[string]interface{}{"count": 2},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	articles, _, err := c.ListAllArticles(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("got %d articles, want 2", len(articles))
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestGetArticle(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/articles/abc123" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/articles/abc123")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"article": map[string]interface{}{
				"id":    "abc123",
				"title": "Test Article",
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	article, _, err := c.GetArticle(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if article.ID != "abc123" {
		t.Errorf("ID = %q, want %q", article.ID, "abc123")
	}
	if article.Title != "Test Article" {
		t.Errorf("Title = %q, want %q", article.Title, "Test Article")
	}
}

func TestGetArticle_SpecialChars(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// url.PathEscape("a/b?c#d") = "a%2Fb%3Fc%23d"
		if r.URL.RawPath != "/articles/a%2Fb%3Fc%23d" {
			t.Errorf("raw path = %q, want encoded special chars", r.URL.RawPath)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"article": map[string]string{"id": "a/b?c#d"},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	article, _, err := c.GetArticle(context.Background(), "a/b?c#d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if article.ID != "a/b?c#d" {
		t.Errorf("ID = %q, want %q", article.ID, "a/b?c#d")
	}
}
