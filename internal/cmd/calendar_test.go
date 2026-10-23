package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
)

func TestBuildCalendarParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		flags  map[string]string
		bools  map[string]bool
		ints   map[string]int
		wantKV map[string]string
	}{
		{
			name:   "empty flags produce empty params",
			flags:  map[string]string{},
			wantKV: map[string]string{},
		},
		{
			name:   "ticker flag",
			flags:  map[string]string{"ticker": "VOLV-B"},
			wantKV: map[string]string{"ticker": "VOLV-B"},
		},
		{
			name:   "company flag",
			flags:  map[string]string{"company": "cmpny1234567"},
			wantKV: map[string]string{"company": "cmpny1234567"},
		},
		{
			name:   "event-type maps to event_type",
			flags:  map[string]string{"event-type": "earnings_report,dividend"},
			wantKV: map[string]string{"event_type": "earnings_report,dividend"},
		},
		{
			name:   "scheduled-after maps to scheduled_after",
			flags:  map[string]string{"scheduled-after": "2026-01-01"},
			wantKV: map[string]string{"scheduled_after": "2026-01-01"},
		},
		{
			name:   "scheduled-before maps to scheduled_before",
			flags:  map[string]string{"scheduled-before": "2026-12-31"},
			wantKV: map[string]string{"scheduled_before": "2026-12-31"},
		},
		{
			name:   "bare-date scheduled window passes through verbatim",
			flags:  map[string]string{"scheduled-after": "2026-10-01", "scheduled-before": "2026-10-31"},
			wantKV: map[string]string{"scheduled_after": "2026-10-01", "scheduled_before": "2026-10-31"},
		},
		{
			name:   "datetime scheduled window passes through verbatim",
			flags:  map[string]string{"scheduled-after": "2026-10-01T00:00:00Z", "scheduled-before": "2026-11-01T00:00:00Z"},
			wantKV: map[string]string{"scheduled_after": "2026-10-01T00:00:00Z", "scheduled_before": "2026-11-01T00:00:00Z"},
		},
		{
			name:   "datetime with UTC offset passes through verbatim",
			flags:  map[string]string{"scheduled-after": "2026-10-01T00:00:00+02:00"},
			wantKV: map[string]string{"scheduled_after": "2026-10-01T00:00:00+02:00"},
		},
		{
			name:   "updated-after maps to updated_after",
			flags:  map[string]string{"updated-after": "2026-03-01T00:00:00Z"},
			wantKV: map[string]string{"updated_after": "2026-03-01T00:00:00Z"},
		},
		{
			name:   "watchlist-id maps to watchlist",
			flags:  map[string]string{"watchlist-id": "a1b2c3d4e5f6"},
			wantKV: map[string]string{"watchlist": "a1b2c3d4e5f6"},
		},
		{
			name:   "exchange, index, sector, status",
			flags:  map[string]string{"exchange": "XSTO", "index": "OMXS30", "sector": "Financials", "status": "scheduled"},
			wantKV: map[string]string{"exchange": "XSTO", "index": "OMXS30", "sector": "Financials", "status": "scheduled"},
		},
		{
			name:   "market-segment maps to market_segment",
			flags:  map[string]string{"market-segment": "large_cap,mid_cap"},
			wantKV: map[string]string{"market_segment": "large_cap,mid_cap"},
		},
		{
			name:   "market flag",
			flags:  map[string]string{"market": "SE"},
			wantKV: map[string]string{"market": "SE"},
		},
		{
			name:   "domicile flag",
			flags:  map[string]string{"domicile": "FI"},
			wantKV: map[string]string{"domicile": "FI"},
		},
		{
			name:   "market and domicile accept comma-separated lists",
			flags:  map[string]string{"market": "SE,NO", "domicile": "FI,DK"},
			wantKV: map[string]string{"market": "SE,NO", "domicile": "FI,DK"},
		},
		{
			name:   "exchange accepts comma-separated MIC codes",
			flags:  map[string]string{"exchange": "XSTO,XCSE"},
			wantKV: map[string]string{"exchange": "XSTO,XCSE"},
		},
		{
			name:   "country, market, and domicile are independent filters",
			flags:  map[string]string{"country": "SE", "market": "NO", "domicile": "DK"},
			wantKV: map[string]string{"country": "SE", "market": "NO", "domicile": "DK"},
		},
		{
			name:   "limit int",
			ints:   map[string]int{"limit": 25},
			wantKV: map[string]string{"limit": "25"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := *calendarListCmd
			cmd.ResetFlags()
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("company", "", "")
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("market", "", "")
			cmd.Flags().String("domicile", "", "")
			cmd.Flags().String("exchange", "", "")
			cmd.Flags().String("index", "", "")
			cmd.Flags().String("sector", "", "")
			cmd.Flags().String("market-segment", "", "")
			cmd.Flags().String("event-type", "", "")
			cmd.Flags().String("status", "", "")
			cmd.Flags().String("scheduled-after", "", "")
			cmd.Flags().String("scheduled-before", "", "")
			cmd.Flags().String("updated-after", "", "")
			cmd.Flags().String("watchlist-id", "", "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().String("cursor", "", "")
			cmd.Flags().String("fields", "", "")

			for k, v := range tt.flags {
				_ = cmd.Flags().Set(k, v)
			}
			for k, v := range tt.bools {
				if v {
					_ = cmd.Flags().Set(k, "true")
				}
			}
			for k, v := range tt.ints {
				_ = cmd.Flags().Set(k, fmt.Sprintf("%d", v))
			}

			params := buildCalendarParams(&cmd)

			for wantKey, wantVal := range tt.wantKV {
				if got := params.Get(wantKey); got != wantVal {
					t.Errorf("params[%q] = %q, want %q", wantKey, got, wantVal)
				}
			}

			for key := range params {
				if _, ok := tt.wantKV[key]; !ok {
					t.Errorf("unexpected param %q = %q", key, params.Get(key))
				}
			}
		})
	}
}

