package cmd

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/spf13/cobra"
)

// setFlags sets each flag on the command, failing the test if one is not
// registered. Using the real command's flag set is the point: buildEventParams
// forwards only registered flags, so a hand-rolled flag set in the test would
// test the test.
func setFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for k, v := range flags {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("setting --%s: %v", k, err)
		}
	}
}

// Every filter the flat events feed offers must reach the API under its
// documented parameter name. This exercises name mapping only: --window with
// --last-article-after is a combination the API rejects.
func TestBuildEventParams(t *testing.T) {
	restoreFlags(t, eventsListCmd)

	cmd := *eventsListCmd
	setFlags(t, &cmd, map[string]string{
		"q":                     "battery",
		"sort":                  "significance",
		"event-type":            "earnings-report,acquisition-announced",
		"significance":          "notable,routine",
		"ticker":                "VOLV-B,ERIC-B",
		"company":               "cmpny1234567",
		"country":               "SE,NO",
		"category":              "cat_earnings1",
		"sources":               "src123456789",
		"source-type":           "stock_exchange",
		"content-type":          "press_release",
		"exchange":              "XSTO,XCSE",
		"market":                "SE",
		"domicile":              "FI",
		"index":                 "OMXS30",
		"sector":                "Industrials",
		"watchlist-id":          "wl_8fk2a1b3c4d5",
		"first-reported-after":  "2026-01-01T00:00:00Z",
		"first-reported-before": "2026-12-31T23:59:59Z",
		"last-article-after":    "2026-03-01T00:00:00Z",
		"last-article-before":   "2026-03-02T00:00:00Z",
		"window":                "24h",
		"updated-after":         "2026-03-01T00:00:00Z",
		"related-to":            "evtrswagsh01",
		"listed":                "true",
		"include-peripheral":    "true",
		"semantic":              "true",
		"limit":                 "25",
		"cursor":                "opaque",
		"fields":                "title,status,article_count",
	})

	want := map[string]string{
		"q":                     "battery",
		"sort":                  "significance",
		"event_type":            "earnings-report,acquisition-announced",
		"significance":          "notable,routine",
		"ticker":                "VOLV-B,ERIC-B",
		"company":               "cmpny1234567",
		"country":               "SE,NO",
		"category":              "cat_earnings1",
		"sources":               "src123456789",
		"source_type":           "stock_exchange",
		"content_type":          "press_release",
		"exchange":              "XSTO,XCSE",
		"market":                "SE",
		"domicile":              "FI",
		"index":                 "OMXS30",
		"sector":                "Industrials",
		"watchlist":             "wl_8fk2a1b3c4d5",
		"first_reported_after":  "2026-01-01T00:00:00Z",
		"first_reported_before": "2026-12-31T23:59:59Z",
		"last_article_after":    "2026-03-01T00:00:00Z",
		"last_article_before":   "2026-03-02T00:00:00Z",
		"window":                "24h",
		"updated_after":         "2026-03-01T00:00:00Z",
		"related_to":            "evtrswagsh01",
		"listed":                "true",
		"include_peripheral":    "true",
		"mode":                  "semantic",
		"limit":                 "25",
		"cursor":                "opaque",
		"fields":                "title,status,article_count",
	}

	params := buildEventParams(&cmd)
	for key, wantVal := range want {
		if got := params.Get(key); got != wantVal {
			t.Errorf("params[%q] = %q, want %q", key, got, wantVal)
		}
	}
	for key := range params {
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected param %q = %q", key, params.Get(key))
		}
	}
}

// An unset flag must stay off the request: sending listed=false or
// include_peripheral=false would be a different request from sending neither.
func TestBuildEventParamsSendsNothingByDefault(t *testing.T) {
	restoreFlags(t, eventsListCmd)

	cmd := *eventsListCmd
	if params := buildEventParams(&cmd); len(params) != 0 {
		t.Errorf("params = %v, want empty", params)
	}
}

// The nested company route rejects the filters that name another company or
// listing, and has no published_after/published_before. buildEventParams
// forwards only registered flags, so not registering them is what keeps them
// off the wire — assert they are absent rather than trusting the builder.
func TestCompanyEventsRouteOmitsRejectedFilters(t *testing.T) {
	rejected := []string{
		"ticker", "company", "watchlist-id", "listed", "exchange",
		"market", "domicile", "index", "sector", "semantic",
		"published-after", "published-before",
	}
	for _, flag := range rejected {
		if f := companiesEventsCmd.Flags().Lookup(flag); f != nil {
			t.Errorf("nfn companies events registers --%s, which the route rejects", flag)
		}
	}
}

