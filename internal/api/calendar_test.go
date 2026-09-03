package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListCalendarEvents(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendar_events" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/calendar_events")
		}
		if got := r.URL.Query().Get("event_type"); got != "dividend" {
			t.Errorf("event_type = %q, want %q", got, "dividend")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"calendar_events": []map[string]interface{}{
				{
					"id":         "cale_q3volvo1",
					"title":      "Volvo AB Q3 2026 interim report",
					"event_type": "earnings_report",
					"company_id": "64hn2d7ul6kr",
					"company": map[string]interface{}{
						"id":     "64hn2d7ul6kr",
						"name":   "Volvo AB",
						"ticker": "VOLV-B",
					},
					"dividend": nil,
				},
				{
					"id":         "cale_dividlim1",
					"title":      "Lime Technologies dividend 2026",
					"event_type": "dividend",
					"dividend": map[string]interface{}{
						"ex_date":      "2026-04-25",
						"record_date":  "2026-04-28",
						"payment_date": "2026-05-06",
						"kind":         "ordinary",
					},
				},
			},
			"pagination": map[string]interface{}{"count": 2, "next_cursor": "next"},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("event_type", "dividend")
	events, pagination, _, err := c.ListCalendarEvents(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].ID != "cale_q3volvo1" {
		t.Errorf("ID = %q, want %q", events[0].ID, "cale_q3volvo1")
	}
	if events[0].Dividend != nil {
		t.Errorf("non-dividend event Dividend = %+v, want nil", events[0].Dividend)
	}
	if events[0].CompanyID != "64hn2d7ul6kr" {
		t.Errorf("CompanyID = %q, want %q", events[0].CompanyID, "64hn2d7ul6kr")
	}
	if events[0].Company == nil {
		t.Fatal("inline Company = nil, want populated object")
	}
	if events[0].Company.Name != "Volvo AB" {
		t.Errorf("Company.Name = %q, want %q", events[0].Company.Name, "Volvo AB")
	}
	if events[0].Company.Ticker != "VOLV-B" {
		t.Errorf("Company.Ticker = %q, want %q", events[0].Company.Ticker, "VOLV-B")
	}
	if events[1].Company != nil {
		t.Errorf("event without inline company = %+v, want nil", events[1].Company)
	}
	if events[1].Dividend == nil {
		t.Fatal("dividend event Dividend = nil, want populated object")
	}
	if events[1].Dividend.ExDate != "2026-04-25" {
		t.Errorf("ExDate = %q, want %q", events[1].Dividend.ExDate, "2026-04-25")
	}
	if events[1].Dividend.Kind != "ordinary" {
		t.Errorf("Kind = %q, want %q", events[1].Dividend.Kind, "ordinary")
	}
	if pagination.Count != 2 {
		t.Errorf("Count = %d, want 2", pagination.Count)
	}
	if pagination.NextCursor != "next" {
		t.Errorf("NextCursor = %q, want %q", pagination.NextCursor, "next")
	}
}

func TestListAllCalendarEvents(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"calendar_events": []map[string]string{{"id": "c1", "title": "One"}},
				"pagination":      map[string]interface{}{"count": 2, "next_cursor": "next"},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"calendar_events": []map[string]string{{"id": "c2", "title": "Two"}},
				"pagination":      map[string]interface{}{"count": 2},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	events, _, err := c.ListAllCalendarEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("got %d events, want 2", len(events))
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestGetCalendarEvent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendar_events/cale_q3volvo1" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/calendar_events/cale_q3volvo1")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"calendar_event": map[string]interface{}{
				"id":         "cale_q3volvo1",
				"title":      "Volvo AB Q3 2026 interim report",
				"event_type": "earnings_report",
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	event, _, err := c.GetCalendarEvent(context.Background(), "cale_q3volvo1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "cale_q3volvo1" {
		t.Errorf("ID = %q, want %q", event.ID, "cale_q3volvo1")
	}
	if event.EventType != "earnings_report" {
		t.Errorf("EventType = %q, want %q", event.EventType, "earnings_report")
	}
}

func TestGetCalendarEvent_SpecialChars(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// url.PathEscape("a/b?c#d") = "a%2Fb%3Fc%23d"
		if r.URL.RawPath != "/calendar_events/a%2Fb%3Fc%23d" {
			t.Errorf("raw path = %q, want encoded special chars", r.URL.RawPath)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"calendar_event": map[string]string{"id": "a/b?c#d"},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	event, _, err := c.GetCalendarEvent(context.Background(), "a/b?c#d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "a/b?c#d" {
		t.Errorf("ID = %q, want %q", event.ID, "a/b?c#d")
	}
}