func TestCalendarCompanyCell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.CalendarEvent
		want  string
	}{
		{
			name:  "a listed issuer is named by ticker",
			event: api.CalendarEvent{CompanyID: "uejazctgchj4", Company: &api.CalendarCompany{Name: "AB Volvo", Ticker: "VOLV-B"}},
			want:  "VOLV-B",
		},
		{
			name:  "an unlisted issuer is named, cut to fit",
			event: api.CalendarEvent{CompanyID: "cmpny1234567", Company: &api.CalendarCompany{Name: "Telefonaktiebolaget LM Ericsson"}},
			want:  "Telefonaktiebolaget…",
		},
		{
			name:  "falls back to company_id when no inline company",
			event: api.CalendarEvent{CompanyID: "cmpny1234567"},
			want:  "cmpny1234567",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := calendarCompanyCell(tt.event); got != tt.want {
				t.Errorf("calendarCompanyCell() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalendarListWhen(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		event api.CalendarEvent
		want  string
	}{
		{
			name:  "a date with no announced time shows the day alone",
			event: api.CalendarEvent{DatePrecision: "day", LocalTime: "2026-10-23T00:00:00.000+02:00", ScheduledAt: "2026-10-22T22:00:00.000Z"},
			want:  "Fri 23 Oct",
		},
		{
			name:  "an announced time is shown on the issuer's clock",
			event: api.CalendarEvent{DatePrecision: "exact", LocalTime: "2026-10-05T10:00:00.000+02:00"},
			want:  "Mon 5 Oct 10:00",
		},
		{
			name:  "another year names it",
			event: api.CalendarEvent{DatePrecision: "day", LocalTime: "2027-02-04T00:00:00.000+01:00"},
			want:  "4 Feb 2027",
		},
		{
			name:  "unparseable is shown as received",
			event: api.CalendarEvent{LocalTime: "2026"},
			want:  "2026",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := calendarListWhen(tt.event, now); got != tt.want {
				t.Errorf("calendarListWhen() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalendarDetailWhen(t *testing.T) {
	t.Parallel()

	day := &api.CalendarEvent{DatePrecision: "day", LocalTime: "2026-10-23T00:00:00.000+02:00", Timezone: "Europe/Stockholm"}
	if got, want := calendarDetailWhen(day), "Fri 23 Oct 2026, time not announced"; got != want {
		t.Errorf("day precision = %q, want %q", got, want)
	}
	exact := &api.CalendarEvent{DatePrecision: "exact", LocalTime: "2026-10-05T10:00:00.000+02:00", Timezone: "Europe/Oslo"}
	if got, want := calendarDetailWhen(exact), "Mon 5 Oct 2026, 10:00 Oslo time"; got != want {
		t.Errorf("exact = %q, want %q", got, want)
	}
}

func TestCalendarList_Table(t *testing.T) {
	body := `{"calendar_events":[` +
		`{"id":"ly89wavugvsl","title":"Q3 2026 Interim Report","event_type":"earnings_report","status":"scheduled",` +
		`"date_precision":"day","local_time":"2026-10-23T00:00:00.000+02:00","company_id":"uejazctgchj4",` +
		`"company":{"id":"uejazctgchj4","name":"AB Volvo","ticker":"VOLV-B"}}],"pagination":{"count":1}}`

	t.Run("type and ticker shown, an all-scheduled page has no Status column", func(t *testing.T) {
		out := runListCommand(t, calendarListCmd, fixedBody(body), "table", "--ticker", "VOLV-B")
		for _, want := range []string{"Earnings", "VOLV-B", "1 calendar event (from today)."} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q\n%s", want, out)
			}
		}
		if strings.Contains(out, "STATUS") || strings.Contains(out, "earnings_report") {
			t.Errorf("output has a Status column or a raw slug\n%s", out)
		}
	})

	t.Run("a cancelled row brings in the Status column", func(t *testing.T) {
		cancelled := strings.Replace(body, `"status":"scheduled"`, `"status":"cancelled"`, 1)
		out := runListCommand(t, calendarListCmd, fixedBody(cancelled), "table", "--ticker", "VOLV-B")
		if !strings.Contains(out, "STATUS") || !strings.Contains(out, "cancelled") {
			t.Errorf("output = %q, want the cancellation shown", out)
		}
	})

	t.Run("an empty upcoming page says past dates need a window", func(t *testing.T) {
		empty := `{"calendar_events":[],"pagination":{"count":0}}`
		out := runListCommand(t, calendarListCmd, fixedBody(empty), "table", "--ticker", "VOLV-B")
		if !strings.Contains(out, "Only upcoming dates are listed; add --scheduled-after") {
			t.Errorf("output = %q, want the upcoming-only default explained", out)
		}
	})

	// Either end of a window, or a sync, lifts the API's upcoming-only default.
	for _, flag := range []string{"--scheduled-after", "--scheduled-before", "--updated-after"} {
		t.Run(flag+" drops the from-today note", func(t *testing.T) {
			out := runListCommand(t, calendarListCmd, fixedBody(body), "table", flag, "2026-01-01")
			if strings.Contains(out, "from today") {
				t.Errorf("output = %q, want no from-today note", out)
			}
		})
	}

	t.Run("an announced time names whose clock it is on", func(t *testing.T) {
		exact := strings.Replace(body, `"date_precision":"day"`, `"date_precision":"exact"`, 1)
		out := runListCommand(t, calendarListCmd, fixedBody(exact), "table", "--scheduled-after", "2026-01-01")
		if !strings.Contains(out, "(times are each issuer's local time)") {
			t.Errorf("output = %q, want the clock named", out)
		}
	})

	t.Run("a capped plan is not told to widen the window", func(t *testing.T) {
		empty := `{"calendar_events":[],"pagination":{"count":0},"plan_limited":true}`
		out := runListCommand(t, calendarListCmd, fixedBody(empty), "table", "--ticker", "VOLV-B")
		if strings.Contains(out, "--scheduled-after") {
			t.Errorf("output = %q, want no advice the plan cannot follow", out)
		}
	})

	// The When column reads local_time and date_precision, so a narrowed table
	// fetches them whichever of its fields was named.
	t.Run("a narrowed table fetches what the When column needs", func(t *testing.T) {
		var got string
		handler := func(w http.ResponseWriter, r *http.Request) {
			got = r.URL.Query().Get("fields")
			fixedBody(body)(w, r)
		}
		runListCommand(t, calendarListCmd, handler, "table", "--fields", "scheduled_at,title")
		if !strings.Contains(got, "local_time") || !strings.Contains(got, "date_precision") {
			t.Errorf("requested fields = %q, want local_time and date_precision added", got)
		}
	})
}

func TestCalendarGet_Detail(t *testing.T) {
	body := `{"calendar_event":{"id":"4q3a0q706wcm","title":"Cash Dividend","event_type":"dividend","status":"scheduled",` +
		`"date_precision":"day","local_time":"2026-10-05T00:00:00.000+02:00","timezone":"Europe/Stockholm",` +
		`"fiscal_period":"fy_2025","source_article_id":"art1","company_id":"co1",` +
		`"company":{"id":"co1","name":"Thule Group AB","ticker":"THULE"},` +
		`"dividend":{"record_date":"2026-10-05","payment_date":"2026-10-08","kind":"ordinary"},` +
		`"updated_at":"2026-09-25T14:17:49.048Z"}}`

	out := runCommandWithArgs(t, calendarGetCmd, fixedBody(body), "table", []string{"4q3a0q706wcm"})

	for _, want := range []string{
		"Dividend", "FY 2025", "Mon 5 Oct 2026, time not announced", "THULE  Thule Group AB",
		"Thu 8 Oct 2026", "nfn companies get THULE", "nfn articles get art1", "nfn calendar list --ticker THULE",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"Ex-Dividend", "Declared", "2026-09-25T14:17", "T00:00:00"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output has %q\n%s", unwanted, out)
		}
	}
}
