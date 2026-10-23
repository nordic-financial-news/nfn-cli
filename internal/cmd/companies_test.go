package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
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
			name:   "market-segment maps to market_segment",
			flags:  map[string]string{"market-segment": "large_cap,mid_cap"},
			wantKV: map[string]string{"market_segment": "large_cap,mid_cap"},
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
			cmd.Flags().String("market-segment", "", "")
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
	rejected := []string{"ticker", "company", "watchlist-id", "listed", "exchange", "market", "domicile", "index", "sector", "market-segment"}
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
			cmd.Flags().String("market-segment", "", "")
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

func TestCompaniesList_Table(t *testing.T) {
	active := `{"companies":[` +
		`{"id":"co1","name":"AB Volvo","ticker":"VOLV-B","is_active":true,"exchange":{"mic_code":"XSTO"}},` +
		`{"id":"co2","name":"Volvo Penta","ticker":"","is_active":true}],` +
		`"pagination":{"count":2,"next_cursor":"cur9"}}`

	t.Run("listing column, footer and next cursor", func(t *testing.T) {
		out := runListCommand(t, companiesListCmd, fixedBody(active), "table", "--q", "Volvo")
		for _, want := range []string{"VOLV-B · XSTO", "unlisted", "2 companies. Details: nfn companies get <id>", "--cursor cur9"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q\n%s", want, out)
			}
		}
		if strings.Contains(out, "STATUS") {
			t.Errorf("output has a Status column on an all-active page\n%s", out)
		}
	})

	t.Run("an inactive company brings in the Status column", func(t *testing.T) {
		body := `{"companies":[{"id":"co3","name":"Anttila","is_active":false}],"pagination":{"count":1}}`
		out := runListCommand(t, companiesListCmd, fixedBody(body), "table", "--q", "Anttila")
		if !strings.Contains(out, "STATUS") || !strings.Contains(out, "inactive") {
			t.Errorf("output = %q, want the company marked inactive", out)
		}
	})

	// --fields cannot request is_active, so a narrowed row reads as inactive;
	// the Status column must not take that at face value.
	t.Run("a --fields selection does not mark every row inactive", func(t *testing.T) {
		body := `{"companies":[{"id":"co1","name":"AB Volvo","ticker":"VOLV-B"}],"pagination":{"count":1}}`
		out := runListCommand(t, companiesListCmd, fixedBody(body), "table", "--q", "Volvo", "--fields", "name,ticker")
		if strings.Contains(out, "inactive") {
			t.Errorf("output = %q, want no Status column under --fields", out)
		}
	})

	t.Run("an unfiltered list says it is alphabetical", func(t *testing.T) {
		out := runListCommand(t, companiesListCmd, fixedBody(active), "table")
		if !strings.Contains(out, "alphabetical; narrow with --q") {
			t.Errorf("output = %q, want the ordering named", out)
		}
	})

	t.Run("no matches says so instead of drawing a header", func(t *testing.T) {
		out := runListCommand(t, companiesListCmd, fixedBody(`{"companies":[],"pagination":{"count":0}}`), "table", "--q", "zzz")
		if strings.Contains(out, "NAME") || !strings.Contains(out, "No companies match.") {
			t.Errorf("output = %q, want only the empty-state line", out)
		}
	})
}

func TestUnfilteredCompanyList(t *testing.T) {
	t.Parallel()
	if !unfilteredCompanyList(url.Values{"limit": {"5"}, "cursor": {"abc"}}) {
		t.Error("paging params alone should count as unfiltered")
	}
	if unfilteredCompanyList(url.Values{"listed": {"true"}}) {
		t.Error("--listed should count as a filter")
	}
}

