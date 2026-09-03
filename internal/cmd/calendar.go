package cmd

import (
	"fmt"
	"net/url"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Manage calendar events",
}

var calendarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Example: `  nfn calendar list --limit 10
  nfn calendar list --ticker VOLV-B
  nfn calendar list --company cmpny1234567
  nfn calendar list --event-type earnings_report,dividend
  nfn calendar list --country SE,NO --scheduled-before 2026-12-31
  nfn calendar list --market SE --event-type earnings_report
  nfn calendar list --domicile FI
  nfn calendar list --exchange XSTO,XCSE
  nfn calendar list --watchlist-id a1b2c3d4e5f6
  nfn calendar list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := buildCalendarParams(cmd)
		watchlistID, _ := cmd.Flags().GetString("watchlist-id")

		all, _ := cmd.Flags().GetBool("all")
		if all {
			events, resp, err := client.ListAllCalendarEvents(cmd.Context(), params)
			if err != nil {
				return watchlistFilterError(err, watchlistID)
			}
			return renderCalendarEvents(f, events, nil, planLimitedFrom(resp))
		}

		events, pagination, resp, err := client.ListCalendarEvents(cmd.Context(), params)
		if err != nil {
			return watchlistFilterError(err, watchlistID)
		}
		return renderCalendarEvents(f, events, pagination, planLimitedFrom(resp))
	},
}

var calendarGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get calendar event details",
	Example: `  nfn calendar get cale_q3volvo1
  nfn calendar get cale_q3volvo1 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		event, _, err := client.GetCalendarEvent(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(event, fmt.Sprintf("calendar event: %s", event.Title), breadcrumbsFor("nfn calendar get"))
		}

		fields := []output.Field{
			{Key: "ID", Value: event.ID},
			{Key: "Title", Value: event.Title},
			{Key: "Description", Value: event.Description},
			{Key: "Type", Value: event.EventType},
			{Key: "Status", Value: event.Status},
			{Key: "Date Precision", Value: event.DatePrecision},
			{Key: "Fiscal Period", Value: event.FiscalPeriod},
			{Key: "Scheduled", Value: event.LocalTime},
			{Key: "Timezone", Value: event.Timezone},
			{Key: "Company", Value: event.CompanyID},
			{Key: "Country", Value: event.Country},
			{Key: "Source Article", Value: event.SourceArticleID},
			{Key: "Updated", Value: event.UpdatedAt},
		}

		if event.Dividend != nil {
			d := event.Dividend
			fields = append(fields,
				output.Field{Key: "Ex Date", Value: d.ExDate},
				output.Field{Key: "Record Date", Value: d.RecordDate},
				output.Field{Key: "Payment Date", Value: d.PaymentDate},
				output.Field{Key: "Declaration Date", Value: d.DeclarationDate},
				output.Field{Key: "Dividend Kind", Value: d.Kind},
			)
		}

		f.RenderDetail(fields)
		return nil
	},
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

func renderCalendarEvents(f *output.Formatter, events []api.CalendarEvent, pagination *api.Pagination, planLimited *bool, cmdPath ...string) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"calendar_events": events}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		addPlanLimited(result, planLimited)
		path := "nfn calendar list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		summary := planLimitedSummary(fmt.Sprintf("%d calendar events", len(events)), planLimited)
		return f.RenderEnvelope(result, summary, breadcrumbsFor(path))
	}

	columns := []string{"ID", "Date", "Type", "Title", "Company", "Status"}
	rows := make([][]string, len(events))
	for i, e := range events {
		rows[i] = []string{e.ID, eventDate(e), e.EventType, truncate(e.Title, 60), e.CompanyID, e.Status}
	}
	f.Render(columns, rows)
	if isPlanLimited(planLimited) {
		f.Println(planLimitedNote)
	}
	return nil
}

// eventDate returns the local calendar date (YYYY-MM-DD) for an event. The API
// stores scheduled_at in UTC, so for Nordic events stored at local midnight the
// UTC value can fall on the previous day — local_time carries the correct date.
func eventDate(e api.CalendarEvent) string {
	s := e.LocalTime
	if s == "" {
		s = e.ScheduledAt
	}
	if len(s) < 10 {
		return s
	}
	return s[:10]
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
	calendarListCmd.Flags().String("event-type", "", "Filter by event type (comma-separated, e.g. earnings_report,dividend)")
	calendarListCmd.Flags().String("status", "", "Filter by status (scheduled, published, cancelled)")
	calendarListCmd.Flags().String("scheduled-after", "", "Only events scheduled on or after this date (ISO 8601)")
	calendarListCmd.Flags().String("scheduled-before", "", "Only events scheduled before this date (ISO 8601)")
	calendarListCmd.Flags().String("updated-after", "", "ISO 8601 datetime for incremental sync")
	calendarListCmd.Flags().String("watchlist-id", "", "Only show events for companies in this watchlist (see 'nfn watchlists list')")
	addPaginationFlags(calendarListCmd)
	addFieldsFlag(calendarListCmd)

	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarGetCmd)
	rootCmd.AddCommand(calendarCmd)
}
