package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// bodyServer serves a fixed JSON body for every request.
func bodyServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGet_PlanLimited(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want *bool
	}{
		{
			name: "plan_limited true is surfaced",
			body: `{"items":[],"plan_limited":true}`,
			want: boolPtr(true),
		},
		{
			name: "plan_limited false stays distinct from absent",
			body: `{"items":[],"plan_limited":false}`,
			want: boolPtr(false),
		},
		{
			name: "absent key yields nil",
			body: `{"items":[]}`,
			want: nil,
		},
		{
			name: "non-object body yields nil without failing the request",
			body: `[1,2,3]`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := bodyServer(t, tt.body)
			c := NewClient("key", WithBaseURL(srv.URL))

			resp, err := c.Get(context.Background(), "/items", nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPlanLimited(t, resp.PlanLimited, tt.want)
		})
	}
}

// The payload must still decode normally now that the body is buffered to
// extract response-level metadata.
func TestGet_PlanLimitedDoesNotDisturbDecode(t *testing.T) {
	t.Parallel()
	srv := bodyServer(t, `{"name":"test","plan_limited":true}`)
	c := NewClient("key", WithBaseURL(srv.URL))

	var got struct {
		Name string `json:"name"`
	}
	resp, err := c.Get(context.Background(), "/test", nil, &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
	assertPlanLimited(t, resp.PlanLimited, boolPtr(true))
}

func TestListPage_PlanLimited(t *testing.T) {
	t.Parallel()
	srv := bodyServer(t, `{"items":[{"id":"1","name":"First"}],"pagination":{"count":1},"plan_limited":true}`)
	c := NewClient("key", WithBaseURL(srv.URL))

	items, _, resp, err := ListPage[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	assertPlanLimited(t, resp.PlanLimited, boolPtr(true))
}

// A restriction on any single page makes the whole aggregated result
// incomplete, so ListAll must not let a later unrestricted page clear the flag.
func TestListAll_PlanLimitedIsStickyAcrossPages(t *testing.T) {
	t.Parallel()

	var page int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page++
		if page == 1 {
			_, _ = w.Write([]byte(`{"items":[{"id":"1"}],"pagination":{"count":1,"next_cursor":"next"},"plan_limited":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"2"}],"pagination":{"count":1}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	items, resp, err := ListAll[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	assertPlanLimited(t, resp.PlanLimited, boolPtr(true))
}

func TestListAll_PlanLimitedAbsentOnAllPages(t *testing.T) {
	t.Parallel()
	srv := bodyServer(t, `{"items":[{"id":"1"}],"pagination":{"count":1}}`)
	c := NewClient("key", WithBaseURL(srv.URL))

	_, resp, err := ListAll[testItem](context.Background(), c, "/items", nil, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPlanLimited(t, resp.PlanLimited, nil)
}

func TestSearch_PlanLimitedHoistedIntoResults(t *testing.T) {
	t.Parallel()
	srv := bodyServer(t, `{"results":{"articles":[{"id":"a1"}],"stories":[],"companies":[]},"plan_limited":true}`)
	c := NewClient("key", WithBaseURL(srv.URL))

	results, _, err := c.Search(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Articles) != 1 {
		t.Fatalf("got %d articles, want 1", len(results.Articles))
	}
	assertPlanLimited(t, results.PlanLimited, boolPtr(true))
}

func TestSearch_PlanLimitedAbsent(t *testing.T) {
	t.Parallel()
	srv := bodyServer(t, `{"results":{"articles":[],"stories":[],"companies":[]}}`)
	c := NewClient("key", WithBaseURL(srv.URL))

	results, _, err := c.Search(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPlanLimited(t, results.PlanLimited, nil)
}

func boolPtr(b bool) *bool { return &b }

func assertPlanLimited(t *testing.T, got, want *bool) {
	t.Helper()
	switch {
	case want == nil && got != nil:
		t.Errorf("PlanLimited = %v, want nil (absent)", *got)
	case want != nil && got == nil:
		t.Errorf("PlanLimited = nil, want %v", *want)
	case want != nil && got != nil && *want != *got:
		t.Errorf("PlanLimited = %v, want %v", *got, *want)
	}
}
