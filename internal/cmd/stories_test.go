package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildStoryParams(t *testing.T) {
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
			name:   "country flag",
			flags:  map[string]string{"country": "SE"},
			wantKV: map[string]string{"country": "SE"},
		},
		{
			name:   "sources flag",
			flags:  map[string]string{"sources": "id1,id2"},
			wantKV: map[string]string{"sources": "id1,id2"},
		},
		{
			name:   "listed bool maps to string true",
			bools:  map[string]bool{"listed": true},
			wantKV: map[string]string{"listed": "true"},
		},
		{
			name:   "watchlist-id maps to watchlist",
			flags:  map[string]string{"watchlist-id": "a1b2c3d4e5f6"},
			wantKV: map[string]string{"watchlist": "a1b2c3d4e5f6"},
		},
		{
			name:   "country accepts comma-separated list",
			flags:  map[string]string{"country": "SE,NO"},
			wantKV: map[string]string{"country": "SE,NO"},
		},
		{
			name:   "company flag",
			flags:  map[string]string{"company": "cmpny1234567"},
			wantKV: map[string]string{"company": "cmpny1234567"},
		},
		{
			name:   "exchange, index, sector filters",
			flags:  map[string]string{"exchange": "XSTO", "index": "OMXS30", "sector": "Industrials"},
			wantKV: map[string]string{"exchange": "XSTO", "index": "OMXS30", "sector": "Industrials"},
		},
		{
			name:   "limit int",
			ints:   map[string]int{"limit": 25},
			wantKV: map[string]string{"limit": "25"},
		},
		{
			name:   "q flag",
			flags:  map[string]string{"q": "battery"},
			wantKV: map[string]string{"q": "battery"},
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
			name:   "q and sort flags",
			flags:  map[string]string{"q": "battery", "sort": "latest"},
			wantKV: map[string]string{"q": "battery", "sort": "latest"},
		},
		{
			name:   "published-after and published-before map to underscores",
			flags:  map[string]string{"published-after": "2025-01-01", "published-before": "2025-06-01"},
			wantKV: map[string]string{"published_after": "2025-01-01", "published_before": "2025-06-01"},
		},
		{
			name:   "ticker accepts comma-separated list",
			flags:  map[string]string{"ticker": "VOLV-B,ERIC-B"},
			wantKV: map[string]string{"ticker": "VOLV-B,ERIC-B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := *storiesListCmd
			cmd.ResetFlags()
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("market", "", "")
			cmd.Flags().String("domicile", "", "")
			cmd.Flags().String("category", "", "")
			cmd.Flags().String("q", "", "")
			cmd.Flags().String("sort", "", "")
			cmd.Flags().String("published-after", "", "")
			cmd.Flags().String("published-before", "", "")
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("company", "", "")
			cmd.Flags().String("exchange", "", "")
			cmd.Flags().String("index", "", "")
			cmd.Flags().String("sector", "", "")
			cmd.Flags().String("sources", "", "")
			cmd.Flags().Bool("semantic", false, "")
			cmd.Flags().Bool("listed", false, "")
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

			params := buildStoryParams(&cmd)

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

func TestStoriesGet_Detail(t *testing.T) {
	body := `{"story":{"id":"w668rgglpr4w","title":"Merus Power board approves 2027–2030 strategy","summary":"The board approved new targets.",` +
		`"content":"The plan calls for revenue of EUR 160 million.\n\n## Financial targets\n- EBITDA margin of at least 12%",` +
		`"published_at":"2026-10-04T07:07:20.000Z","article_count":2,"article_ids":["iwfcowgf08vz","ysytnjcpqn8p"],"country":"FI",` +
		`"companies":[{"id":"fj0j795s2ru4","name":"Merus Power Oyj","ticker":"MERUS"}],` +
		`"category":{"id":"2m0aj78adld4","name":"Product & Strategy"},"event_id":"d33h6ob7dbjg"}}`

	out := runCommandWithArgs(t, storiesGetCmd, fixedBody(body), "table", []string{"w668rgglpr4w"})

	for _, want := range []string{
		"MERUS  Merus Power Oyj", "| iwfcowgf08vz", "| ysytnjcpqn8p",
		"FINANCIAL TARGETS\n", "• EBITDA margin of at least 12%",
		"nfn events get d33h6ob7dbjg", "nfn articles get iwfcowgf08vz", "nfn companies get fj0j795s2ru4",
		"Articles (2)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	// The body is prose under the table, not a cell, and keeps no markdown.
	for _, unwanted := range []string{"## ", "- EBITDA", "2026-10-04T07:07", "| The plan calls"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output has %q\n%s", unwanted, out)
		}
	}
}
