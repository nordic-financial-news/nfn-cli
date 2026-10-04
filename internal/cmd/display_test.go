package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

func TestFormatListTime(t *testing.T) {
	t.Parallel()

	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Skipf("zone data unavailable: %v", err)
	}
	now := time.Date(2026, 10, 4, 14, 0, 0, 0, stockholm)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "today shows the time alone, in the viewer's zone", in: "2026-10-04T08:06:00.000Z", want: "10:06"},
		{name: "late UTC yesterday can be early today locally", in: "2026-10-03T22:30:00Z", want: "00:30"},
		{name: "earlier this year shows day and time", in: "2026-10-02T06:00:00.000Z", want: "2 Oct 08:00"},
		{name: "a previous year shows the date alone", in: "2025-12-30T10:00:00Z", want: "30 Dec 2025"},
		{name: "blank stays blank", in: "", want: ""},
		{name: "unparseable is shown as received", in: "2026-10", want: "2026-10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatListTime(tt.in, now, stockholm); got != tt.want {
				t.Errorf("formatListTime(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatDetailTime(t *testing.T) {
	t.Parallel()

	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Skipf("zone data unavailable: %v", err)
	}
	if got, want := formatDetailTime("2026-10-02T06:00:00.000Z", stockholm), "2 Oct 2026, 08:00 CEST"; got != want {
		t.Errorf("formatDetailTime() = %q, want %q", got, want)
	}
	if got := formatDetailTime("", stockholm); got != "" {
		t.Errorf("formatDetailTime(\"\") = %q, want blank", got)
	}
}

func TestListFooter(t *testing.T) {
	t.Parallel()

	render := func(lf listFooter, count int) string {
		var buf bytes.Buffer
		lf.print(output.NewFormatterWithWriter("table", true, &buf), count)
		return buf.String()
	}

	t.Run("names the count, notes and detail command", func(t *testing.T) {
		t.Parallel()
		got := render(listFooter{noun: [2]string{"event", "events"}, getCmd: "nfn events get", notes: []string{"all notable"}}, 3)
		if want := "3 events (all notable). Details: nfn events get <id>\n"; got != want {
			t.Errorf("footer = %q, want %q", got, want)
		}
	})

	t.Run("offers the next cursor when there is one", func(t *testing.T) {
		t.Parallel()
		got := render(listFooter{noun: [2]string{"article", "articles"}, pagination: &api.Pagination{Count: 2, NextCursor: "abc123"}}, 2)
		if !strings.Contains(got, "Next page: add --cursor abc123") {
			t.Errorf("footer = %q, want the next cursor", got)
		}
	})

	t.Run("says nothing about paging on the last page", func(t *testing.T) {
		t.Parallel()
		got := render(listFooter{noun: [2]string{"article", "articles"}, pagination: &api.Pagination{Count: 2}}, 2)
		if strings.Contains(got, "cursor") {
			t.Errorf("footer = %q, want no cursor line", got)
		}
	})

	t.Run("a single row is singular", func(t *testing.T) {
		t.Parallel()
		got := render(listFooter{noun: [2]string{"story", "stories"}}, 1)
		if want := "1 story.\n"; got != want {
			t.Errorf("footer = %q, want %q", got, want)
		}
	})

	t.Run("an empty page says so", func(t *testing.T) {
		t.Parallel()
		got := render(listFooter{noun: [2]string{"event", "events"}, getCmd: "nfn events get", emptyHint: "Try a wider --window."}, 0)
		if want := "No events match. Try a wider --window.\n"; got != want {
			t.Errorf("footer = %q, want %q", got, want)
		}
	})
}

func TestCompanyNotFoundError(t *testing.T) {
	t.Parallel()

	got := companyNotFoundError(&api.APIError{StatusCode: 404, Title: "The requested resource could not be found"}, "NOTATICKER")
	if !strings.Contains(got.Error(), `"NOTATICKER"`) || !strings.Contains(got.Error(), "nfn companies list --q") {
		t.Errorf("error = %q, want it to name the identifier and how to find one", got)
	}

	other := &api.APIError{StatusCode: 403, Title: "Forbidden"}
	if got := companyNotFoundError(other, "VOLV-B"); got != other {
		t.Errorf("a non-404 error was rewritten to %q", got)
	}
}

func TestEventTypeError(t *testing.T) {
	t.Parallel()

	in := &api.APIError{StatusCode: 400, Detail: "Unrecognized event type slug(s): banana. GET /api/v1/event_types lists the active vocabulary."}
	got := eventTypeError(in)
	if want := "Unrecognized event type slug(s): banana. 'nfn event-types' lists the active vocabulary."; got.Error() != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	var apiErr *api.APIError
	if !errors.As(got, &apiErr) || apiErr.StatusCode != 400 {
		t.Errorf("rewritten error lost its API status: %#v", got)
	}

	other := errors.New("network down")
	if got := eventTypeError(other); got != other {
		t.Errorf("an unrelated error was rewritten to %q", got)
	}
}

func TestFormatNextSteps(t *testing.T) {
	t.Parallel()
	got := formatNextSteps([]nextStep{{"Events:", "nfn companies events VOLV-B"}, {"637 articles:", "nfn companies articles VOLV-B"}})
	want := []string{"Events:       nfn companies events VOLV-B", "637 articles: nfn companies articles VOLV-B"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("formatNextSteps() = %q, want %q", got, want)
	}
}

// Every line must start with something 'nfn companies get' accepts, so a
// company without a ticker is shown by its ID.
func TestHandleLines(t *testing.T) {
	t.Parallel()
	got := handleLines([]api.Company{{ID: "uejazctgchj4", Name: "AB Volvo", TickerSymbol: "VOLV-B"}, {ID: "hp14n1ibr1dz", Name: "Volvo Penta"}})
	want := "VOLV-B        AB Volvo\nhp14n1ibr1dz  Volvo Penta"
	if got != want {
		t.Errorf("handleLines() = %q, want %q", got, want)
	}
}

func TestOrderNote(t *testing.T) {
	newCmd := func() *cobra.Command {
		c := &cobra.Command{}
		c.Flags().String("q", "", "")
		c.Flags().String("sort", "", "")
		c.Flags().Bool("semantic", false, "")
		return c
	}
	tests := []struct {
		name      string
		flags     map[string]string
		relevance bool
		want      string
	}{
		{name: "a feed is newest first and says nothing", relevance: true, want: ""},
		{name: "--q alone ranks by relevance", flags: map[string]string{"q": "battery"}, relevance: true, want: "ranked by relevance; add --sort latest for newest first"},
		{name: "--sort latest restores date order", flags: map[string]string{"q": "battery", "sort": "latest"}, relevance: true, want: ""},
		{name: "--semantic ranks by meaning", flags: map[string]string{"q": "battery", "semantic": "true"}, want: "ranked by meaning"},
		{name: "a command whose --q is not relevance-ranked", flags: map[string]string{"q": "Volvo"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newCmd()
			for k, v := range tt.flags {
				_ = c.Flags().Set(k, v)
			}
			if got := orderNote(c, tt.relevance); got != tt.want {
				t.Errorf("orderNote() = %q, want %q", got, tt.want)
			}
		})
	}
}
