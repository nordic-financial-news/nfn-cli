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

var storiesCmd = &cobra.Command{
	Use:   "stories",
	Short: "Browse stories",
}

var storiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stories",
	Example: `  nfn stories list --limit 5
  nfn stories list --country SE,NO --category <category-id>
  nfn stories list --ticker VOLV-B
  nfn stories list --ticker VOLV-B,ERIC-B
  nfn stories list --company uejazctgchj4
  nfn stories list --exchange XSTO,XCSE --sector Industrials
  nfn stories list --market SE
  nfn stories list --domicile FI
  nfn stories list --index OMXS30
  nfn stories list --q "battery" --sort latest --all
  nfn stories list --sources abc123,def456
  nfn stories list --q "supply chain disruption" --semantic
  nfn stories list --listed
  nfn stories list --watchlist-id a1b2c3d4e5f6
  nfn stories list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		if err := validateSemantic(cmd); err != nil {
			return err
		}

		params := buildStoryParams(cmd)
		watchlistID, _ := cmd.Flags().GetString("watchlist-id")
		semantic := semanticRequested(cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			stories, resp, err := client.ListAllStories(cmd.Context(), params)
			if err != nil {
				return watchlistFilterError(err, watchlistID)
			}
			return renderStories(f, stories, nil, planLimitedFrom(resp), listOpts{order: orderNote(cmd, true)})
		}

		stories, pagination, resp, err := client.ListStories(cmd.Context(), params)
		if err != nil {
			return semanticError(watchlistFilterError(err, watchlistID), semantic)
		}
		return renderStories(f, stories, pagination, planLimitedFrom(resp), listOpts{order: orderNote(cmd, true)})
	},
}

var storiesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get story details",
	Example: `  nfn stories get w668rgglpr4w
  nfn stories get w668rgglpr4w --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		story, _, err := client.GetStory(cmd.Context(), args[0])
		if err != nil {
			return notFoundError(err, fmt.Sprintf("no story has ID %q. Find one with 'nfn stories list'", args[0]))
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(story, fmt.Sprintf("story: %s", story.Title), breadcrumbsFor("nfn stories get"))
		}

		f.RenderDetail(storyFields(story))
		if story.Content != "" {
			f.RenderProse(story.Content)
			f.Println()
		}
		for _, line := range storyNextSteps(story) {
			f.Println(line)
		}
		return nil
	},
}

// storyFields runs from what the story says, to who it is about, to what it
// was built from. The body is left to the text under the table, where it reads
// as prose instead of one tall cell.
func storyFields(s *api.Story) []output.Field {
	fields := []output.Field{
		{Key: "ID", Value: s.ID},
		{Key: "Title", Value: s.Title},
		{Key: "Summary", Value: s.Summary},
	}
	if len(s.Companies) > 0 {
		fields = append(fields, output.Field{Key: "Companies", Value: handleLines(s.Companies)})
	}
	fields = append(fields, output.Field{Key: "Published", Value: detailTime(s.PublishedAt)})
	if s.Category != nil {
		fields = append(fields, output.Field{Key: "Category", Value: s.Category.Name})
	}
	if s.Country != "" {
		fields = append(fields, output.Field{Key: "Country", Value: s.Country})
	}
	// One ID per line, so each can be copied whole, with the count in the label.
	if len(s.ArticleIDs) > 0 {
		fields = append(fields, output.Field{Key: fmt.Sprintf("Articles (%d)", len(s.ArticleIDs)), Value: strings.Join(s.ArticleIDs, "\n")})
	} else {
		fields = append(fields, output.Field{Key: "Articles", Value: strconv.Itoa(s.ArticleCount)})
	}
	if s.EventID != "" {
		fields = append(fields, output.Field{Key: "Event", Value: s.EventID})
	}
	return fields
}

func storyNextSteps(s *api.Story) []string {
	var steps []nextStep
	if s.EventID != "" {
		steps = append(steps, nextStep{"The event behind it:", "nfn events get " + s.EventID})
	}
	if len(s.ArticleIDs) > 0 {
		steps = append(steps, nextStep{"Read an article:", "nfn articles get " + s.ArticleIDs[0]})
	}
	if len(s.Companies) == 1 {
		// By ID: a ticker can resolve to more than one company.
		steps = append(steps, nextStep{"The company:", "nfn companies get " + s.Companies[0].ID})
	}
	return formatNextSteps(steps)
}