func TestCompaniesGet_Detail(t *testing.T) {
	body := `{"company":{"id":"co1","name":"AB Volvo","slug":"ab-volvo","ticker":"VOLV-B","is_active":true,` +
		`"exchange":{"name":"Nasdaq Stockholm","mic_code":"XSTO"},` +
		`"market_segment":"large_cap",` +
		`"stock_listings":[{"ticker":"VOLV-B","primary":true,"exchange":{"name":"Nasdaq Stockholm","mic_code":"XSTO"},"market_segment":"large_cap"},` +
		`{"ticker":"VOLV-A","primary":false,"exchange":{"name":"Nasdaq Stockholm","mic_code":"XSTO"},"market_segment":"large_cap"}],` +
		`"also_listed_on":[{"ticker":"VOLV-A","mic_code":"XSTO"}],` +
		`"subsidiaries":[{"id":"sub1","name":"Volvo Penta"},{"id":"sub2","name":"Volvo Cars","ticker":"VOLCAR-B"}],` +
		`"article_count":637,"story_count":24}}`

	out := runCommandWithArgs(t, companiesGetCmd, fixedBody(body), "table", []string{"VOLV-B"})

	if strings.Contains(out, "ab-volvo") {
		t.Errorf("output = %q, want the slug left to JSON", out)
	}
	if n := strings.Count(out, "VOLV-A"); n != 1 {
		t.Errorf("VOLV-A appears %d times, want the two listing fields merged into one\n%s", n, out)
	}
	for _, want := range []string{
		"VOLV-B · Nasdaq Stockholm (XSTO) · Large Cap · primary",
		"VOLV-A · Nasdaq Stockholm (XSTO) · Large Cap",
		"sub1      Volvo Penta", "VOLCAR-B  Volvo Cars",
		"nfn companies events VOLV-B",
		"637 articles:", "nfn companies articles VOLV-B",
		"nfn calendar list --ticker VOLV-B",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestCompaniesGet_JSONKeepsMarketSegment(t *testing.T) {
	body := `{"company":{"id":"co1","name":"AB Volvo","ticker":"VOLV-B","is_active":true,"market_segment":"large_cap",` +
		`"stock_listings":[{"ticker":"VOLV-B","primary":true,"exchange":{"mic_code":"XSTO"},"market_segment":"large_cap"},` +
		`{"ticker":"VOLV-US","primary":false,"exchange":{"mic_code":"OTCM"},"market_segment":null}]}}`

	out := runCommandWithArgs(t, companiesGetCmd, fixedBody(body), "json", []string{"VOLV-B"})

	var envelope struct {
		Data struct {
			MarketSegment string                   `json:"market_segment"`
			StockListings []map[string]interface{} `json:"stock_listings"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decoding output: %v\n%s", err, out)
	}
	if envelope.Data.MarketSegment != "large_cap" {
		t.Errorf("market_segment = %q, want large_cap\n%s", envelope.Data.MarketSegment, out)
	}
	if len(envelope.Data.StockListings) != 2 {
		t.Fatalf("len(stock_listings) = %d, want 2\n%s", len(envelope.Data.StockListings), out)
	}
	if got := envelope.Data.StockListings[0]["market_segment"]; got != "large_cap" {
		t.Errorf("stock_listings[0].market_segment = %v, want large_cap", got)
	}
	if _, ok := envelope.Data.StockListings[1]["market_segment"]; ok {
		t.Errorf("stock_listings[1] carries market_segment, want it omitted for a listing with none")
	}
}

// The CLI does not validate --market-segment itself: the API's 400 already
// names the valid values, so its detail has to reach the user unchanged.
func TestCompaniesList_MarketSegmentRejectionPassesThrough(t *testing.T) {
	const detail = "Unknown market segment(s): Large_Cap. Valid values: large_cap, mid_cap, small_cap, first_north_premier, first_north_growth"
	var gotParams url.Values
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"type": "invalid-parameter", "title": "Invalid parameter", "detail": detail})
	}

	err := runListCommandExpectingError(t, companiesListCmd, handler, "--market-segment", "Large_Cap")

	if got := gotParams.Get("market_segment"); got != "Large_Cap" {
		t.Errorf("market_segment = %q, want the value forwarded as typed", got)
	}
	if err.Error() != detail {
		t.Errorf("error = %q, want the API's detail %q", err.Error(), detail)
	}
}

func TestCompanyListings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		company api.Company
		want    []string
	}{
		{name: "unlisted", company: api.Company{ID: "co1"}, want: []string{"Not listed"}},
		{
			name:    "a single listing is not marked primary",
			company: api.Company{StockListings: []api.StockListing{{TickerSymbol: "SAAB-B", Primary: true, Exchange: api.Exchange{MicCode: "XSTO"}}}},
			want:    []string{"SAAB-B · XSTO"},
		},
		{
			name:    "summary fields when the detail listings are absent",
			company: api.Company{TickerSymbol: "VOLV-B", Exchange: &api.Exchange{MicCode: "XSTO"}, AlsoListedOn: []api.OtherListing{{TickerSymbol: "VOLV-A", MicCode: "XSTO"}}},
			want:    []string{"VOLV-B · XSTO", "VOLV-A · XSTO"},
		},
		{
			name: "each listing names its own segment, and none is shown where it is empty",
			company: api.Company{StockListings: []api.StockListing{
				{TickerSymbol: "NDA-FI", Primary: true, Exchange: api.Exchange{MicCode: "XHEL"}, MarketSegment: "large_cap"},
				{TickerSymbol: "NDA-SE", Exchange: api.Exchange{MicCode: "XSTO"}, MarketSegment: "first_north_growth"},
				{TickerSymbol: "NDA-US", Exchange: api.Exchange{MicCode: "OTCM"}},
			}},
			want: []string{"NDA-FI · XHEL · Large Cap · primary", "NDA-SE · XSTO · First North Growth", "NDA-US · OTCM"},
		},
		{
			name:    "the summary's segment follows its primary ticker",
			company: api.Company{TickerSymbol: "ELISA", Exchange: &api.Exchange{MicCode: "XHEL"}, MarketSegment: "mid_cap"},
			want:    []string{"ELISA · XHEL · Mid Cap"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := companyListings(&tt.company); strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("companyListings() = %q, want %q", got, tt.want)
			}
		})
	}
}

// An unlisted company has no ticker, so its next steps fall back to the ID and
// leave out the calendar, which is keyed by listing.
func TestCompanyNextStepsForAnUnlistedCompany(t *testing.T) {
	t.Parallel()
	got := strings.Join(companyNextSteps(&api.Company{ID: "co2", ArticleCount: 1}), "\n")
	if !strings.Contains(got, "nfn companies events co2") || !strings.Contains(got, "1 article:") {
		t.Errorf("steps = %q, want events and articles by ID", got)
	}
	if strings.Contains(got, "calendar") || strings.Contains(got, "stories") {
		t.Errorf("steps = %q, want no calendar or empty stories step", got)
	}
}
