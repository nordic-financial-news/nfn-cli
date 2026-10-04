package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Browse calendar events",
}

var calendarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Long: `List scheduled corporate dates: earnings reports, AGMs, dividends and the like.

Only dates from today onward are listed unless --scheduled-after,
--scheduled-before or --updated-after is given, so a past report or dividend
needs a window that includes it. Times are shown in each issuer's local time,
and a date with no announced time shows none.`,
	Example: `  nfn calendar list --limit 10
  nfn calendar list --ticker VOLV-B
  nfn calendar list --ticker VOLV-B --scheduled-after 2026-01-01
  nfn calendar list --company uejazctgchj4
  nfn calendar list --event-type earnings_report,dividend
  nfn calendar list --country SE,NO --scheduled-before 2026-12-31
  nfn calendar list --scheduled-after 2026-10-01 --scheduled-before 2026-10-31
  nfn calendar list --market SE --event-type earnings_report
  nfn calendar list --domicile FI
  nfn calendar list --exchange XSTO,XCSE
  nfn calendar list --watchlist-id a1b2c3d4e5f6
  nfn calendar list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := buildCalendarParams(cmd)
		if f.Format() != "json" {
			withWhenFields(params)
		}
		watchlistID, _ := cmd.Flags().GetString("watchlist-id")

		all, _ := cmd.Flags().GetBool("all")
		if all {
			events, resp, err := client.ListAllCalendarEvents(cmd.Context(), params)
			if err != nil {
				return watchlistFilterError(err, watchlistID)
			}
			return renderCalendarEvents(f, events, nil, planLimitedFrom(resp), calendarWindowed(params))
		}

		events, pagination, resp, err := client.ListCalendarEvents(cmd.Context(), params)
		if err != nil {
			return watchlistFilterError(err, watchlistID)
		}
		return renderCalendarEvents(f, events, pagination, planLimitedFrom(resp), calendarWindowed(params))
	},
}

var calendarGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get calendar event details",
	Example: `  nfn calendar get ly89wavugvsl
  nfn calendar get ly89wavugvsl --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		event, _, err := client.GetCalendarEvent(cmd.Context(), args[0])
		if err != nil {
			return notFoundError(err, fmt.Sprintf("no calendar event has ID %q. Find one with 'nfn calendar list --ticker <ticker>'", args[0]))
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(event, fmt.Sprintf("calendar event: %s", event.Title), breadcrumbsFor("nfn calendar get"))
		}

		f.RenderDetail(calendarFields(event))
		for _, line := range calendarNextSteps(event) {
			f.Println(line)
		}
		return nil
	},
}

// calendarFields runs from what the date is, to when, to whose it is. Blank
// fields are left out: most events carry no fiscal period, and a dividend
// often has only some of its dates announced.
func calendarFields(e *api.CalendarEvent) []output.Field {
	var fields []output.Field
	add := func(key, value string) {
		if value != "" {
			fields = append(fields, output.Field{Key: key, Value: value})
		}
	}
	add("ID", e.ID)
	add("Title", e.Title)
	add("Description", e.Description)
	add("Type", calendarTypeLabel(e.EventType))
	add("Fiscal Period", fiscalPeriod(e.FiscalPeriod))
	add("When", calendarDetailWhen(e))
	add("Status", e.Status)
	add("Date Changed", detailTime(e.ScheduledAtChangedAt))
	if c := calendarIssuer(e); c.Name != "" {
		add("Company", handleLines([]api.Company{c}))
	} else {
		add("Company", c.ID)
	}
	add("Country", e.Country)
	// In the order a shareholder meets them: the ex-date decides who is paid.
	if d := e.Dividend; d != nil {
		add("Ex-Dividend", calendarDay(d.ExDate))
		add("Record Date", calendarDay(d.RecordDate))
		add("Payment Date", calendarDay(d.PaymentDate))
		if d.Amount != "" {
			add("Amount", strings.TrimSpace(d.Amount+" "+d.Currency))
		}
		add("Dividend Kind", humanize(d.Kind))
		add("Declared", calendarDay(d.DeclarationDate))
	}
	add("Updated", detailTime(e.UpdatedAt))
	return fields
}