// Conversely, every filter the nested route does accept must be offered.
func TestCompanyEventsRouteOffersAcceptedFilters(t *testing.T) {
	restoreFlags(t, companiesEventsCmd)

	cmd := *companiesEventsCmd
	setFlags(t, &cmd, map[string]string{
		"q":                     "battery",
		"sort":                  "latest",
		"event-type":            "earnings-report",
		"significance":          "notable",
		"country":               "SE",
		"category":              "cat_earnings1",
		"sources":               "src123456789",
		"source-type":           "wire_service",
		"content-type":          "news",
		"first-reported-after":  "2026-01-01T00:00:00Z",
		"first-reported-before": "2026-12-31T23:59:59Z",
		"last-article-before":   "2026-03-02T00:00:00Z",
		"window":                "30d",
		"updated-after":         "2026-03-01T00:00:00Z",
		"related-to":            "evtrswagsh01",
		"include-peripheral":    "true",
		"limit":                 "10",
		"cursor":                "opaque",
		"fields":                "title,status",
	})

	params := buildEventParams(&cmd)
	for _, param := range []string{
		"q", "sort", "event_type", "significance", "country", "category", "sources",
		"source_type", "content_type", "first_reported_after", "first_reported_before",
		"last_article_before", "window", "updated_after", "related_to",
		"include_peripheral", "limit", "cursor", "fields",
	} {
		if params.Get(param) == "" {
			t.Errorf("params[%q] is empty, want the flag forwarded", param)
		}
	}
	for _, param := range []string{
		"ticker", "company", "watchlist", "listed", "exchange", "market",
		"domicile", "index", "sector", "mode", "published_after", "published_before",
	} {
		if _, ok := params[param]; ok {
			t.Errorf("params[%q] = %q, want it absent on the company route", param, params.Get(param))
		}
	}
}

func TestEventReportedDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.Event
		want  string
	}{
		{name: "trims a timestamp to its date", event: api.Event{FirstReportedAt: "2026-10-22T05:00:00.000Z"}, want: "2026-10-22"},
		{name: "short string returned as-is without panic", event: api.Event{FirstReportedAt: "2026"}, want: "2026"},
		{name: "tombstone carries no timestamp", event: api.Event{Status: "merged"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eventReportedDate(tt.event); got != tt.want {
				t.Errorf("eventReportedDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A tombstone omits article_count rather than reporting zero, so the column has
// to stay blank instead of claiming the event has no articles.
func TestEventArticleCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.Event
		want  string
	}{
		{name: "live event with articles", event: api.Event{Status: "active", ArticleCount: 3}, want: "3"},
		{name: "live event with none visible", event: api.Event{Status: "closed"}, want: "0"},
		{name: "merged tombstone", event: api.Event{Status: "merged"}, want: ""},
		{name: "removed tombstone", event: api.Event{Status: "removed"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eventArticleCount(tt.event); got != tt.want {
				t.Errorf("eventArticleCount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventSummaryLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.Event
		want  string
	}{
		{
			name:  "live event names its title",
			event: api.Event{ID: "evt1", Status: "active", Title: "Volvo AB announces Q3 2026 results"},
			want:  "event: Volvo AB announces Q3 2026 results",
		},
		{
			name:  "merged tombstone names the survivor",
			event: api.Event{ID: "evt1", Status: "merged", MergedInto: "evt2"},
			want:  "event evt1: merged into evt2",
		},
		{
			name:  "merged tombstone without a survivor",
			event: api.Event{ID: "evt1", Status: "merged"},
			want:  "event evt1: merged",
		},
		{
			name:  "removed tombstone",
			event: api.Event{ID: "evt1", Status: "removed"},
			want:  "event evt1: removed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eventSummaryLine(&tt.event); got != tt.want {
				t.Errorf("eventSummaryLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventTombstoneNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.Event
		want  []string
	}{
		{
			name:  "merged points at the survivor",
			event: api.Event{ID: "evt1", Status: "merged", MergedInto: "evt2"},
			want:  []string{"merged into evt2", "nfn events get evt2"},
		},
		{
			name:  "merged without a survivor says so",
			event: api.Event{ID: "evt1", Status: "merged"},
			want:  []string{"does not name"},
		},
		{
			name:  "removed explains the terminal state",
			event: api.Event{ID: "evt1", Status: "removed"},
			want:  []string{"removed"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := eventTombstoneNote(&tt.event)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("eventTombstoneNote() = %q, want it to mention %q", got, want)
				}
			}
		})
	}
}

func TestEventCompanyLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		company api.EventCompany
		want    string
	}{
		{
			name:    "primary principal company",
			company: api.EventCompany{Name: "Volvo AB", TickerSymbol: "VOLV-B", IsPrimary: true, AssociationTier: "principal"},
			want:    "Volvo AB (VOLV-B) [primary, principal]",
		},
		{
			name:    "peripheral company",
			company: api.EventCompany{Name: "Advisory Partners", AssociationTier: "peripheral"},
			want:    "Advisory Partners [peripheral]",
		},
		{
			name:    "unlisted company with no tier reported",
			company: api.EventCompany{Name: "Private Holdco"},
			want:    "Private Holdco",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eventCompanyLabel(tt.company); got != tt.want {
				t.Errorf("eventCompanyLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventDivergenceLines(t *testing.T) {
	t.Parallel()

	lo, hi := 3.95e8, 4.02e8
	// A loss: the negative bounds are why the range reads "to" and not a dash.
	lossLo, lossHi := -1.2e5, -1.0e5
	fd := &api.FigureDivergence{Groups: []api.FigureDivergenceGroup{
		{
			Concept: "net_sales", Period: "Q3 2026", Currency: "SEK",
			Min: &lo, Max: &hi,
			Values: []api.FigureValue{{ArticleID: "a1"}, {ArticleID: "a2"}},
		},
		{
			Concept: "operating_profit", Period: "2026-Q1", Currency: "SEK",
			Min: &lossLo, Max: &lossHi,
			Values: []api.FigureValue{{ArticleID: "a1"}, {ArticleID: "a2"}, {ArticleID: "a3"}},
		},
		{Concept: "eps", Values: []api.FigureValue{{ArticleID: "a1"}}},
	}}

	got := eventDivergenceLines(fd)
	want := []string{
		"net_sales Q3 2026: 395000000 to 402000000 SEK, 2 values",
		"operating_profit 2026-Q1: -120000 to -100000 SEK, 3 values",
		"eps: 1 value",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}

	if got := eventDivergenceLines(nil); got != nil {
		t.Errorf("eventDivergenceLines(nil) = %v, want nil", got)
	}
}

// A semantic search reports truncated when further matches exist that no cursor
// can reach. Presenting that page as complete is the failure to avoid.
func TestEventsList_TruncatedEndToEnd(t *testing.T) {
	body := `{"events":[{"id":"evt1","title":"Saab lands order","status":"active","significance":"notable",` +
		`"article_count":2,"first_reported_at":"2026-08-01T00:00:00Z","updated_at":"2026-08-01T00:00:00Z",` +
		`"distance":0.1842}],"pagination":{"count":1},"truncated":true}`

	t.Run("table prints the note", func(t *testing.T) {
		out := runListCommand(t, eventsListCmd, fixedBody(body), "table", "--q", "defence orders", "--semantic")
		if !strings.Contains(out, "beyond the search horizon") {
			t.Errorf("output = %q, want the truncated note", out)
		}
		if !strings.Contains(out, "0.1842") {
			t.Errorf("output = %q, want the distance value", out)
		}
	})

	t.Run("json carries the flag and warns in the summary", func(t *testing.T) {
		out := runListCommand(t, eventsListCmd, fixedBody(body), "json", "--q", "defence orders", "--semantic")

		var env struct {
			Summary string `json:"summary"`
			Data    struct {
				Events    []api.Event `json:"events"`
				Truncated *bool       `json:"truncated"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, out)
		}
		if env.Data.Truncated == nil || !*env.Data.Truncated {
			t.Errorf("truncated = %v, want true", env.Data.Truncated)
		}
		if !strings.Contains(env.Summary, "beyond the search horizon") {
			t.Errorf("summary = %q, want it to warn about the horizon", env.Summary)
		}
	})
}

// An ordinary listing reports no horizon, so neither the note nor the key
// should appear.
func TestEventsList_NoTruncatedNoteWithoutTheFlag(t *testing.T) {
	body := `{"events":[{"id":"evt1","title":"Saab lands order","status":"active","article_count":2,` +
		`"first_reported_at":"2026-08-01T00:00:00Z","updated_at":"2026-08-01T00:00:00Z"}],"pagination":{"count":1}}`

	out := runListCommand(t, eventsListCmd, fixedBody(body), "json", "--limit", "25")

	var env struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, out)
	}
	if _, ok := env.Data["truncated"]; ok {
		t.Errorf("data carries truncated = %s, want the key absent", env.Data["truncated"])
	}
}

// The Status column earns its width only when a sync walk returns tombstones,
// whose other columns are empty.
func TestEventsList_StatusColumnOnlyForTombstones(t *testing.T) {
	live := `{"events":[{"id":"evt1","title":"Saab lands order","status":"active","article_count":2,` +
		`"first_reported_at":"2026-08-01T00:00:00Z","updated_at":"2026-08-01T00:00:00Z"}],"pagination":{"count":1}}`
	sync := `{"events":[{"id":"evt1","status":"merged","merged_into":"evt2","updated_at":"2026-08-02T00:00:00Z"}],` +
		`"pagination":{"count":1}}`

	t.Run("absent on a live listing", func(t *testing.T) {
		out := runListCommand(t, eventsListCmd, fixedBody(live), "table", "--limit", "25")
		if strings.Contains(strings.ToUpper(out), "STATUS") {
			t.Errorf("output = %q, want no Status column", out)
		}
	})

	t.Run("present on a sync walk", func(t *testing.T) {
		out := runListCommand(t, eventsListCmd, fixedBody(sync), "table", "--updated-after", "2026-08-01T00:00:00Z")
		if !strings.Contains(strings.ToUpper(out), "STATUS") {
			t.Errorf("output = %q, want a Status column", out)
		}
		if !strings.Contains(out, "merged") {
			t.Errorf("output = %q, want the merged status", out)
		}
	})
}

// A tombstone must not be rendered as an event whose content went missing.
func TestEventsGet_Tombstone(t *testing.T) {
	t.Run("merged names the survivor", func(t *testing.T) {
		body := `{"event":{"id":"evt1","status":"merged","merged_into":"evt2","updated_at":"2026-08-02T00:00:00Z"}}`

		out := runCommandWithArgs(t, eventsGetCmd, fixedBody(body), "table", []string{"evt1"})

		if !strings.Contains(out, "Merged Into") {
			t.Errorf("output = %q, want the Merged Into row", out)
		}
		if !strings.Contains(out, "nfn events get evt2") {
			t.Errorf("output = %q, want a pointer to follow the merge", out)
		}
		for _, unwanted := range []string{"Title", "Summary", "Significance", "Articles"} {
			if strings.Contains(out, unwanted) {
				t.Errorf("output = %q, want no %q row on a tombstone", out, unwanted)
			}
		}
	})

	// A removal has no survivor, so the row would always be blank.
	t.Run("removed omits the Merged Into row", func(t *testing.T) {
		body := `{"event":{"id":"evt1","status":"removed","updated_at":"2026-08-02T00:00:00Z"}}`

		out := runCommandWithArgs(t, eventsGetCmd, fixedBody(body), "table", []string{"evt1"})

		if strings.Contains(out, "Merged Into") {
			t.Errorf("output = %q, want no Merged Into row on a removal", out)
		}
		if !strings.Contains(out, "removed") {
			t.Errorf("output = %q, want the removed status and note", out)
		}
	})
}

func TestEventsGet_IncludePeripheral(t *testing.T) {
	var gotParams url.Values
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"event":{"id":"evt1","title":"Volvo Q3","status":"active","article_count":3,` +
			`"updated_at":"2026-08-02T00:00:00Z","companies":[` +
			`{"id":"c1","name":"Volvo AB","ticker":"VOLV-B","is_primary":true,"association_tier":"principal"},` +
			`{"id":"c2","name":"Advisory Partners","association_tier":"peripheral"}]}}`))
	}

	out := runCommandWithArgs(t, eventsGetCmd, handler, "table", []string{"evt1"}, "--include-peripheral")

	if got := gotParams.Get("include_peripheral"); got != "true" {
		t.Errorf("include_peripheral = %q, want %q", got, "true")
	}
	if !strings.Contains(out, "peripheral") {
		t.Errorf("output = %q, want the peripheral company's tier", out)
	}
}

// The nested route renders through the same renderer as the flat feed, under its
// own breadcrumb path.
func TestCompaniesEvents_EndToEnd(t *testing.T) {
	var gotPath string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[{"id":"evt1","title":"Volvo Q3","status":"active","significance":"notable",` +
			`"article_count":3,"first_reported_at":"2026-08-01T00:00:00Z","updated_at":"2026-08-01T00:00:00Z"}],` +
			`"pagination":{"count":1}}`))
	}

	out := runCommandWithArgs(t, companiesEventsCmd, handler, "json", []string{"VOLV-B"}, "--significance", "notable")

	if !strings.HasSuffix(gotPath, "/companies/VOLV-B/events") {
		t.Errorf("path = %q, want the nested company events route", gotPath)
	}

	var env struct {
		Breadcrumbs []struct {
			Command string `json:"command"`
		} `json:"breadcrumbs"`
		Data struct {
			Events []api.Event `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, out)
	}
	if len(env.Data.Events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(env.Data.Events))
	}
	if len(env.Breadcrumbs) == 0 {
		t.Error("breadcrumbs are empty, want the company events follow-ups")
	}
}

func TestEventTypes_EndToEnd(t *testing.T) {
	body := `{"event_types":[{"id":"evt_earnrep01","slug":"earnings-report","label":"Earnings Report",` +
		`"category":"earnings","summary":"A company's scheduled results."}],"pagination":{"count":1}}`

	out := runListCommand(t, eventTypesCmd, fixedBody(body), "table")

	for _, want := range []string{"earnings-report", "Earnings Report", "earnings"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

// The article and story detail views gained an event_id, which is the hop from a
// piece of coverage to the development it reports. It lives here with the rest
// of the events surface rather than in articles_test.go / stories_test.go.
func TestArticleAndStoryDetailShowTheEvent(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		withID  string
		without string
	}{
		{
			name:    "articles get",
			cmd:     articlesGetCmd,
			withID:  `{"article":{"id":"a1","title":"Volvo reports Q3","article_url":"https://example.com/a1","content_type":"news","published_at":"2026-08-01T00:00:00Z","country":"SE","event_id":"evt1"}}`,
			without: `{"article":{"id":"a1","title":"Volvo reports Q3","article_url":"https://example.com/a1","content_type":"news","published_at":"2026-08-01T00:00:00Z","country":"SE"}}`,
		},
		{
			name:    "stories get",
			cmd:     storiesGetCmd,
			withID:  `{"story":{"id":"s1","title":"Volvo Q3 coverage","published_at":"2026-08-01T00:00:00Z","country":"SE","event_id":"evt1"}}`,
			without: `{"story":{"id":"s1","title":"Volvo Q3 coverage","published_at":"2026-08-01T00:00:00Z","country":"SE"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" shows the event", func(t *testing.T) {
			out := runCommandWithArgs(t, tt.cmd, fixedBody(tt.withID), "table", []string{"x1"})
			if !strings.Contains(out, "evt1") {
				t.Errorf("output = %q, want the event id", out)
			}
		})

		// Unassigned coverage has no event, and a blank Event row would read as
		// one the CLI failed to fetch.
		t.Run(tt.name+" omits the row when unassigned", func(t *testing.T) {
			out := runCommandWithArgs(t, tt.cmd, fixedBody(tt.without), "table", []string{"x1"})
			if strings.Contains(out, "Event") {
				t.Errorf("output = %q, want no Event row", out)
			}
		})
	}
}

// TestValidateEventTypeRejectsUnderscoredSlugs covers the guard that spends no
// request on a calendar slug. The two vocabularies overlap in a few spellings,
// so the message has to name the offending values and agree its verb with how
// many there are.
func TestValidateEventTypeRejectsUnderscoredSlugs(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantErr   bool
		wantParts []string
	}{
		{name: "empty", eventType: "", wantErr: false},
		{name: "hyphenated", eventType: "ceo-departure", wantErr: false},
		{name: "several hyphenated", eventType: "ceo-departure,contract-won", wantErr: false},
		{
			name:      "one underscored",
			eventType: "earnings_report",
			wantErr:   true,
			wantParts: []string{`"earnings_report"`, " is underscored"},
		},
		{
			name:      "two underscored",
			eventType: "earnings_report,dividend_payment",
			wantErr:   true,
			wantParts: []string{`"earnings_report"`, `"dividend_payment"`, " are underscored"},
		},
		{
			name:      "mixed names only the underscored one",
			eventType: "ceo-departure,earnings_report",
			wantErr:   true,
			wantParts: []string{`"earnings_report"`, " is underscored"},
		},
		{
			name:      "surrounding spaces are trimmed",
			eventType: "ceo-departure, earnings_report ",
			wantErr:   true,
			wantParts: []string{`"earnings_report"`, " is underscored"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreFlags(t, eventsListCmd)
			cmd := *eventsListCmd
			if err := cmd.Flags().Set("event-type", tt.eventType); err != nil {
				t.Fatalf("Set(event-type, %q) = %v", tt.eventType, err)
			}

			err := validateEventType(&cmd)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("validateEventType() = %v, want nil for %q", err, tt.eventType)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateEventType() = nil, want an error for %q", tt.eventType)
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q does not contain %q", err, part)
				}
			}
			// The guard exists to point at the other command, not just to refuse.
			if !strings.Contains(err.Error(), "nfn calendar list --event-type") {
				t.Errorf("error %q does not point at the calendar command", err)
			}
			if strings.Contains(err.Error(), `"ceo-departure"`) {
				t.Errorf("error %q names a hyphenated slug it should accept", err)
			}
		})
	}
}
