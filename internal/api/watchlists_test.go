package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListWatchlists(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/watchlists" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/watchlists")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"watchlists": []map[string]interface{}{
				{"id": "wl1", "name": "Nordic Banks", "position": 1, "company_count": 4},
				{"id": "wl2", "name": "Green Energy", "position": 2, "company_count": 7},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	watchlists, _, err := c.ListWatchlists(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(watchlists) != 2 {
		t.Fatalf("got %d watchlists, want 2", len(watchlists))
	}
	if watchlists[0].ID != "wl1" {
		t.Errorf("ID = %q, want %q", watchlists[0].ID, "wl1")
	}
	if watchlists[0].Name != "Nordic Banks" {
		t.Errorf("Name = %q, want %q", watchlists[0].Name, "Nordic Banks")
	}
	if watchlists[0].CompanyCount != 4 {
		t.Errorf("CompanyCount = %d, want 4", watchlists[0].CompanyCount)
	}
	if watchlists[1].Position != 2 {
		t.Errorf("Position = %d, want 2", watchlists[1].Position)
	}
}

func TestGetWatchlist(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/watchlists/wl1" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/watchlists/wl1")
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want %q", got, "10")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"watchlist": map[string]interface{}{
				"id": "wl1", "name": "Nordic Banks", "company_count": 1,
			},
			"companies": []map[string]interface{}{
				{"id": "c1", "name": "Volvo", "ticker": "VOLV-B"},
			},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("limit", "10")
	watchlist, companies, pagination, _, err := c.GetWatchlist(context.Background(), "wl1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watchlist.ID != "wl1" {
		t.Errorf("ID = %q, want %q", watchlist.ID, "wl1")
	}
	if watchlist.Name != "Nordic Banks" {
		t.Errorf("Name = %q, want %q", watchlist.Name, "Nordic Banks")
	}
	if len(companies) != 1 {
		t.Fatalf("got %d companies, want 1", len(companies))
	}
	if companies[0].TickerSymbol != "VOLV-B" {
		t.Errorf("ticker = %q, want %q", companies[0].TickerSymbol, "VOLV-B")
	}
	if pagination.Count != 1 {
		t.Errorf("Count = %d, want 1", pagination.Count)
	}
}

func TestGetWatchlist_EscapesID(t *testing.T) {
	t.Parallel()
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"watchlist":  map[string]interface{}{"id": "a/b", "name": "Slashy"},
			"companies":  []map[string]interface{}{},
			"pagination": map[string]interface{}{"count": 0},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	if _, _, _, _, err := c.GetWatchlist(context.Background(), "a/b", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/watchlists/a%2Fb" {
		t.Errorf("escaped path = %q, want %q", gotPath, "/watchlists/a%2Fb")
	}
}

func TestGetWatchlist_APIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"detail": "watchlist not found"})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	_, _, _, _, err := c.GetWatchlist(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
