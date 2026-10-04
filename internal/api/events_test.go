package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListEvents(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/events")
		}
		if got := r.URL.Query().Get("event_type"); got != "earnings-report" {
			t.Errorf("event_type = %q, want %q", got, "earnings-report")
		}
		if got := r.URL.Query().Get("significance"); got != "notable" {
			t.Errorf("significance = %q, want %q", got, "notable")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []map[string]interface{}{
				{
					"id":           "k3n8vdczq1mp",
					"title":        "Volvo AB announces Q3 2026 results",
					"status":       "active",
					"significance": "notable",
					"event_type": map[string]interface{}{
						"slug":     "earnings-report",
						"label":    "Earnings Report",
						"category": "earnings",
					},
					"article_count":     3,
					"first_reported_at": "2026-10-22T05:00:00.000Z",
					"last_article_at":   "2026-10-23T09:15:00.000Z",
					"updated_at":        "2026-10-23T09:20:00.000Z",
					"company_ids":       []string{"uejazctgchj4"},
					"story_ids":         []string{"8fzq2rmtnk5c"},
					"merged_into":       nil,
					"related_to":        nil,
				},
			},
			"pagination": map[string]interface{}{"count": 1, "next_cursor": "next"},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("event_type", "earnings-report")
	params.Set("significance", "notable")
	events, pagination, _, err := c.ListEvents(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	e := events[0]
	if e.ID != "k3n8vdczq1mp" {
		t.Errorf("ID = %q, want %q", e.ID, "k3n8vdczq1mp")
	}
	if e.Significance != "notable" {
		t.Errorf("Significance = %q, want %q", e.Significance, "notable")
	}
	if e.EventType == nil {
		t.Fatal("EventType = nil, want the inlined type")
	}
	if e.EventType.Slug != "earnings-report" || e.EventType.Label != "Earnings Report" {
		t.Errorf("EventType = %+v, want slug earnings-report / label Earnings Report", *e.EventType)
	}
	if e.ArticleCount != 3 {
		t.Errorf("ArticleCount = %d, want 3", e.ArticleCount)
	}
	if e.IsTombstone() {
		t.Error("IsTombstone() = true for an active event, want false")
	}
	// merged_into and related_to arrive as JSON null on a live event.
	if e.MergedInto != "" {
		t.Errorf("MergedInto = %q, want empty", e.MergedInto)
	}
	if e.RelatedTo != "" {
		t.Errorf("RelatedTo = %q, want empty", e.RelatedTo)
	}
	if pagination.NextCursor != "next" {
		t.Errorf("NextCursor = %q, want %q", pagination.NextCursor, "next")
	}
}

func TestListEventsSemanticDistance(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events":     []map[string]interface{}{{"id": "k3n8vdczq1mp", "status": "active", "distance": 0.2913}},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	events, _, _, err := c.ListEvents(context.Background(), url.Values{"mode": {"semantic"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events[0].Distance == nil || *events[0].Distance != 0.2913 {
		t.Errorf("Distance = %v, want 0.2913", events[0].Distance)
	}
}

func TestListAllEvents(t *testing.T) {
	t.Parallel()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/events" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/events")
		}
		if r.URL.Query().Get("cursor") == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events":     []map[string]string{{"id": "e1", "title": "One"}},
				"pagination": map[string]interface{}{"count": 2, "next_cursor": "next"},
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events":     []map[string]string{{"id": "e2", "title": "Two"}},
				"pagination": map[string]interface{}{"count": 2},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	events, _, err := c.ListAllEvents(context.Background(), nil)
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