func calendarNextSteps(e *api.CalendarEvent) []string {
	var steps []nextStep
	c := calendarIssuer(e)
	if handle := companyHandle(c); handle != "" {
		steps = append(steps, nextStep{"The company:", "nfn companies get " + handle})
	}
	if e.SourceArticleID != "" {
		steps = append(steps, nextStep{"Where it was announced:", "nfn articles get " + e.SourceArticleID})
	}
	switch {
	case c.TickerSymbol != "":
		steps = append(steps, nextStep{"Its other dates:", "nfn calendar list --ticker " + c.TickerSymbol})
	case c.ID != "":
		steps = append(steps, nextStep{"Its other dates:", "nfn calendar list --company " + c.ID})
	}
	return formatNextSteps(steps)
}

// calendarIssuer is the event's issuer as a Company, so the company helpers
// apply. Only the ID is known when the API withholds the inline issuer.
func calendarIssuer(e *api.CalendarEvent) api.Company {
	c := api.Company{ID: e.CompanyID}
	if e.Company != nil {
		c.Name, c.TickerSymbol = e.Company.Name, e.Company.Ticker
	}
	return c
}

func buildCalendarParams(cmd *cobra.Command) url.Values {
	params := url.Values{}

	if v, _ := cmd.Flags().GetString("ticker"); v != "" {
		params.Set("ticker", v)
	}
	if v, _ := cmd.Flags().GetString("company"); v != "" {
		params.Set("company", v)
	}
	if v, _ := cmd.Flags().GetString("country"); v != "" {
		params.Set("country", v)
	}
	if v, _ := cmd.Flags().GetString("market"); v != "" {
		params.Set("market", v)
	}
	if v, _ := cmd.Flags().GetString("domicile"); v != "" {
		params.Set("domicile", v)
	}
	if v, _ := cmd.Flags().GetString("exchange"); v != "" {
		params.Set("exchange", v)
	}
	if v, _ := cmd.Flags().GetString("index"); v != "" {
		params.Set("index", v)
	}
	if v, _ := cmd.Flags().GetString("sector"); v != "" {
		params.Set("sector", v)
	}
	if v, _ := cmd.Flags().GetString("event-type"); v != "" {
		params.Set("event_type", v)
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		params.Set("status", v)
	}
	if v, _ := cmd.Flags().GetString("scheduled-after"); v != "" {
		params.Set("scheduled_after", v)
	}
	if v, _ := cmd.Flags().GetString("scheduled-before"); v != "" {
		params.Set("scheduled_before", v)
	}
	if v, _ := cmd.Flags().GetString("updated-after"); v != "" {
		params.Set("updated_after", v)
	}
	if v, _ := cmd.Flags().GetString("watchlist-id"); v != "" {
		params.Set("watchlist", v)
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		params.Set("limit", fmt.Sprintf("%d", v))
	}
	if v, _ := cmd.Flags().GetString("cursor"); v != "" {
		params.Set("cursor", v)
	}
	if v, _ := cmd.Flags().GetString("fields"); v != "" {
		params.Set("fields", v)
	}

	return params
}

