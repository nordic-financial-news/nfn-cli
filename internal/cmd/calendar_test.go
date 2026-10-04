package cmd

import (
	"fmt"
	"testing"

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

func TestEventCompany(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.CalendarEvent
		want  string
	}{
		{
			name:  "prefers inline company name",
			event: api.CalendarEvent{CompanyID: "cmpny1234567", Company: &api.CalendarCompany{ID: "cmpny1234567", Name: "Telefonaktiebolaget LM Ericsson", Ticker: "ERIC-B"}},
			want:  "Telefonaktiebolaget LM Ericss…",
		},
		{
			name:  "falls back to company_id when no inline company",
			event: api.CalendarEvent{CompanyID: "cmpny1234567"},
			want:  "cmpny1234567",
		},
		{
			name:  "falls back to company_id when inline company has no name",
			event: api.CalendarEvent{CompanyID: "cmpny1234567", Company: &api.CalendarCompany{ID: "cmpny1234567"}},
			want:  "cmpny1234567",
		},
		{
			name:  "short name returned in full",
			event: api.CalendarEvent{CompanyID: "cmpny1234567", Company: &api.CalendarCompany{Name: "Volvo"}},
			want:  "Volvo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eventCompany(tt.event); got != tt.want {
				t.Errorf("eventCompany() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event api.CalendarEvent
		want  string
	}{
		{
			name:  "prefers local_time date",
			event: api.CalendarEvent{LocalTime: "2026-10-22T07:00:00.000+02:00", ScheduledAt: "2026-10-22T05:00:00.000Z"},
			want:  "2026-10-22",
		},
		{
			name:  "falls back to scheduled_at when local_time empty",
			event: api.CalendarEvent{ScheduledAt: "2026-04-24T22:00:00.000Z"},
			want:  "2026-04-24",
		},
		{
			name:  "short string returned as-is without panic",
			event: api.CalendarEvent{LocalTime: "2026"},
			want:  "2026",
		},
		{
			name:  "empty event yields empty string",
			event: api.CalendarEvent{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := eventDate(tt.event); got != tt.want {
				t.Errorf("eventDate() = %q, want %q", got, tt.want)
			}
		})
	}
}
