package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var articlesCmd = &cobra.Command{
	Use:   "articles",
	Short: "Browse articles",
}

var articlesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List articles",
	Long: `List articles.

Note: article title, summary, and key_points are machine-translated and
AI-paraphrased from the source article. They are not verbatim quotations and
are not human-verified.`,
	Example: `  nfn articles list --limit 5
  nfn articles list --country SE,NO --category <category-id>
  nfn articles list --ticker VOLV-B --published-after 2025-01-01
  nfn articles list --ticker VOLV-B,ERIC-B
  nfn articles list --ticker VOLV-B --primary-only
  nfn articles list --company uejazctgchj4
  nfn articles list --exchange XSTO,XCSE --sector Industrials
  nfn articles list --market SE
  nfn articles list --domicile FI
  nfn articles list --index OMXS30
  nfn articles list --q "battery" --limit 10
  nfn articles list --q "battery" --sort latest --all
  nfn articles list --q "defence contract wins and capacity expansion" --semantic
  nfn articles list --sources abc123,def456
  nfn articles list --source-type stock_exchange,wire_service
  nfn articles list --listed
  nfn articles list --watchlist-id a1b2c3d4e5f6
  nfn articles list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		if err := validateSemantic(cmd); err != nil {
			return err
		}

		params := buildArticleParams(cmd)
		watchlistID, _ := cmd.Flags().GetString("watchlist-id")
		semantic := semanticRequested(cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			articles, resp, err := client.ListAllArticles(cmd.Context(), params)
			if err != nil {
				return watchlistFilterError(err, watchlistID)
			}
			return renderArticles(f, articles, nil, planLimitedFrom(resp), listOpts{order: orderNote(cmd, true)})
		}

		articles, pagination, resp, err := client.ListArticles(cmd.Context(), params)
		if err != nil {
			return semanticError(watchlistFilterError(err, watchlistID), semantic)
		}
		return renderArticles(f, articles, pagination, planLimitedFrom(resp), listOpts{order: orderNote(cmd, true)})
	},
}

var articlesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get article details",
	Long: `Get article details.

Note: title, summary, and key_points are machine-translated and AI-paraphrased
from the source article. They are not verbatim quotations and are not
human-verified. Use the article_url / URL field to reach the original source.`,
	Example: `  nfn articles get bvnozwnshtp8
  nfn articles get bvnozwnshtp8 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		article, _, err := client.GetArticle(cmd.Context(), args[0])
		if err != nil {
			return notFoundError(err, fmt.Sprintf("no article has ID %q. Find one with 'nfn articles list'", args[0]))
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(article, fmt.Sprintf("article: %s", article.Title), breadcrumbsFor("nfn articles get"))
		}

		f.RenderDetail(articleFields(article))
		for _, line := range articleNextSteps(article) {
			f.Println(line)
		}
		return nil
	},
}

// articleFields runs from what the article says, to who it is about, to where
// it came from. The URL is left to the lines under the table: wrapped inside a
// cell it would break mid-token and no longer open.
func articleFields(a *api.Article) []output.Field {
	fields := []output.Field{
		{Key: "ID", Value: a.ID},
		{Key: "Title", Value: a.Title},
		{Key: "Summary", Value: a.Summary},
	}
	if len(a.KeyPoints) > 0 {
		points := make([]string, len(a.KeyPoints))
		for i, p := range a.KeyPoints {
			points[i] = output.Bullet + p
		}
		fields = append(fields, output.Field{Key: "Key Points", Value: strings.Join(points, "\n")})
	}
	if len(a.Companies) > 0 {
		fields = append(fields, output.Field{Key: "Companies", Value: handleLines(a.Companies)})
	}
	fields = append(fields, output.Field{Key: "Published", Value: detailTime(a.PublishedAt)})
	if a.Source != nil {
		source := a.Source.Name
		if a.Source.SourceType != "" {
			source += " (" + humanize(a.Source.SourceType) + ")"
		}
		fields = append(fields, output.Field{Key: "Source", Value: source})
	}
	if a.ContentType != "" {
		fields = append(fields, output.Field{Key: "Type", Value: humanize(a.ContentType)})
	}
	if a.Category != nil {
		fields = append(fields, output.Field{Key: "Category", Value: a.Category.Name})
	}
	if a.Country != "" {
		fields = append(fields, output.Field{Key: "Country", Value: a.Country})
	}
	// The headline as published, before translation.
	if a.SourceTitle != "" && a.SourceTitle != a.Title {
		fields = append(fields, output.Field{Key: "Original Title", Value: a.SourceTitle})
	}
	if a.EventID != "" {
		fields = append(fields, output.Field{Key: "Event", Value: a.EventID})
	}
	return fields
}

func articleNextSteps(a *api.Article) []string {
	var steps []nextStep
	if a.ArticleURL != "" {
		steps = append(steps, nextStep{"Read the original:", a.ArticleURL})
	}
	if a.EventID != "" {
		steps = append(steps, nextStep{"The event it reports:", "nfn events get " + a.EventID})
	}
	return formatNextSteps(steps)
}

