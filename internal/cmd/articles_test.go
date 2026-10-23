package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildArticleParams(t *testing.T) {
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
			name:   "content-type maps to content_type",
			flags:  map[string]string{"content-type": "press_release"},
			wantKV: map[string]string{"content_type": "press_release"},
		},
		{
			name:   "published-after maps to published_after",
			flags:  map[string]string{"published-after": "2025-01-01"},
			wantKV: map[string]string{"published_after": "2025-01-01"},
		},
		{
			name:   "listed bool maps to string true",
			bools:  map[string]bool{"listed": true},
			wantKV: map[string]string{"listed": "true"},
		},
		{
			name:   "limit int",
			ints:   map[string]int{"limit": 25},
			wantKV: map[string]string{"limit": "25"},
		},
		{
			name:   "sources flag",
			flags:  map[string]string{"sources": "id1,id2"},
			wantKV: map[string]string{"sources": "id1,id2"},
		},
		{
			name:   "source-type maps to source_type",
			flags:  map[string]string{"source-type": "stock_exchange,wire_service"},
			wantKV: map[string]string{"source_type": "stock_exchange,wire_service"},
		},
		{
			name:   "watchlist-id maps to watchlist",
			flags:  map[string]string{"watchlist-id": "a1b2c3d4e5f6"},
			wantKV: map[string]string{"watchlist": "a1b2c3d4e5f6"},
		},
		{
			name:   "sort flag",
			flags:  map[string]string{"sort": "latest"},
			wantKV: map[string]string{"sort": "latest"},
		},
		{
			name:   "primary-only bool maps to primary_only true",
			bools:  map[string]bool{"primary-only": true},
			wantKV: map[string]string{"primary_only": "true"},
		},
		{
			name:   "ticker accepts comma-separated list",
			flags:  map[string]string{"ticker": "VOLV-B,ERIC-B"},
			wantKV: map[string]string{"ticker": "VOLV-B,ERIC-B"},
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
			name: "multiple flags combined",
			flags: map[string]string{
				"country":  "NO",
				"category": "Earnings",
				"ticker":   "VOLV-B",
				"sources":  "src1",
			},
			wantKV: map[string]string{
				"country":  "NO",
				"category": "Earnings",
				"ticker":   "VOLV-B",
				"sources":  "src1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a fresh command with the same flags as articlesListCmd
			cmd := *articlesListCmd
			cmd.ResetFlags()
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("market", "", "")
			cmd.Flags().String("domicile", "", "")
			cmd.Flags().String("category", "", "")
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("company", "", "")
			cmd.Flags().String("exchange", "", "")
			cmd.Flags().String("index", "", "")
			cmd.Flags().String("sector", "", "")
			cmd.Flags().String("market-segment", "", "")
			cmd.Flags().String("content-type", "", "")
			cmd.Flags().String("published-after", "", "")
			cmd.Flags().String("published-before", "", "")
			cmd.Flags().String("updated-after", "", "")
			cmd.Flags().String("q", "", "")
			cmd.Flags().String("sort", "", "")
			cmd.Flags().String("ids", "", "")
			cmd.Flags().String("sources", "", "")
			cmd.Flags().String("source-type", "", "")
			cmd.Flags().Bool("semantic", false, "")
			cmd.Flags().Bool("listed", false, "")
			cmd.Flags().String("watchlist-id", "", "")
			cmd.Flags().Bool("primary-only", false, "")
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

			params := buildArticleParams(&cmd)

			for wantKey, wantVal := range tt.wantKV {
				if got := params.Get(wantKey); got != wantVal {
					t.Errorf("params[%q] = %q, want %q", wantKey, got, wantVal)
				}
			}

			// Verify no extra params set
			for key := range params {
				if _, ok := tt.wantKV[key]; !ok {
					t.Errorf("unexpected param %q = %q", key, params.Get(key))
				}
			}
		})
	}
}

func TestArticlesGet_Detail(t *testing.T) {
	body := `{"article":{"id":"art1","title":"Volvo orders fell","source_title":"Orderingången sjönk",` +
		`"summary":"Orders fell.","article_url":"https://efn.se/a-very-long-path-that-must-stay-whole-1234567890",` +
		`"content_type":"press_release","published_at":"2026-10-02T20:09:53.000Z","country":"SE",` +
		`"source":{"name":"EFN","source_type":"news_publication"},` +
		`"companies":[{"id":"co1","name":"AB Volvo","ticker":"VOLV-B"}],` +
		`"key_points":["Orders fell 9.5%."],"event_id":"evt1"}}`

	out := runCommandWithArgs(t, articlesGetCmd, fixedBody(body), "table", []string{"art1"})

	if strings.Contains(out, "2026-10-02T20:09:53") {
		t.Errorf("output = %q, want the time rendered for reading", out)
	}
	for _, want := range []string{
		"• Orders fell 9.5%.", "VOLV-B  AB Volvo", "EFN (news publication)", "press release",
		"Original Title", "Read the original:", "https://efn.se/a-very-long-path-that-must-stay-whole-1234567890",
		"nfn events get evt1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}