// windowed is whether the request set its own start; without one the API
// lists only upcoming dates, which the footer says so an empty page is not
// read as "none ever".
func renderCalendarEvents(f *output.Formatter, events []api.CalendarEvent, pagination *api.Pagination, planLimited *bool, windowed bool, cmdPath ...string) error {
	if f.Format() == "json" {
		rows, err := narrowJSON(f.Fields(), events)
		if err != nil {
			return err
		}
		result := map[string]interface{}{"calendar_events": rows}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		addPlanLimited(result, planLimited)
		path := "nfn calendar list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		summary := planLimitedSummary(countNoun(len(events), calendarNoun), planLimited)
		return f.RenderEnvelope(result, summary, breadcrumbsFor(path))
	}

	// Every upcoming date is scheduled, so the column appears only when a row
	// says otherwise. A --fields selection without status leaves it blank,
	// which is not a change of status either.
	showStatus := f.Fields()["status"]
	for _, e := range events {
		if e.Status != "" && e.Status != "scheduled" {
			showStatus = true
		}
	}

	columns := []string{"ID", "When", "Type", "Company", "Title"}
	colFields := [][]string{{"id"}, {"local_time", "scheduled_at", "timezone", "date_precision"}, {"event_type"}, {"company", "company_id"}, {"title"}}
	if showStatus {
		columns = append(columns, "Status")
		colFields = append(colFields, []string{"status"})
	}
	now := time.Now()
	rows := make([][]string, len(events))
	for i, e := range events {
		rows[i] = []string{e.ID, calendarListWhen(e, now), calendarTypeLabel(e.EventType), calendarCompanyCell(e), e.Title}
		if showStatus {
			rows[i] = append(rows[i], e.Status)
		}
	}
	columns, rows = narrowColumns(f.Fields(), columns, rows, colFields)
	f.RenderFit(columns, rows, output.Layout{Flex: "Title", Drop: []string{"Status", "Type", "Company"}, Floor: 20})
	if note := noColumnNote(fieldsWithoutColumn(f.Fields(), colFields)); note != "" {
		f.Println(note)
	}
	footer := listFooter{
		noun:       calendarNoun,
		getCmd:     "nfn calendar get",
		pagination: pagination,
		emptyHint:  "Try a wider window, or fewer filters.",
	}
	if !windowed {
		footer.notes = append(footer.notes, "from today")
		// A capped plan serves only upcoming dates whatever the window, so
		// widening it would not help there.
		if !isPlanLimited(planLimited) {
			footer.emptyHint = fmt.Sprintf("Only upcoming dates are listed; add --scheduled-after %d-01-01 to include past ones.", now.Year())
		}
	}
	for _, e := range events {
		if e.DatePrecision == "exact" {
			footer.notes = append(footer.notes, "times are each issuer's local time")
			break
		}
	}
	footer.print(f, len(events))
	if isPlanLimited(planLimited) {
		f.Println(planLimitedNote)
	}
	return nil
}

// calendarWindowed reports whether the request lifts the API's default of
// listing only dates from today onward: either end of a window does, and so
// does a sync, which is a change feed over past dates too.
func calendarWindowed(params url.Values) bool {
	return params.Get("scheduled_after") != "" || params.Get("scheduled_before") != "" || params.Get("updated_after") != ""
}

// withWhenFields makes a narrowed table request fetch what the When column is
// drawn from, whichever of those fields was named: local_time for the issuer's
// date and date_precision for whether it has a time. Only the columns the user
// named are shown, so the extra fields change nothing else.
func withWhenFields(params url.Values) {
	if fields := params.Get("fields"); fields != "" {
		params.Set("fields", fields+",local_time,date_precision")
	}
}

// calendarTypes are the calendar's event types, as --event-type takes them and
// in the words a listing uses, short enough to keep the Type column narrow.
var calendarTypes = [][2]string{
	{"earnings_report", "Earnings"},
	{"earnings_call", "Earnings call"},
	{"agm", "AGM"},
	{"egm", "EGM"},
	{"dividend", "Dividend"},
	{"capital_markets_day", "Capital markets day"},
	{"trading_update", "Trading update"},
	{"conference_presentation", "Conference"},
	{"ma_announcement", "M&A"},
	{"ma_offer_deadline", "Offer deadline"},
	{"ma_completion", "M&A completion"},
	{"listing", "Listing"},
	{"delisting", "Delisting"},
}

// calendarTypeLabel falls back to the slug in words for a type this build does
// not know, rather than a blank.
func calendarTypeLabel(slug string) string {
	for _, t := range calendarTypes {
		if t[0] == slug {
			return t[1]
		}
	}
	return humanize(slug)
}

func calendarTypeSlugs() string {
	slugs := make([]string, len(calendarTypes))
	for i, t := range calendarTypes {
		slugs[i] = t[0]
	}
	return strings.Join(slugs, ", ")
}

// fiscalPeriod turns q3_2026 into Q3 2026.
func fiscalPeriod(p string) string {
	return strings.ToUpper(humanize(p))
}

// calendarCompanyCell names the issuer by ticker, which is how a listing is
// scanned, falling back to the name for an unlisted issuer and to the raw
// company_id when the issuer is withheld or --fields dropped it.
func calendarCompanyCell(e api.CalendarEvent) string {
	switch {
	case e.Company != nil && e.Company.Ticker != "":
		return e.Company.Ticker
	case e.Company != nil && e.Company.Name != "":
		return output.Ellipsize(e.Company.Name, 20)
	default:
		return e.CompanyID
	}
}

