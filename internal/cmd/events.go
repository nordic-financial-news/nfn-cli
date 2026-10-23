package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Browse events",
	Long: `Browse events.

An event groups the articles reporting one real-world development — an earnings
report, an acquisition, an executive change. This is distinct from 'nfn calendar',
which lists scheduled future dates.`,
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long: `List events, most recently reported first.

The unfiltered feed mixes all three significance tiers, so opinion pieces and
routine filings sit alongside material news. Pass --significance notable to
restrict it to material events.

--event-type takes the hyphenated Events taxonomy slugs from 'nfn event-types'.
The underscored values 'nfn calendar list' takes are rejected here, and a few
spellings overlap across the two vocabularies while denoting different things.

Unlike 'nfn articles list', --q here filters the ordinary feed order rather than
returning a relevance-ranked page, so it composes with --window, --sort and --all.`,
	Example: `  nfn events list --limit 10
  nfn events list --significance notable
  nfn events list --event-type earnings-report,acquisition-announced
  nfn events list --ticker VOLV-B --window 7d
  nfn events list --q "battery" --all
  nfn events list --q "defence contract wins and capacity expansion" --semantic
  nfn events list --sort significance --market SE
  nfn events list --related-to b8n3vx6plq2d
  nfn events list --updated-after 2026-03-01T00:00:00Z --all
  nfn events list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		if err := validateSemantic(cmd); err != nil {
			return err
		}
		if err := validateEventType(cmd); err != nil {
			return err
		}

		params := buildEventParams(cmd)
		watchlistID, _ := cmd.Flags().GetString("watchlist-id")
		semantic := semanticRequested(cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			events, resp, err := client.ListAllEvents(cmd.Context(), params)
			if err != nil {
				return eventTypeError(watchlistFilterError(err, watchlistID))
			}
			return renderEvents(f, events, nil, planLimitedFrom(resp), truncatedFrom(resp))
		}

		events, pagination, resp, err := client.ListEvents(cmd.Context(), params)
		if err != nil {
			return eventTypeError(semanticError(watchlistFilterError(err, watchlistID), semantic))
		}
		return renderEvents(f, events, pagination, planLimitedFrom(resp), truncatedFrom(resp))
	},
}

var eventsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get event details",
	Long: `Get event details.

