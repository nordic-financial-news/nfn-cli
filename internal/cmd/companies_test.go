package cmd

import (
	"fmt"
	"testing"
)

func TestBuildCompanyParams(t *testing.T) {
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
			name:   "q flag",
			flags:  map[string]string{"q": "Volvo"},
			wantKV: map[string]string{"q": "Volvo"},
		},
		{
			name:   "ticker batch lookup accepts comma-separated list",
			flags:  map[string]string{"ticker": "VOLV-B,ERIC-B"},
			wantKV: map[string]string{"ticker": "VOLV-B,ERIC-B"},
		},
		{
			name:   "company batch lookup accepts comma-separated ids",
			flags:  map[string]string{"company": "64hn2d7ul6kr,7f3k9m2p1q8r"},
			wantKV: map[string]string{"company": "64hn2d7ul6kr,7f3k9m2p1q8r"},
		},
		{
			name:   "ticker and company intersect when both sent",
			flags:  map[string]string{"ticker": "VOLV-B", "company": "64hn2d7ul6kr"},
			wantKV: map[string]string{"ticker": "VOLV-B", "company": "64hn2d7ul6kr"},
		},
		{
			name:   "country flag",
			flags:  map[string]string{"country": "SE,NO"},
			wantKV: map[string]string{"country": "SE,NO"},
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
			// domicile is the canonical name for country here, but the API
			// intersects them like any other pair of filters, so the CLI
			// forwards both rather than collapsing one into the other.
			name:   "country and domicile are both forwarded",
			flags:  map[string]string{"country": "SE", "domicile": "FI"},
			wantKV: map[string]string{"country": "SE", "domicile": "FI"},
		},
		{
			name:   "exchange accepts comma-separated MIC codes",
			flags:  map[string]string{"exchange": "XSTO,XCSE"},
			wantKV: map[string]string{"exchange": "XSTO,XCSE"},
		},
		{
			name:   "sector flag",
			flags:  map[string]string{"sector": "Financials"},
			wantKV: map[string]string{"sector": "Financials"},
		},
		{
			name:   "listed bool maps to string true",
			bools:  map[string]bool{"listed": true},
			wantKV: map[string]string{"listed": "true"},
		},
		{
			name:   "watchlist bool maps to string true",
			bools:  map[string]bool{"watchlist": true},
			wantKV: map[string]string{"watchlist": "true"},
		},
		{
			name:   "limit int",
			ints:   map[string]int{"limit": 25},
			wantKV: map[string]string{"limit": "25"},
		},
		{
			name:   "market combined with sector",
			flags:  map[string]string{"market": "SE,NO", "sector": "Industrials"},
			wantKV: map[string]string{"market": "SE,NO", "sector": "Industrials"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := *companiesListCmd
			cmd.ResetFlags()
			cmd.Flags().String("q", "", "")
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("company", "", "")
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("market", "", "")
			cmd.Flags().String("domicile", "", "")
			cmd.Flags().String("exchange", "", "")
			cmd.Flags().String("sector", "", "")
			cmd.Flags().Bool("listed", false, "")
			cmd.Flags().Bool("is-active", false, "")
			cmd.Flags().Bool("watchlist", false, "")
			cmd.Flags().Bool("semantic", false, "")
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

			params := buildCompanyParams(&cmd)

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

// TestCompanyNewsRoutesRegisterFilterFlags pins the item 9 filter-parity contract
// for the nested company article/story routes: they gain the flat-feed content
// filters, but must not expose params the /companies/{id}/... API now rejects.
func TestCompanyNewsRoutesRegisterFilterFlags(t *testing.T) {
	t.Parallel()

	articleFilters := []string{
		"q", "sort", "country", "category", "sources", "source-type",
		"content-type", "published-after", "published-before", "updated-after",
		"fields", "primary-only",
	}
	for _, name := range articleFilters {
		if companiesArticlesCmd.Flags().Lookup(name) == nil {
			t.Errorf("companies articles missing --%s flag", name)
		}
	}

	storyFilters := []string{
		"q", "sort", "country", "category", "sources",
		"published-after", "published-before", "updated-after", "fields",
	}
	for _, name := range storyFilters {
		if companiesStoriesCmd.Flags().Lookup(name) == nil {
			t.Errorf("companies stories missing --%s flag", name)
		}
	}

	// The /stories API does not accept source-type/content-type; the route must not offer them.
	for _, name := range []string{"source-type", "content-type"} {
		if companiesStoriesCmd.Flags().Lookup(name) != nil {
			t.Errorf("companies stories unexpectedly registers --%s (the API rejects it)", name)
		}
	}

	// The nested routes describe one company via the path, so the company-selecting
	// filters the API now 400s on must not be exposed on either route.
	rejected := []string{"ticker", "company", "watchlist-id", "listed", "exchange", "market", "domicile", "index", "sector"}
	for _, name := range rejected {
		if companiesArticlesCmd.Flags().Lookup(name) != nil {
			t.Errorf("companies articles unexpectedly registers rejected filter --%s", name)
		}
		if companiesStoriesCmd.Flags().Lookup(name) != nil {
			t.Errorf("companies stories unexpectedly registers rejected filter --%s", name)
		}
	}
}

// is-active is the one companies filter that forwards an explicit false, so it
// is exercised separately from the table above.
func TestBuildCompanyParamsIsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		set     bool
		value   string
		want    string
		wantSet bool
	}{
		{name: "unset omits is_active", set: false},
		{name: "true forwards true", set: true, value: "true", want: "true", wantSet: true},
		{name: "false forwards false", set: true, value: "false", want: "false", wantSet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := *companiesListCmd
			cmd.ResetFlags()
			cmd.Flags().String("q", "", "")
			cmd.Flags().String("ticker", "", "")
			cmd.Flags().String("company", "", "")
			cmd.Flags().String("country", "", "")
			cmd.Flags().String("market", "", "")
			cmd.Flags().String("domicile", "", "")
			cmd.Flags().String("exchange", "", "")
			cmd.Flags().String("sector", "", "")
			cmd.Flags().Bool("listed", false, "")
			cmd.Flags().Bool("is-active", false, "")
			cmd.Flags().Bool("watchlist", false, "")
			cmd.Flags().Bool("semantic", false, "")
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().String("cursor", "", "")
			cmd.Flags().String("fields", "", "")

			if tt.set {
				_ = cmd.Flags().Set("is-active", tt.value)
			}

			params := buildCompanyParams(&cmd)
			got, ok := params["is_active"]
			if ok != tt.wantSet {
				t.Fatalf("is_active present = %v, want %v", ok, tt.wantSet)
			}
			if tt.wantSet && got[0] != tt.want {
				t.Errorf("is_active = %q, want %q", got[0], tt.want)
			}
		})
	}
}
