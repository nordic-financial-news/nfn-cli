package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
)

func parseAPITime(s string, loc *time.Location) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, false
	}
	return t.In(loc), true
}

// listTime renders an API timestamp for a list column in the viewer's zone, as
// short as it can be without becoming ambiguous: the time alone today, the day
// and time this year, the date alone before that. A value that does not parse
// is shown as received, and blank stays blank.
func listTime(s string) string {
	return formatListTime(s, time.Now(), time.Local)
}

func formatListTime(s string, now time.Time, loc *time.Location) string {
	t, ok := parseAPITime(s, loc)
	if !ok {
		return s
	}
	now = now.In(loc)
	switch {
	case sameDay(t, now):
		return t.Format("15:04")
	case t.Year() == now.Year():
		return t.Format("2 Jan 15:04")
	default:
		return t.Format("2 Jan 2006")
	}
}

// detailTime renders an API timestamp in full for a detail view, in the
// viewer's zone and naming it, so nothing about the instant is left implicit.
func detailTime(s string) string {
	return formatDetailTime(s, time.Local)
}

func formatDetailTime(s string, loc *time.Location) string {
	t, ok := parseAPITime(s, loc)
	if !ok {
		return s
	}
	return t.Format("2 Jan 2006, 15:04 MST")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// listFooter is what a list table cannot say for itself: how many rows it
// holds, how to open one, and how to reach the next page. noun is singular and
// plural ("event", "events"), getCmd the detail command its IDs feed, and notes
// any extra clauses for the count line. An empty page says so in place of a
// bare header.
type listFooter struct {
	noun       [2]string
	getCmd     string
	pagination *api.Pagination
	notes      []string
	emptyHint  string
}

func (lf listFooter) print(f *output.Formatter, count int) {
	if count == 0 {
		msg := fmt.Sprintf("No %s match.", lf.noun[1])
		if lf.emptyHint != "" {
			msg += " " + lf.emptyHint
		}
		f.Println(msg)
		return
	}
	noun := lf.noun[1]
	if count == 1 {
		noun = lf.noun[0]
	}
	line := fmt.Sprintf("%d %s", count, noun)
	if len(lf.notes) > 0 {
		line += " (" + strings.Join(lf.notes, "; ") + ")"
	}
	line += "."
	if lf.getCmd != "" {
		line += " Details: " + lf.getCmd + " <id>"
	}
	f.Println(line)
	if lf.pagination != nil && lf.pagination.NextCursor != "" {
		f.Println("Next page: add --cursor " + lf.pagination.NextCursor)
	}
}

// companyNotFoundError names the identifier a company route could not resolve,
// where the API's 404 says only that some resource was missing.
func companyNotFoundError(err error, identifier string) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return fmt.Errorf("no company matches %q. Find its ticker or ID with 'nfn companies list --q <name>'", identifier)
	}
	return err
}

// eventTypeError points an unrecognized --event-type slug at the CLI command
// that lists the vocabulary, where the API's message names its own endpoint.
func eventTypeError(err error) error {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Error(), "/event_types") {
		return err
	}
	const endpoint, command = "GET /api/v1/event_types", "'nfn event-types'"
	rewritten := *apiErr
	rewritten.Detail = strings.Replace(apiErr.Detail, endpoint, command, 1)
	rewritten.Title = strings.Replace(apiErr.Title, endpoint, command, 1)
	return &rewritten
}