func TestGetEvent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events/k3n8vdczq1mp" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/events/k3n8vdczq1mp")
		}
		if got := r.URL.Query().Get("include_peripheral"); got != "true" {
			t.Errorf("include_peripheral = %q, want %q", got, "true")
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"event": map[string]interface{}{
				"id":                            "k3n8vdczq1mp",
				"title":                         "Volvo AB announces Q3 2026 results",
				"status":                        "active",
				"significance":                  "notable",
				"summary":                       "Volvo AB reported Q3 2026 results...",
				"article_count":                 3,
				"article_ids":                   []string{"h4mzq918kxwc", "q2fnk75dxzmv"},
				"primary_disclosure_article_id": "h4mzq918kxwc",
				"member_figures_disagree":       true,
				"updated_at":                    "2026-10-23T09:20:00.000Z",
				"figure_divergence": map[string]interface{}{
					"groups": []map[string]interface{}{
						{
							"concept":  "operating_profit",
							"basis":    "reported",
							"period":   "2026-Q3",
							"currency": "SEK",
							"labels":   []string{"Rörelseresultatet"},
							"min":      313000000.0,
							"max":      395000000.0,
							"values": []map[string]interface{}{
								{"id": "h4mzq918kxwc", "source": "MFN", "normalized_value": 395000000.0, "value": "395 MSEK"},
							},
						},
					},
				},
				"companies": []map[string]interface{}{
					{"id": "uejazctgchj4", "name": "AB Volvo (Volvo Group)", "ticker": "VOLV-B", "is_primary": true, "association_tier": "principal"},
					{"id": "7f3k9m2p1q8r", "name": "Some Advisor AB", "ticker": nil, "is_primary": false, "association_tier": "peripheral"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("include_peripheral", "true")
	event, _, err := c.GetEvent(context.Background(), "k3n8vdczq1mp", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Summary == "" {
		t.Error("Summary is empty, want the detail summary")
	}
	if len(event.ArticleIDs) != 2 {
		t.Errorf("got %d article IDs, want 2", len(event.ArticleIDs))
	}
	if event.PrimaryDisclosureArticleID != "h4mzq918kxwc" {
		t.Errorf("PrimaryDisclosureArticleID = %q, want %q", event.PrimaryDisclosureArticleID, "h4mzq918kxwc")
	}
	if event.MemberFiguresDisagree == nil || !*event.MemberFiguresDisagree {
		t.Errorf("MemberFiguresDisagree = %v, want true", event.MemberFiguresDisagree)
	}
	if event.FigureDivergence == nil || len(event.FigureDivergence.Groups) != 1 {
		t.Fatalf("FigureDivergence = %+v, want 1 group", event.FigureDivergence)
	}
	g := event.FigureDivergence.Groups[0]
	if g.Concept != "operating_profit" {
		t.Errorf("Concept = %q, want %q", g.Concept, "operating_profit")
	}
	if g.Min == nil || *g.Min != 313000000.0 {
		t.Errorf("Min = %v, want 313000000", g.Min)
	}
	if len(g.Values) != 1 || g.Values[0].ArticleID != "h4mzq918kxwc" {
		t.Errorf("Values = %+v, want one value for article h4mzq918kxwc", g.Values)
	}
	if len(event.Companies) != 2 {
		t.Fatalf("got %d companies, want 2", len(event.Companies))
	}
	if !event.Companies[0].IsPrimary || event.Companies[0].AssociationTier != "principal" {
		t.Errorf("first company = %+v, want primary/principal", event.Companies[0])
	}
	if event.Companies[1].AssociationTier != "peripheral" {
		t.Errorf("second company tier = %q, want peripheral", event.Companies[1].AssociationTier)
	}
	if event.Companies[1].TickerSymbol != "" {
		t.Errorf("unlisted company ticker = %q, want empty", event.Companies[1].TickerSymbol)
	}
}

// A merged or removed event answers with 200 and the tombstone shape rather
// than a 404, carrying no content beyond where it went.
func TestGetEventTombstone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         string
		mergedInto     interface{}
		wantMergedInto string
	}{
		{name: "merged points at the survivor", status: "merged", mergedInto: "v3xdq8fmz1kn", wantMergedInto: "v3xdq8fmz1kn"},
		{name: "removed has nowhere to point", status: "removed", mergedInto: nil, wantMergedInto: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"event": map[string]interface{}{
						"id":          "k3n8vdczq1mp",
						"status":      tt.status,
						"merged_into": tt.mergedInto,
						"updated_at":  "2026-10-24T08:00:00.000Z",
					},
				})
			}))
			t.Cleanup(srv.Close)

			c := NewClient("key", WithBaseURL(srv.URL))
			event, _, err := c.GetEvent(context.Background(), "k3n8vdczq1mp", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !event.IsTombstone() {
				t.Errorf("IsTombstone() = false for a %s event, want true", tt.status)
			}
			if event.MergedInto != tt.wantMergedInto {
				t.Errorf("MergedInto = %q, want %q", event.MergedInto, tt.wantMergedInto)
			}
			if event.Title != "" || event.Significance != "" || event.EventType != nil || event.ArticleCount != 0 {
				t.Errorf("tombstone carried content: title=%q significance=%q type=%v count=%d",
					event.Title, event.Significance, event.EventType, event.ArticleCount)
			}
			if event.MemberFiguresDisagree != nil {
				t.Errorf("MemberFiguresDisagree = %v on a tombstone, want nil", *event.MemberFiguresDisagree)
			}
		})
	}
}

