package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCompany_AlsoListedOn(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/companies/NDA-FI" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/companies/NDA-FI")
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"company": map[string]interface{}{
				"id":     "cmp1",
				"name":   "Nordea Bank",
				"ticker": "NDA-FI",
				"also_listed_on": []map[string]string{
					{"ticker": "NDA-SE", "mic_code": "XSTO"},
					{"ticker": "NDA-DK", "mic_code": "XCSE"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient("key", WithBaseURL(srv.URL))
	company, _, err := c.GetCompany(context.Background(), "NDA-FI")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(company.AlsoListedOn) != 2 {
		t.Fatalf("got %d also_listed_on entries, want 2", len(company.AlsoListedOn))
	}
	if company.AlsoListedOn[0].TickerSymbol != "NDA-SE" {
		t.Errorf("AlsoListedOn[0].TickerSymbol = %q, want %q", company.AlsoListedOn[0].TickerSymbol, "NDA-SE")
	}
	if company.AlsoListedOn[0].MicCode != "XSTO" {
		t.Errorf("AlsoListedOn[0].MicCode = %q, want %q", company.AlsoListedOn[0].MicCode, "XSTO")
	}
	if company.AlsoListedOn[1].TickerSymbol != "NDA-DK" {
		t.Errorf("AlsoListedOn[1].TickerSymbol = %q, want %q", company.AlsoListedOn[1].TickerSymbol, "NDA-DK")
	}
}

func TestCompany_MarketSegmentRoundTrip(t *testing.T) {
	t.Parallel()
	in := `{"id":"cmp1","name":"Elisa","ticker":"ELISA","is_active":true,"exchange":null,"market_segment":"large_cap",` +
		`"stock_listings":[{"ticker":"ELISA","primary":true,"exchange":{"mic_code":"XHEL"},"market_segment":"large_cap"}]}`

	var company Company
	if err := json.Unmarshal([]byte(in), &company); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if company.MarketSegment != "large_cap" {
		t.Errorf("MarketSegment = %q, want large_cap", company.MarketSegment)
	}
	if len(company.StockListings) != 1 || company.StockListings[0].MarketSegment != "large_cap" {
		t.Fatalf("StockListings = %+v, want one large_cap listing", company.StockListings)
	}

	out, err := json.Marshal(company)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]interface{}
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-decoding: %v", err)
	}
	if back["market_segment"] != "large_cap" {
		t.Errorf("market_segment = %v after the round trip, want large_cap", back["market_segment"])
	}
	listing := back["stock_listings"].([]interface{})[0].(map[string]interface{})
	if listing["market_segment"] != "large_cap" {
		t.Errorf("stock_listings[0].market_segment = %v after the round trip, want large_cap", listing["market_segment"])
	}
}