A merged or removed event still resolves, to a tombstone carrying only its id,
status, surviving event and update time. An event whose coverage is entirely
outside your plan's reach returns 403 rather than 404.`,
	Example: `  nfn events get h2k9rqz4m7wt
  nfn events get h2k9rqz4m7wt --include-peripheral
  nfn events get h2k9rqz4m7wt --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		if v, _ := cmd.Flags().GetBool("include-peripheral"); v {
			params.Set("include_peripheral", "true")
		}

		event, _, err := client.GetEvent(cmd.Context(), args[0], params)
		if err != nil {
			return notFoundError(err, fmt.Sprintf("no event has ID %q. Find one with 'nfn events list'", args[0]))
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(event, eventSummaryLine(event), breadcrumbsFor("nfn events get"))
		}

		// A tombstone has no title, summary, counts or id arrays, so the full
		// field list would read as an event whose content went missing.
		if event.IsTombstone() {
			fields := []output.Field{
				{Key: "ID", Value: event.ID},
				{Key: "Status", Value: event.Status},
			}
			// Only a merge has a survivor; on a removal the row would always be
			// blank, and the note already says what happened.
			if event.MergedInto != "" {
				fields = append(fields, output.Field{Key: "Merged Into", Value: event.MergedInto})
			}
			fields = append(fields, output.Field{Key: "Updated", Value: detailTime(event.UpdatedAt)})
			f.RenderDetail(fields)
			f.Println(eventTombstoneNote(event))
			return nil
		}

		eventType := ""
		if event.EventType != nil {
			eventType = event.EventType.Label
			if event.EventType.Slug != "" {
				eventType += " (" + event.EventType.Slug + ")"
			}
		}

		// Rows run from what happened, to who and when, to the IDs that lead
		// further. Status is left out: on a live event it is active or closed,
		// which the dates already say (see the tombstone branch above for the
		// states that matter).
		fields := []output.Field{
			{Key: "ID", Value: event.ID},
			{Key: "Title", Value: event.Title},
			{Key: "Type", Value: eventType},
			{Key: "Significance", Value: event.Significance},
		}

		if len(event.Companies) > 0 {
			labels := make([]string, len(event.Companies))
			for i, c := range event.Companies {
				labels[i] = eventCompanyLabel(c)
			}
			fields = append(fields, output.Field{Key: "Companies", Value: strings.Join(labels, "\n")})
		} else if len(event.CompanyIDs) > 0 {
			fields = append(fields, output.Field{Key: "Company IDs", Value: strings.Join(event.CompanyIDs, "\n")})
		}

		fields = append(fields,
			output.Field{Key: "First Reported", Value: detailTime(event.FirstReportedAt)},
			output.Field{Key: "Last Article", Value: detailTime(event.LastArticleAt)},
			output.Field{Key: "Summary", Value: event.Summary},
			output.Field{Key: "Articles", Value: strconv.Itoa(event.ArticleCount)},
		)

		// One ID per line, so each can be copied whole.
		if len(event.ArticleIDs) > 0 {
			fields = append(fields, output.Field{Key: "Article IDs", Value: strings.Join(event.ArticleIDs, "\n")})
		}
		if event.PrimaryDisclosureArticleID != "" {
			fields = append(fields, output.Field{Key: "Primary Disclosure", Value: event.PrimaryDisclosureArticleID})
		}
		if len(event.StoryIDs) > 0 {
			fields = append(fields, output.Field{Key: "Story IDs", Value: strings.Join(event.StoryIDs, "\n")})
		}
		if event.RelatedTo != "" {
			fields = append(fields, output.Field{Key: "Comments On", Value: event.RelatedTo})
		}
		if event.MemberFiguresDisagree != nil && *event.MemberFiguresDisagree {
			fields = append(fields, output.Field{Key: "Figures Disagree", Value: "Yes"})
			if lines := eventDivergenceLines(event.FigureDivergence); len(lines) > 0 {
				fields = append(fields, output.Field{Key: "Divergence", Value: strings.Join(lines, "\n")})
			}
		}
		fields = append(fields, output.Field{Key: "Updated", Value: detailTime(event.UpdatedAt)})

		f.RenderDetail(fields)
		for _, hint := range eventNextSteps(event) {
			f.Println(hint)
		}
		return nil
	},
}

var eventTypesCmd = &cobra.Command{
	Use:   "event-types",
	Short: "List event types",
	Long: `List the Events taxonomy: every value --event-type accepts on 'nfn events list'.

These hyphenated slugs are a separate vocabulary from the underscored
--event-type values 'nfn calendar list' takes. Each rejects the other's values,
and a few spellings overlap while denoting different things.

The table cuts each type's definition to the terminal width; --format json
carries the full text, which is what distinguishes the near neighbours.`,
	Example: `  nfn event-types
  nfn event-types --fields slug,label
  nfn event-types --format json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		if v, _ := cmd.Flags().GetString("fields"); v != "" {
			params.Set("fields", v)
		}

		eventTypes, pagination, _, err := client.ListEventTypes(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderEventTypes(f, eventTypes, pagination)
	},
}

// validateEventType rejects an underscored --event-type value before a request
// is spent on it. Every slug in the Events taxonomy is hyphenated, so an
// underscore means the value came from another vocabulary — most often the
// calendar event types, which this filter rejects and which overlap it in a few
// spellings. The API's own 400 names the bad slug but not that cause.
func validateEventType(cmd *cobra.Command) error {
	v, _ := cmd.Flags().GetString("event-type")
	var underscored []string
	for _, slug := range strings.Split(v, ",") {
		if strings.Contains(slug, "_") {
			underscored = append(underscored, strings.TrimSpace(slug))
		}
	}
	if len(underscored) == 0 {
		return nil
	}
	verb := "is"
	if len(underscored) > 1 {
		verb = "are"
	}
	return fmt.Errorf(
		"--event-type takes hyphenated Events taxonomy slugs, and %s %s underscored. "+
			"Underscored values are calendar event types, which 'nfn calendar list --event-type' takes instead "+
			"(a few spellings overlap while meaning different things). Run 'nfn event-types' for the slugs this filter accepts",
		strings.Join(quoteAll(underscored), ", "),
		verb,
	)
}

// quoteAll quotes each value for inclusion in an error message.
func quoteAll(values []string) []string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return quoted
}

