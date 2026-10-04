package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
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
	line := countNoun(count, lf.noun)
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

// countNoun pairs a count with the singular or plural noun it takes.
func countNoun(n int, noun [2]string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun[0])
	}
	return fmt.Sprintf("%d %s", n, noun[1])
}

// nextStep is one line of the "where to go from here" block a detail view ends
// with: a short label and the command or link it leads to.
type nextStep struct{ label, target string }

// formatNextSteps lines the targets up in one column so they scan as a list.
func formatNextSteps(steps []nextStep) []string {
	width := 0
	for _, s := range steps {
		width = max(width, len(s.label))
	}
	lines := make([]string, len(steps))
	for i, s := range steps {
		lines[i] = fmt.Sprintf("%-*s %s", width, s.label, s.target)
	}
	return lines
}

// handleLines renders companies one per line as "handle  name", the handle
// being the ticker when there is one and the ID otherwise, so every line starts
// with something 'nfn companies get' accepts. Handles are padded to one width
// so the names line up.
func handleLines(companies []api.Company) string {
	width := 0
	for _, c := range companies {
		width = max(width, len(companyHandle(c)))
	}
	lines := make([]string, len(companies))
	for i, c := range companies {
		lines[i] = fmt.Sprintf("%-*s  %s", width, companyHandle(c), c.Name)
	}
	return strings.Join(lines, "\n")
}

func companyHandle(c api.Company) string {
	if c.TickerSymbol != "" {
		return c.TickerSymbol
	}
	return c.ID
}

// humanize turns an API slug such as press_release into words.
func humanize(slug string) string {
	return strings.ReplaceAll(slug, "_", " ")
}

// notFoundError replaces the API's 404, which says only that some resource was
// missing, with msg naming what was looked up and how to find a real one.
func notFoundError(err error, msg string) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return errors.New(msg)
	}
	return err
}

// listOpts carries what a shared list renderer cannot work out from the rows:
// the command path for the JSON breadcrumbs (blank for the renderer's own
// list command) and, when the rows are not newest first, how they are ordered.
type listOpts struct {
	path  string
	order string
}

func (o listOpts) pathOr(fallback string) string {
	if o.path != "" {
		return o.path
	}
	return fallback
}

func (o listOpts) notes() []string {
	if o.order == "" {
		return nil
	}
	return []string{o.order}
}

// orderNote says how a search orders its page when the order is not by date,
// since a relevance-ranked page otherwise reads as a broken date sort.
// relevance is whether --q without --sort ranks by relevance on this command.
func orderNote(cmd *cobra.Command, relevance bool) string {
	if semanticRequested(cmd) {
		return "ranked by meaning"
	}
	if !relevance || cmd.Flags().Lookup("sort") == nil {
		return ""
	}
	q, _ := cmd.Flags().GetString("q")
	sort, _ := cmd.Flags().GetString("sort")
	if q == "" || sort != "" {
		return ""
	}
	return "ranked by relevance; add --sort latest for newest first"
}

// companyNotFoundError names the identifier a company route could not resolve,
// where the API's 404 says only that some resource was missing.
func companyNotFoundError(err error, identifier string) error {
	return notFoundError(err, fmt.Sprintf("no company matches %q. Find its ticker or ID with 'nfn companies list --q <name>'", identifier))
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
