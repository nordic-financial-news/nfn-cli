package cmd

import (
	"fmt"
	"testing"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{name: "under limit", s: "hello", max: 10, want: "hello"},
		{name: "exact limit", s: "hello", max: 5, want: "hello"},
		{name: "over limit", s: "hello world", max: 8, want: "hello..."},
		{name: "empty string", s: "", max: 5, want: ""},
		{name: "just at boundary", s: "abcdef", max: 6, want: "abcdef"},
		{name: "one over", s: "abcdefg", max: 6, want: "abc..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

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
			cmd.Flags().String("category", "", "")
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("company", "", "")
			cmd.Flags().String("exchange", "", "")
			cmd.Flags().String("index", "", "")
			cmd.Flags().String("sector", "", "")
			cmd.Flags().String("content-type", "", "")
			cmd.Flags().String("published-after", "", "")
			cmd.Flags().String("published-before", "", "")
			cmd.Flags().String("updated-after", "", "")
			cmd.Flags().String("q", "", "")
			cmd.Flags().String("ids", "", "")
			cmd.Flags().String("sources", "", "")
			cmd.Flags().String("source-type", "", "")
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
