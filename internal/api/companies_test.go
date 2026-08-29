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
