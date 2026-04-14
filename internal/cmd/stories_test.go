package cmd

import (
	"fmt"
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
			name:   "limit int",
			ints:   map[string]int{"limit": 25},
			wantKV: map[string]string{"limit": "25"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := *storiesListCmd
			cmd.ResetFlags()
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("category", "", "")
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("sources", "", "")
			cmd.Flags().Bool("listed", false, "")
			cmd.Flags().Bool("watchlist", false, "")
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