// eventFilterFlags maps the event-feed filter flags to their query-param names,
// beyond the content filters setContentFilters already covers. As there, only a
// flag the command registered and the user set is sent, so the nested company
// route never forwards a filter it rejects — it simply does not register one.
var eventFilterFlags = []struct{ flag, param string }{
	{"event-type", "event_type"},
	{"significance", "significance"},
	{"ticker", "ticker"},
	{"company", "company"},
	{"exchange", "exchange"},
	{"market", "market"},
	{"domicile", "domicile"},
	{"index", "index"},
	{"sector", "sector"},
	{"market-segment", "market_segment"},
	{"watchlist-id", "watchlist"},
	{"first-reported-after", "first_reported_after"},
	{"first-reported-before", "first_reported_before"},
	{"last-article-after", "last_article_after"},
	{"last-article-before", "last_article_before"},
	{"window", "window"},
	{"related-to", "related_to"},
}

// eventBoolFilterFlags maps the event-feed boolean flags to their query-param
// names. Only the on value is ever sent: listed=false and
// include_peripheral=false are the API's own defaults, and mode has no off
// value, so an unset flag means "say nothing" rather than "say false".
var eventBoolFilterFlags = []struct{ flag, param, value string }{
	{"listed", "listed", "true"},
	{"include-peripheral", "include_peripheral", "true"},
	{"semantic", "mode", "semantic"},
}

func buildEventParams(cmd *cobra.Command) url.Values {
	params := url.Values{}

	// The events feed shares q, sort, country, category, sources, source_type,
	// content_type, updated_after and fields with the article and story feeds.
	// It has no published_after/published_before — neither command registers
	// them, so neither sends them.
	setContentFilters(params, cmd)

	for _, f := range eventFilterFlags {
		if cmd.Flags().Lookup(f.flag) == nil {
			continue
		}
		if v, _ := cmd.Flags().GetString(f.flag); v != "" {
			params.Set(f.param, v)
		}
	}

	for _, f := range eventBoolFilterFlags {
		if cmd.Flags().Lookup(f.flag) == nil {
			continue
		}
		if v, _ := cmd.Flags().GetBool(f.flag); v {
			params.Set(f.param, f.value)
		}
	}
	setPaginationParams(params, cmd)

	return params
}

func renderEvents(f *output.Formatter, events []api.Event, pagination *api.Pagination, planLimited, truncated *bool, cmdPath ...string) error {
	if f.Format() == "json" {
		rows, err := narrowJSON(f.Fields(), events)
		if err != nil {
			return err
		}
		result := map[string]interface{}{"events": rows}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		addPlanLimited(result, planLimited)
		path := "nfn events list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		summary := planLimitedSummary(countNoun(len(events), eventNoun), planLimited)
		if truncated != nil {
			result["truncated"] = *truncated
			if *truncated {
				summary += " (more matches exist beyond the search horizon)"
			}
		}
		return f.RenderEnvelope(result, summary, breadcrumbsFor(path))
	}

	// Only a semantic search returns distances, so the column appears only when
	// the results actually carry them.
	showDistance := false
	// Tombstones reach a table only on an --updated-after sync, where their rows
	// are near-empty; the status column is what explains them.
	showStatus := false
	for _, e := range events {
		if e.Distance != nil {
			showDistance = true
		}
		if e.IsTombstone() {
			showStatus = true
		}
	}

	// A tier marker that never varies says nothing per row, so a page of one
	// tier names it once in the footer instead. The same goes for the article
	// count, which is 1 on most fresh events. Either still gets its column when
	// --fields asks for it by name.
	want := f.Fields()
	tiers := map[string]bool{}
	untiered := false
	showArticles := want["article_count"]
	for _, e := range events {
		if e.Significance != "" {
			tiers[e.Significance] = true
		} else {
			untiered = true
		}
		if e.ArticleCount > 1 {
			showArticles = true
		}
	}
	showTier := len(tiers) > 1 || want["significance"]

	columns := []string{"ID", "Reported"}
	colFields := [][]string{{"id"}, {"first_reported_at"}}
	if showTier {
		columns = append(columns, "Sig")
		colFields = append(colFields, []string{"significance"})
	}
	columns = append(columns, "Type", "Title")
	colFields = append(colFields, []string{"event_type"}, []string{"title"})
	if showArticles {
		columns = append(columns, "Articles")
		colFields = append(colFields, []string{"article_count"})
	}
	if showStatus {
		columns = append(columns, "Status")
	}
	if showDistance {
		columns = append(columns, "Distance")
	}
	rows := make([][]string, len(events))
	for i, e := range events {
		eventType := ""
		if e.EventType != nil {
			eventType = e.EventType.Slug
		}
		row := []string{e.ID, listTime(e.FirstReportedAt)}
		if showTier {
			row = append(row, tierMarker(e.Significance))
		}
		row = append(row, eventType, e.Title)
		if showArticles {
			row = append(row, eventArticleCount(e))
		}
		if showStatus {
			row = append(row, e.Status)
		}
		if showDistance {
			row = append(row, formatDistance(e.Distance))
		}
		rows[i] = row
	}
	columns, rows = narrowColumns(f.Fields(), columns, rows, colFields)
	f.RenderFit(columns, rows, output.Layout{Flex: "Title", Drop: []string{"Articles", "Type", "Sig"}})
	if note := noColumnNote(fieldsWithoutColumn(f.Fields(), colFields)); note != "" {
		f.Println(note)
	}
	footer := listFooter{
		noun:       eventNoun,
		getCmd:     "nfn events get",
		pagination: pagination,
		emptyHint:  "Try a wider --window, or fewer filters.",
	}
	switch {
	case showTier:
		footer.notes = append(footer.notes, tierLegend)
	case len(tiers) == 1 && !untiered:
		for tier := range tiers {
			footer.notes = append(footer.notes, "all "+tier)
		}
	}
	footer.print(f, len(events))
	if isPlanLimited(planLimited) {
		f.Println(planLimitedNote)
	}
	if truncated != nil && *truncated {
		f.Println(truncatedNote)
	}
	return nil
}