func buildArticleParams(cmd *cobra.Command) url.Values {
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
	if v, _ := cmd.Flags().GetString("ids"); v != "" {
		params.Set("ids", v)
	}
	if v, _ := cmd.Flags().GetBool("listed"); v {
		params.Set("listed", "true")
	}
	if v, _ := cmd.Flags().GetString("watchlist-id"); v != "" {
		params.Set("watchlist", v)
	}
	if v, _ := cmd.Flags().GetBool("primary-only"); v {
		params.Set("primary_only", "true")
	}
	setPaginationParams(params, cmd)

	return params
}

func renderArticles(f *output.Formatter, articles []api.Article, pagination *api.Pagination, planLimited *bool, opts listOpts) error {
	if f.Format() == "json" {
		rows, err := narrowJSON(f.Fields(), articles)
		if err != nil {
			return err
		}
		result := map[string]interface{}{"articles": rows}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		addPlanLimited(result, planLimited)
		summary := planLimitedSummary(countNoun(len(articles), articleNoun), planLimited)
		return f.RenderEnvelope(result, summary, breadcrumbsFor(opts.pathOr("nfn articles list")))
	}

	// Only a semantic search returns distances, so the column appears only when
	// the results actually carry them.
	showDistance := false
	for _, a := range articles {
		if a.Distance != nil {
			showDistance = true
			break
		}
	}

	columns := []string{"ID", "Title", "Country", "Source", "Published"}
	if showDistance {
		columns = append(columns, "Distance")
	}
	rows := make([][]string, len(articles))
	for i, a := range articles {
		source := ""
		if a.Source != nil {
			source = a.Source.Name
		}
		rows[i] = []string{a.ID, a.Title, a.Country, source, listTime(a.PublishedAt)}
		if showDistance {
			rows[i] = append(rows[i], formatDistance(a.Distance))
		}
	}
	colFields := [][]string{{"id"}, {"title"}, {"country"}, {"source"}, {"published_at"}}
	columns, rows = narrowColumns(f.Fields(), columns, rows, colFields)
	f.RenderFit(columns, rows, output.Layout{Flex: "Title", Drop: []string{"Country", "Source"}})
	if note := noColumnNote(fieldsWithoutColumn(f.Fields(), colFields)); note != "" {
		f.Println(note)
	}
	listFooter{noun: articleNoun, getCmd: "nfn articles get", pagination: pagination, notes: opts.notes()}.print(f, len(articles))
	if isPlanLimited(planLimited) {
		f.Println(planLimitedNote)
	}
	return nil
}

func init() {
	articlesListCmd.Flags().String("country", "", "Filter by country code (comma-separated ISO2, e.g. SE,NO)")
	articlesListCmd.Flags().String("market", "", "Filter by listing-venue country (comma-separated ISO2, e.g. SE,NO): companies with an active equity listing on an exchange in these countries, including foreign-domiciled issuers")
	articlesListCmd.Flags().String("domicile", "", "Filter by legal domicile country (comma-separated ISO2, e.g. SE,NO): reaches unlisted companies, excludes foreign-domiciled issuers listed locally")
	articlesListCmd.Flags().String("category", "", "Filter by category ID from nfn categories (comma-separated for several; unknown IDs are ignored)")
	articlesListCmd.Flags().String("ticker", "", "Filter by company ticker (comma-separated, max 25, e.g. VOLV-B,ERIC-B); every value must resolve or the request errors")
	articlesListCmd.Flags().String("company", "", "Filter by company ID (comma-separated, max 25; reaches unlisted companies with no ticker); every value must resolve or the request errors")
	articlesListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (comma-separated, e.g. XSTO,XCSE)")
	articlesListCmd.Flags().String("index", "", "Filter by stock index ID or symbol (e.g. OMXS30)")
	articlesListCmd.Flags().String("sector", "", "Filter by company sector (e.g. Financials)")
	articlesListCmd.Flags().String("content-type", "", "Filter by content type")
	articlesListCmd.Flags().String("published-after", "", "Filter by published after date")
	articlesListCmd.Flags().String("published-before", "", "Filter by published before date")
	articlesListCmd.Flags().String("updated-after", "", "ISO 8601 datetime for incremental sync")
	articlesListCmd.Flags().String("q", "", "Search query (relevance-ranked single page; add --sort latest for newest-first with cursor paging)")
	articlesListCmd.Flags().String("sort", "", "Sort order (e.g. latest)")
	articlesListCmd.Flags().String("ids", "", "Comma-separated article IDs")
	articlesListCmd.Flags().String("sources", "", "Filter by source IDs (comma-separated, max 25)")
	articlesListCmd.Flags().String("source-type", "", "Filter by source type (comma-separated: news_publication, wire_service, press_release, government, trade_publication, blog, stock_exchange, research)")
	articlesListCmd.Flags().Bool("listed", false, "Only show articles about listed companies")
	articlesListCmd.Flags().String("watchlist-id", "", "Only show articles for companies in this watchlist (see 'nfn watchlists list')")
	articlesListCmd.Flags().Bool("primary-only", false, "Only articles where the matched company is the primary subject, not a passing mention (requires --ticker, --company, or --watchlist-id)")
	addPaginationFlags(articlesListCmd)
	addFieldsFlagFor(articlesListCmd, "article")
	addSemanticFlag(articlesListCmd)

	articlesCmd.AddCommand(articlesListCmd)
	articlesCmd.AddCommand(articlesGetCmd)
	rootCmd.AddCommand(articlesCmd)
}

var articleNoun = [2]string{"article", "articles"}
