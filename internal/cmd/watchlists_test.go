package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
)

func sampleWatchlists() []api.Watchlist {
	return []api.Watchlist{
		{ID: "wl1", Name: "Nordic Banks", CompanyCount: 4},
		{ID: "wl2", Name: "Green Energy", CompanyCount: 7},
	}
}

func TestRenderWatchlists_Table(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := output.NewFormatterWithWriter("table", true, &buf)

	if err := renderWatchlists(f, sampleWatchlists()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// The table renderer uppercases column headers.
	for _, want := range []string{"ID", "NAME", "COMPANIES", "wl1", "Nordic Banks", "4", "wl2", "Green Energy", "7"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestRenderWatchlists_JSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := output.NewFormatterWithWriter("json", true, &buf)

	if err := renderWatchlists(f, sampleWatchlists()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Watchlists []api.Watchlist `json:"watchlists"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, buf.String())
	}
	if !envelope.OK {
		t.Errorf("envelope.ok = false, want true")
	}
	if len(envelope.Data.Watchlists) != 2 {
		t.Fatalf("got %d watchlists, want 2", len(envelope.Data.Watchlists))
	}
	if envelope.Data.Watchlists[0].ID != "wl1" {
		t.Errorf("ID = %q, want %q", envelope.Data.Watchlists[0].ID, "wl1")
	}
}

func TestRenderWatchlists_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := output.NewFormatterWithWriter("json", true, &buf)

	if err := renderWatchlists(f, []api.Watchlist{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envelope struct {
		Data struct {
			Watchlists []api.Watchlist `json:"watchlists"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(envelope.Data.Watchlists) != 0 {
		t.Errorf("got %d watchlists, want 0", len(envelope.Data.Watchlists))
	}
}