func renderEventTypes(f *output.Formatter, eventTypes []api.EventTypeDetail, pagination *api.Pagination) error {
	if f.Format() == "json" {
		rows, err := narrowJSON(f.Fields(), eventTypes)
		if err != nil {
			return err
		}
		result := map[string]interface{}{"event_types": rows}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d event types", len(eventTypes)), breadcrumbsFor("nfn event-types"))
	}

	// Slug leads because --event-type takes it. The API groups types by
	// category, so Category next reads as sections. Near neighbours share
	// their summaries' opening words, so Summary gets the spare width.
	columns := []string{"Slug", "Category", "Label", "Summary"}
	rows := make([][]string, len(eventTypes))
	for i, t := range eventTypes {
		rows[i] = []string{t.Slug, t.Category, t.Label, t.Summary}
	}
	colFields := [][]string{{"slug"}, {"category"}, {"label"}, {"summary"}}
	columns, rows = narrowColumns(f.Fields(), columns, rows, colFields)
	f.RenderFit(columns, rows, output.Layout{Flex: "Summary", Drop: []string{"Label", "Category"}})
	if note := noColumnNote(fieldsWithoutColumn(f.Fields(), colFields)); note != "" {
		f.Println(note)
	}
	switch {
	case len(eventTypes) == 0:
		f.Println("No event types returned.")
	case f.Fields() == nil || f.Fields()["summary"]:
		f.Println("Full definitions: nfn event-types --format json")
	}
	return nil
}

// truncatedNote is printed under table output when a semantic events search
// exhausted its match horizon, so a short page isn't read as the whole answer.
// Only the events feed reports it.
const truncatedNote = "Note: more matching events exist beyond the search horizon — no cursor reaches them. Narrow the query or add filters."

// truncatedFrom returns the truncated flag carried on an API response, the
// events-only counterpart to planLimitedFrom. Unlike plan_limited it needs no
// stickiness across a --all walk: only a semantic search reports it, and a
// semantic response has no cursor to walk.
func truncatedFrom(resp *api.Response) *bool {
	if resp == nil {
		return nil
	}
	return resp.Truncated
}

// tierMarker is the one-cell significance marker in the events table, heavier
// for the tiers that matter more. tierLegend decodes it under the table.
func tierMarker(significance string) string {
	switch significance {
	case "notable":
		return "●"
	case "routine":
		return "○"
	case "commentary":
		return "·"
	}
	// A tier this build does not know is shown by name rather than as a blank
	// the legend cannot explain.
	return significance
}

const tierLegend = "● notable, ○ routine, · commentary"

// eventArticleCount renders the member-article count, left blank on a tombstone
// where the API omits the field rather than reporting zero.
func eventArticleCount(e api.Event) string {
	if e.IsTombstone() {
		return ""
	}
	return strconv.Itoa(e.ArticleCount)
}

