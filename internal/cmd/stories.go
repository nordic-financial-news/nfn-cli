package cmd

import (
	"fmt"
	"net/url"
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
  nfn stories list --company cmpny1234567
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
			return renderStories(f, stories, nil, planLimitedFrom(resp))
		}

		stories, pagination, resp, err := client.ListStories(cmd.Context(), params)
		if err != nil {
			return semanticError(watchlistFilterError(err, watchlistID), semantic)
		}
		return renderStories(f, stories, pagination, planLimitedFrom(resp))
	},
}

var storiesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get story details",
	Example: `  nfn stories get abc123
  nfn stories get abc123 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		story, _, err := client.GetStory(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(story, fmt.Sprintf("story: %s", story.Title), breadcrumbsFor("nfn stories get"))
		}

		category := ""
		if story.Category != nil {
			category = story.Category.Name
		}
		companies := make([]string, len(story.Companies))
		for i, c := range story.Companies {
			if c.TickerSymbol != "" {
				companies[i] = c.Name + " (" + c.TickerSymbol + ")"
			} else {
				companies[i] = c.Name
			}
		}

		fields := []output.Field{
			{Key: "ID", Value: story.ID},
			{Key: "Title", Value: story.Title},
			{Key: "Summary", Value: story.Summary},
			{Key: "Published", Value: story.PublishedAt},
			{Key: "Country", Value: story.Country},
			{Key: "Category", Value: category},
			{Key: "Articles", Value: strings.Join(story.ArticleIDs, ", ")},
			{Key: "Companies", Value: strings.Join(companies, ", ")},
		}
		if story.EventID != "" {
			fields = append(fields, output.Field{Key: "Event", Value: story.EventID})
		}
		if story.Content != "" {
			fields = append(fields, output.Field{Key: "Content", Value: story.Content})
		}

		f.RenderDetail(fields)
		return nil
	},
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

func renderStories(f *output.Formatter, stories []api.Story, pagination *api.Pagination, planLimited *bool, cmdPath ...string) error {
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
		path := "nfn stories list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		summary := planLimitedSummary(fmt.Sprintf("%d stories", len(stories)), planLimited)
		return f.RenderEnvelope(result, summary, breadcrumbsFor(path))
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
	listFooter{noun: [2]string{"story", "stories"}, getCmd: "nfn stories get", pagination: pagination}.print(f, len(stories))
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