// calendarLocal is when the event falls in the issuer's own timezone, which
// local_time carries. scheduled_at is UTC, where a date stored at local
// midnight falls on the day before, so it is only a fallback.
func calendarLocal(e api.CalendarEvent) (time.Time, bool) {
	for _, s := range []string{e.LocalTime, e.ScheduledAt} {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// calendarListWhen is the day, with the weekday since a calendar is read by
// week, and the time only when one was announced: a day-precision date sits
// at a nominal midnight that means "some time that day".
func calendarListWhen(e api.CalendarEvent, now time.Time) string {
	t, ok := calendarLocal(e)
	if !ok {
		return e.LocalTime
	}
	layout := "Mon 2 Jan"
	if t.Year() != now.Year() {
		layout = "2 Jan 2006"
	}
	if e.DatePrecision == "exact" {
		layout += " 15:04"
	}
	return t.Format(layout)
}

// calendarDetailWhen says outright when no time has been announced, and names
// the city whose clock an announced time is on.
func calendarDetailWhen(e *api.CalendarEvent) string {
	t, ok := calendarLocal(*e)
	if !ok {
		return e.LocalTime
	}
	day := t.Format("Mon 2 Jan 2006")
	if e.DatePrecision != "exact" {
		return day + ", time not announced"
	}
	return fmt.Sprintf("%s, %s %s", day, t.Format("15:04"), zoneClock(e.Timezone, t))
}

// zoneClock names a timezone the way people say it: "Stockholm time" for
// Europe/Stockholm, falling back to the offset's abbreviation.
func zoneClock(tz string, t time.Time) string {
	if i := strings.LastIndex(tz, "/"); i >= 0 {
		return humanize(tz[i+1:]) + " time"
	}
	return t.Format("MST")
}

// calendarDay renders a bare API date (2026-10-05) with its weekday.
func calendarDay(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("Mon 2 Jan 2006")
}

func init() {
	calendarListCmd.Flags().String("ticker", "", "Filter by company ticker (comma-separated, max 25, e.g. VOLV-B,ERIC-B); every value must resolve or the request errors")
	calendarListCmd.Flags().String("company", "", "Filter by company ID (comma-separated, max 25; reaches unlisted companies with no ticker); every value must resolve or the request errors")
	calendarListCmd.Flags().String("country", "", "Filter by the event's own country (comma-separated ISO2): the issuer's country, falling back to the source's")
	calendarListCmd.Flags().String("market", "", "Filter by listing-venue country (comma-separated ISO2, e.g. SE,NO): issuers with an active equity listing on an exchange in these countries, including foreign-domiciled issuers")
	calendarListCmd.Flags().String("domicile", "", "Filter by legal domicile country (comma-separated ISO2, e.g. SE,NO): reaches unlisted issuers, excludes foreign-domiciled issuers listed locally")
	calendarListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (comma-separated, e.g. XSTO,XCSE)")
	calendarListCmd.Flags().String("index", "", "Filter by stock index ID or symbol (e.g. OMXS30)")
	calendarListCmd.Flags().String("sector", "", "Filter by company sector (e.g. Financials)")
	calendarListCmd.Flags().String("event-type", "", "Filter by event type (comma-separated): "+calendarTypeSlugs())
	calendarListCmd.Flags().String("status", "", "Filter by status (scheduled, published, cancelled)")
	calendarListCmd.Flags().String("scheduled-after", "", "Start of the scheduled-time window. A bare date (2026-10-01) is that calendar date in the issuer's own timezone, inclusive; an ISO 8601 datetime (2026-10-01T00:00:00Z, or with a UTC offset) is that exact instant, at or after")
	calendarListCmd.Flags().String("scheduled-before", "", "End of the scheduled-time window. A bare date (2026-10-31) is that calendar date in the issuer's own timezone, inclusive of the whole day; an ISO 8601 datetime (2026-11-01T00:00:00Z, or with a UTC offset) is that exact instant, strictly before")
	calendarListCmd.Flags().String("updated-after", "", "ISO 8601 datetime for incremental sync")
	calendarListCmd.Flags().String("watchlist-id", "", "Only show events for companies in this watchlist (see 'nfn watchlists list')")
	addPaginationFlags(calendarListCmd)
	addFieldsFlagFor(calendarListCmd, "calendar_event")

	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarGetCmd)
	rootCmd.AddCommand(calendarCmd)
}

var calendarNoun = [2]string{"calendar event", "calendar events"}