// eventNextSteps are the follow-up commands printed under an event's detail
// table: the hops its IDs lead to, which JSON output carries as breadcrumbs.
func eventNextSteps(e *api.Event) []string {
	var steps []nextStep
	switch {
	case e.PrimaryDisclosureArticleID != "":
		steps = append(steps, nextStep{"Read the disclosure:", "nfn articles get " + e.PrimaryDisclosureArticleID})
	case len(e.ArticleIDs) > 0:
		steps = append(steps, nextStep{"Read an article:", "nfn articles get " + e.ArticleIDs[0]})
	}
	if e.RelatedTo != "" {
		steps = append(steps, nextStep{"The event it covers:", "nfn events get " + e.RelatedTo})
	} else if e.Significance != "commentary" {
		steps = append(steps, nextStep{"Commentary on it:", "nfn events list --related-to " + e.ID})
	}
	return formatNextSteps(steps)
}

// eventSummaryLine is the envelope summary for a single event, naming the
// terminal state on a tombstone since it has no title to report.
func eventSummaryLine(e *api.Event) string {
	if !e.IsTombstone() {
		return fmt.Sprintf("event: %s", e.Title)
	}
	if e.Status == "merged" && e.MergedInto != "" {
		return fmt.Sprintf("event %s: merged into %s", e.ID, e.MergedInto)
	}
	return fmt.Sprintf("event %s: %s", e.ID, e.Status)
}

// eventTombstoneNote explains a tombstone in text output, where the absent
// fields otherwise read as an event whose content went missing.
func eventTombstoneNote(e *api.Event) string {
	switch {
	case e.Status == "merged" && e.MergedInto != "":
		return fmt.Sprintf("Note: this event was merged into %s — follow it with 'nfn events get %s'.", e.MergedInto, e.MergedInto)
	case e.Status == "merged":
		return "Note: this event was merged into another event that this record does not name, as some older merges do not."
	default:
		return "Note: this event was removed and carries no content. Removals are terminal, but the id keeps resolving so a sync can see it."
	}
}

// eventCompanyLabel renders one involved company: its name, ticker, whether it
// is the primary subject, and its association tier (peripheral entries appear
// only under --include-peripheral).
func eventCompanyLabel(c api.EventCompany) string {
	label := c.Name
	if c.TickerSymbol != "" {
		label += " (" + c.TickerSymbol + ")"
	}
	var tags []string
	if c.IsPrimary {
		tags = append(tags, "primary")
	}
	if c.AssociationTier != "" {
		tags = append(tags, c.AssociationTier)
	}
	if len(tags) > 0 {
		label += " [" + strings.Join(tags, ", ") + "]"
	}
	return label
}

// eventDivergenceLines renders one line per figure the member articles disagree
// on: enough to see what disagrees and by how much, with the per-article values
// left to --format json.
func eventDivergenceLines(fd *api.FigureDivergence) []string {
	if fd == nil {
		return nil
	}
	lines := make([]string, 0, len(fd.Groups))
	for _, g := range fd.Groups {
		label := g.Concept
		if g.Period != "" {
			label += " " + g.Period
		}
		var parts []string
		if g.Min != nil && g.Max != nil {
			// "to" rather than a dash: these figures are routinely negative, and
			// -120000--100000 is unreadable.
			span := formatFigure(*g.Min) + " to " + formatFigure(*g.Max)
			if g.Currency != "" {
				span += " " + g.Currency
			}
			parts = append(parts, span)
		}
		// Values, not sources: one article can contribute more than one figure,
		// and several articles can share a publisher. Guarded like Min/Max above,
		// which the schema leaves equally optional.
		if len(g.Values) > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", len(g.Values), pluralize("value", len(g.Values))))
		}
		if len(parts) == 0 {
			continue
		}
		lines = append(lines, label+": "+strings.Join(parts, ", "))
	}
	return lines
}