func TestListEventTypes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/event_types" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/event_types")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"event_types": []map[string]interface{}{
				{
					"id":       "evt_earnrep01",
					"slug":     "earnings-report",
					"label":    "Earnings Report",
					"category": "earnings",
					"summary":  "A company's scheduled quarterly, interim or annual results without consensus framing.",
				},
				{
					"id":       "evt_execapp01",
					"slug":     "executive-appointment",
					"label":    "Executive Appointment",
					"category": "executive_changes",
				},
			},
			"pagination": map[string]interface{}{"count": 2, "next_cursor": nil},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	types, pagination, _, err := c.ListEventTypes(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) != 2 {
		t.Fatalf("got %d event types, want 2", len(types))
	}
	if types[0].Slug != "earnings-report" || types[0].Category != "earnings" {
		t.Errorf("first type = %+v, want earnings-report / earnings", types[0])
	}
	if types[0].Summary == "" {
		t.Error("Summary is empty, want the taxonomy summary")
	}
	if types[1].Category != "executive_changes" {
		t.Errorf("second type category = %q, want %q", types[1].Category, "executive_changes")
	}
	if pagination.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", pagination.NextCursor)
	}
}

func TestListCompanyEvents(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/companies/VOLV-B/events" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/companies/VOLV-B/events")
		}
		if got := r.URL.Query().Get("window"); got != "30d" {
			t.Errorf("window = %q, want %q", got, "30d")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []map[string]interface{}{
				{"id": "k3n8vdczq1mp", "title": "Q3 2026 results", "status": "active", "updated_at": "2026-10-23T09:20:00.000Z"},
			},
			"pagination": map[string]interface{}{"count": 1, "next_cursor": nil},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	params := url.Values{}
	params.Set("window", "30d")
	events, _, _, err := c.ListCompanyEvents(context.Background(), "VOLV-B", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].ID != "k3n8vdczq1mp" {
		t.Errorf("events = %+v, want the single Q3 event", events)
	}
}

// event_id is additive on the article and story feeds: it resolves to an event
// when one has been assigned, and stays empty otherwise.
func TestArticleAndStoryCarryEventID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/articles":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"articles": []map[string]interface{}{
					{"id": "h4mzq918kxwc", "title": "Volvo Q3", "event_id": "k3n8vdczq1mp"},
					{"id": "q2fnk75dxzmv", "title": "Unassigned", "event_id": nil},
				},
				"pagination": map[string]interface{}{"count": 2},
			})
		case "/stories":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"stories": []map[string]interface{}{
					{"id": "8fzq2rmtnk5c", "title": "Volvo Q3 story", "event_id": "k3n8vdczq1mp"},
				},
				"pagination": map[string]interface{}{"count": 1},
			})
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))

	articles, _, _, err := c.ListArticles(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if articles[0].EventID != "k3n8vdczq1mp" {
		t.Errorf("article EventID = %q, want %q", articles[0].EventID, "k3n8vdczq1mp")
	}
	if articles[1].EventID != "" {
		t.Errorf("unassigned article EventID = %q, want empty", articles[1].EventID)
	}

	stories, _, _, err := c.ListStories(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stories[0].EventID != "k3n8vdczq1mp" {
		t.Errorf("story EventID = %q, want %q", stories[0].EventID, "k3n8vdczq1mp")
	}
}

// The events cursor walk is at-least-once: a merge landing mid-walk moves the
// surviving event earlier in the newest-first sort, so a later page can re-emit
// a row an earlier page already returned. The spec puts the dedupe obligation on
// the client, so --all must not double-count.
func TestListAllEventsDedupesRepeatedRows(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events": []map[string]interface{}{
					{"id": "e1", "title": "One", "status": "active"},
					{"id": "e2", "title": "Two", "status": "active"},
				},
				"pagination": map[string]interface{}{"count": 2, "next_cursor": "next"},
			})
			return
		}
		// e2 comes back a second time, as a mid-walk merge can cause.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []map[string]interface{}{
				{"id": "e2", "title": "Two", "status": "active"},
				{"id": "e3", "title": "Three", "status": "active"},
			},
			"pagination": map[string]interface{}{"count": 2},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	events, _, err := c.ListAllEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 distinct", len(events))
	}
	seen := map[string]int{}
	for _, e := range events {
		seen[e.ID]++
	}
	for _, id := range []string{"e1", "e2", "e3"} {
		if seen[id] != 1 {
			t.Errorf("event %s appeared %d times, want 1", id, seen[id])
		}
	}
}