func buildStoryParams(cmd *cobra.Command) url.Values {
	params := url.Values{}
	setContentFilters(params, cmd)

	if v, _ := cmd.Flags().GetString("market"); v != "" {
		params.Set("market", v)
	}
	if v, _ := cmd.Flags().GetString("domicile"); v != "" {
		params.Set("domicile", v)
	}
	if v, _ := cmd.Flags().GetString("ticker"); v != "" {
		params.Set("ticker", v)
	}
	if v, _ := cmd.Flags().GetString("company"); v != "" {
		params.Set("company", v)
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
	if v, _ := cmd.Flags().GetBool("semantic"); v {
		params.Set("mode", "semantic")
	}
	if v, _ := cmd.Flags().GetBool("listed"); v {
		params.Set("listed", "true")
	}
	if v, _ := cmd.Flags().GetString("watchlist-id"); v != "" {
		params.Set("watchlist", v)
	}
	setPaginationParams(params, cmd)

	return params
}

func renderStories(f *output.Formatter, stories []api.Story, pagination *api.Pagination, planLimited *bool, opts listOpts) error {
	if f.Format() == "json" {
		rows, err := narrowJSON(f.Fields(), stories)
		if err != nil {
			return err
		}
		result := map[string]interface{}{"stories": rows}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		addPlanLimited(result, planLimited)
		summary := planLimitedSummary(countNoun(len(stories), storyNoun), planLimited)
		return f.RenderEnvelope(result, summary, breadcrumbsFor(opts.pathOr("nfn stories list")))
	}

	// Only a semantic search returns distances, so the column appears only when
	// the results actually carry them.
	showDistance := false
	for _, s := range stories {
		if s.Distance != nil {
			showDistance = true
			break
		}
	}

	columns := []string{"ID", "Title", "Country", "Articles", "Published"}
	if showDistance {
		columns = append(columns, "Distance")
	}
	rows := make([][]string, len(stories))
	for i, s := range stories {
		rows[i] = []string{s.ID, s.Title, s.Country, fmt.Sprintf("%d", s.ArticleCount), listTime(s.PublishedAt)}
		if showDistance {
			rows[i] = append(rows[i], formatDistance(s.Distance))
		}
	}
	colFields := [][]string{{"id"}, {"title"}, {"country"}, {"article_count"}, {"published_at"}}
	columns, rows = narrowColumns(f.Fields(), columns, rows, colFields)
	f.RenderFit(columns, rows, output.Layout{Flex: "Title", Drop: []string{"Country", "Articles"}})
	if note := noColumnNote(fieldsWithoutColumn(f.Fields(), colFields)); note != "" {
		f.Println(note)
	}
	listFooter{noun: storyNoun, getCmd: "nfn stories get", pagination: pagination, notes: opts.notes()}.print(f, len(stories))
	if isPlanLimited(planLimited) {
		f.Println(planLimitedNote)
	}
	return nil
}

func init() {
	storiesListCmd.Flags().String("country", "", "Filter by country code (comma-separated ISO2, e.g. SE,NO)")
	storiesListCmd.Flags().String("market", "", "Filter by listing-venue country (comma-separated ISO2, e.g. SE,NO): companies with an active equity listing on an exchange in these countries, including foreign-domiciled issuers")
	storiesListCmd.Flags().String("domicile", "", "Filter by legal domicile country (comma-separated ISO2, e.g. SE,NO): reaches unlisted companies, excludes foreign-domiciled issuers listed locally")
	storiesListCmd.Flags().String("category", "", "Filter by category ID from nfn categories (comma-separated for several; unknown IDs are ignored)")
	storiesListCmd.Flags().String("q", "", "Search query (relevance-ranked single page; add --sort latest for newest-first with cursor paging)")
	storiesListCmd.Flags().String("sort", "", "Sort order (e.g. latest)")
	storiesListCmd.Flags().String("published-after", "", "Filter by published after date")
	storiesListCmd.Flags().String("published-before", "", "Filter by published before date")
	storiesListCmd.Flags().String("ticker", "", "Filter by company ticker (comma-separated, max 25, e.g. VOLV-B,ERIC-B); every value must resolve or the request errors")
	storiesListCmd.Flags().String("company", "", "Filter by company ID (comma-separated, max 25; reaches unlisted companies with no ticker); every value must resolve or the request errors")
	storiesListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (comma-separated, e.g. XSTO,XCSE)")
	storiesListCmd.Flags().String("index", "", "Filter by stock index ID or symbol (e.g. OMXS30)")
	storiesListCmd.Flags().String("sector", "", "Filter by company sector (e.g. Financials)")
	storiesListCmd.Flags().String("sources", "", "Filter by source IDs (comma-separated, max 25)")
	storiesListCmd.Flags().Bool("listed", false, "Only show stories about listed companies")
	storiesListCmd.Flags().String("watchlist-id", "", "Only show stories for companies in this watchlist (see 'nfn watchlists list')")
	addPaginationFlags(storiesListCmd)
	addFieldsFlagFor(storiesListCmd, "story")
	addSemanticFlag(storiesListCmd)

	storiesCmd.AddCommand(storiesListCmd)
	storiesCmd.AddCommand(storiesGetCmd)
	rootCmd.AddCommand(storiesCmd)
}

var storyNoun = [2]string{"story", "stories"}