// formatFigure renders a normalized figure in absolute base units without
// exponent notation or invented precision.
func formatFigure(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func init() {
	addEventFilterFlags(eventsListCmd)
	eventsListCmd.Flags().String("ticker", "", "Filter to events involving companies with these tickers (comma-separated, max 25, e.g. VOLV-B,ERIC-B); every value must resolve or the request errors. Matches active-involvement roles only unless --include-peripheral")
	eventsListCmd.Flags().String("company", "", "Filter to events involving these company IDs (comma-separated, max 25; reaches companies with no ticker); every value must resolve or the request errors. Matches active-involvement roles only unless --include-peripheral")
	eventsListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (comma-separated, e.g. XSTO,XCSE)")
	eventsListCmd.Flags().String("market", "", "Filter by listing-venue country (comma-separated ISO2, e.g. SE,NO): events involving issuers with an active equity listing on an exchange in these countries, including foreign-domiciled issuers")
	eventsListCmd.Flags().String("domicile", "", "Filter by legal domicile country (comma-separated ISO2, e.g. SE,NO): reaches unlisted issuers, excludes foreign-domiciled issuers listed locally")
	eventsListCmd.Flags().String("index", "", "Filter by stock index ID or symbol (e.g. OMXS30)")
	eventsListCmd.Flags().String("sector", "", "Filter by company sector (e.g. Industrials); must match a standard sector name exactly")
	addMarketSegmentFlag(eventsListCmd)
	eventsListCmd.Flags().Bool("listed", false, "Only events involving at least one company with an active listing; --include-peripheral does not widen this")
	eventsListCmd.Flags().String("watchlist-id", "", "Only show events for companies in this watchlist (see 'nfn watchlists list')")
	addPaginationFlags(eventsListCmd)
	addFieldsFlagFor(eventsListCmd, "event")
	addSemanticFlag(eventsListCmd)
	// --q is registered by the shared helper, but only this route offers
	// --semantic, so only this route's help may point at it.
	eventsListCmd.Flags().Lookup("q").Usage += "; add --semantic to match by meaning instead"

	eventsGetCmd.Flags().Bool("include-peripheral", false, "Also list companies with a peripheral role (advisor, speaker, passing mention)")

	addFieldsFlagFor(eventTypesCmd, "event_type")

	eventsCmd.AddCommand(eventsListCmd)
	eventsCmd.AddCommand(eventsGetCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(eventTypesCmd)
}

// addEventFilterFlags registers the filters both the flat events feed and the
// nested company events route accept. The flat feed adds the company and
// market-slice filters the nested route rejects on top of these.
func addEventFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("q", "", "Filter by full-text match over the event and its member articles, keeping feed order and cursor paging (unlike --q on articles and stories)")
	cmd.Flags().String("sort", "", "Sort order: latest (default, most recently reported first) or significance (notable, then routine, then commentary, with large ties inside each tier). significance is a best-effort snapshot, so an --all walk can miss an event that changes tier mid-walk; prefer --significance notable for a plain importance filter")
	cmd.Flags().String("event-type", "", "Filter by Events taxonomy slug (comma-separated, e.g. earnings-report,acquisition-announced); see 'nfn event-types'. The underscored calendar values are rejected")
	cmd.Flags().String("significance", "", "Filter by editorial importance (comma-separated): notable (material news), routine (filings and scheduled disclosures), commentary (opinion and analysis)")
	cmd.Flags().String("country", "", "Filter to events with a member article from these countries (comma-separated ISO2, e.g. SE,NO)")
	cmd.Flags().String("category", "", "Filter to events with a member article in these categories, by ID from 'nfn categories' (comma-separated; unknown IDs are ignored)")
	cmd.Flags().String("sources", "", "Filter to events with a member article from these source IDs (comma-separated, max 25)")
	cmd.Flags().String("source-type", "", "Filter to events with a member article from these source types (comma-separated: news_publication, wire_service, press_release, government, trade_publication, blog, stock_exchange, research)")
	cmd.Flags().String("content-type", "", "Filter to events with a member article of these content types (comma-separated: news, analysis, press_release, market_commentary, market_news, trading_halt, trading_event, other)")
	cmd.Flags().String("first-reported-after", "", "ISO 8601 datetime: only events first reported at or after this time")
	cmd.Flags().String("first-reported-before", "", "ISO 8601 datetime: only events first reported before this time")
	cmd.Flags().String("last-article-after", "", "ISO 8601 datetime: only events last active at or after this time. Mutually exclusive with --window")
	cmd.Flags().String("last-article-before", "", "ISO 8601 datetime: only events last active before this time. Mutually exclusive with --window")
	cmd.Flags().String("window", "", "Shorthand for --last-article-after relative to now (hours or days, e.g. 8h, 24h, 7d, max 365d). Mutually exclusive with --last-article-after and --last-article-before. Evaluated per request, so compute --last-article-after yourself for a stable multi-page snapshot")
	cmd.Flags().String("updated-after", "", "ISO 8601 datetime for incremental sync. Orders by update time and includes merge and removal tombstones, which no other request returns")
	cmd.Flags().String("related-to", "", "Filter to commentary events reacting to this event ID; an unresolvable ID returns no results rather than an error")
	cmd.Flags().Bool("include-peripheral", false, "Widen --ticker, --company, --watchlist-id and the returned company IDs to peripheral roles (advisor, speaker, passing mention)")
}

var eventNoun = [2]string{"event", "events"}