// An updated_at walk is ordered ascending, so an event merged mid-walk is
// re-emitted as a tombstone after the cursor has passed its original row. That
// tombstone is the whole point of the sync, so the later copy has to win: a
// first-wins dedupe would hand back the stale active row and leave a mirror
// holding an event the API no longer serves.
func TestListAllEventsKeepsTheLatestCopyOfARepeatedRow(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events": []map[string]interface{}{
					{"id": "e1", "title": "Merged away", "status": "active", "article_count": 3},
				},
				"pagination": map[string]interface{}{"count": 1, "next_cursor": "next"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []map[string]interface{}{
				{"id": "e1", "status": "merged", "merged_into": "e9"},
			},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	events, _, err := c.ListAllEvents(context.Background(), url.Values{"updated_after": {"2026-01-01T00:00:00Z"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if !events[0].IsTombstone() {
		t.Errorf("status = %q, want the merged tombstone from the later page", events[0].Status)
	}
	if events[0].MergedInto != "e9" {
		t.Errorf("MergedInto = %q, want e9", events[0].MergedInto)
	}
}

// The same walk backs the nested company feed.
func TestListAllCompanyEventsDedupesRepeatedRows(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events":     []map[string]interface{}{{"id": "e1", "status": "active"}},
				"pagination": map[string]interface{}{"count": 1, "next_cursor": "next"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events":     []map[string]interface{}{{"id": "e1", "status": "active"}},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	events, _, err := c.ListAllCompanyEvents(context.Background(), "VOLV-B", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("got %d events, want 1 after dedupe", len(events))
	}
}

// Articles and stories sort on an immutable published_at, so their walks are
// exactly-once and ListAll keeps every row it is handed. The fixture repeats one
// id across pages — the shape ListAllDeduped collapses — so this fails if the
// article walk is ever switched to the deduping one.
func TestListAllArticlesDoesNotDedupe(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") == "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"articles":   []map[string]interface{}{{"id": "a1", "title": "Same headline"}},
				"pagination": map[string]interface{}{"count": 1, "next_cursor": "next"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"articles":   []map[string]interface{}{{"id": "a1", "title": "Same headline"}},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	articles, _, err := c.ListAllArticles(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("got %d articles, want 2 — the repeated id must be kept", len(articles))
	}
}

// truncated is unique to the semantic events search: it means further matches
// exist that no cursor can reach, so it must not be dropped on the floor.
func TestListEventsTruncated(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events":     []map[string]interface{}{{"id": "e1", "status": "active"}},
			"pagination": map[string]interface{}{"count": 1},
			"truncated":  true,
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	_, _, resp, err := c.ListEvents(context.Background(), url.Values{"mode": {"semantic"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Truncated == nil || !*resp.Truncated {
		t.Errorf("Truncated = %v, want true", resp.Truncated)
	}
}

// Absent stays absent, so a consumer can tell "nothing beyond this page" from
// "this endpoint does not report truncation at all".
func TestTruncatedAbsent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events":     []map[string]interface{}{{"id": "e1", "status": "active"}},
			"pagination": map[string]interface{}{"count": 1},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	_, _, resp, err := c.ListEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Truncated != nil {
		t.Errorf("Truncated = %v, want nil when the key is absent", *resp.Truncated)
	}
}

// A legitimate article_count of 0 has to survive into JSON output rather than
// being elided, so "no live articles" stays distinct from "field not sent".
func TestEventArticleCountZeroSurvivesJSON(t *testing.T) {
	t.Parallel()
	out, err := json.Marshal(Event{ID: "e1", Status: "active", ArticleCount: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var round map[string]interface{}
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := round["article_count"]; !ok {
		t.Errorf("article_count was elided from %s", out)
	}
}
