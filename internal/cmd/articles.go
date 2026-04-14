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
	Short: "Manage articles",
}

var articlesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List articles",
	Example: `  nfn articles list --limit 5
  nfn articles list --country SE --category "Economic Policy"
  nfn articles list --ticker VOLV-B --published-after 2025-01-01
  nfn articles list --q "battery" --limit 10
  nfn articles list --sources abc123,def456
  nfn articles list --listed
  nfn articles list --watchlist
  nfn articles list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := buildArticleParams(cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			articles, _, err := client.ListAllArticles(cmd.Context(), params)
			if err != nil {
				return err
			}
			return renderArticles(f, articles, nil)
		}

		articles, pagination, _, err := client.ListArticles(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderArticles(f, articles, pagination)
	},
}

var articlesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get article details",
	Example: `  nfn articles get bvnozwnshtp8
  nfn articles get bvnozwnshtp8 --format json`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		article, _, err := client.GetArticle(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(article, fmt.Sprintf("article: %s", article.Title), breadcrumbsFor("nfn articles get"))
		}

		category := ""
		if article.Category != nil {
			category = article.Category.Name
		}
		source := ""
		if article.Source != nil {
			source = article.Source.Name
		}
		companies := make([]string, len(article.Companies))
		for i, c := range article.Companies {
			if c.TickerSymbol != "" {
				companies[i] = c.Name + " (" + c.TickerSymbol + ")"
			} else {
				companies[i] = c.Name
			}
		}

		fields := []output.Field{
			{Key: "ID", Value: article.ID},
			{Key: "Title", Value: article.Title},
			{Key: "Source Title", Value: article.SourceTitle},
			{Key: "Summary", Value: article.Summary},
			{Key: "URL", Value: article.ArticleURL},
			{Key: "Type", Value: article.ContentType},
			{Key: "Published", Value: article.PublishedAt},
			{Key: "Country", Value: article.Country},
			{Key: "Category", Value: category},
			{Key: "Source", Value: source},
			{Key: "Companies", Value: strings.Join(companies, ", ")},
		}

		if len(article.KeyPoints) > 0 {
			fields = append(fields, output.Field{Key: "Key Points", Value: strings.Join(article.KeyPoints, "\n")})
		}

		f.RenderDetail(fields)
		return nil
	},
}

func buildArticleParams(cmd *cobra.Command) url.Values {
	params := url.Values{}

	if v, _ := cmd.Flags().GetString("country"); v != "" {
		params.Set("country", v)
	}
	if v, _ := cmd.Flags().GetString("category"); v != "" {
		params.Set("category", v)
	}
	if v, _ := cmd.Flags().GetString("ticker"); v != "" {
		params.Set("ticker", v)
	}
	if v, _ := cmd.Flags().GetString("content-type"); v != "" {
		params.Set("content_type", v)
	}
	if v, _ := cmd.Flags().GetString("published-after"); v != "" {
		params.Set("published_after", v)
	}
	if v, _ := cmd.Flags().GetString("published-before"); v != "" {
		params.Set("published_before", v)
	}
	if v, _ := cmd.Flags().GetString("updated-after"); v != "" {
		params.Set("updated_after", v)
	}
	if v, _ := cmd.Flags().GetString("q"); v != "" {
		params.Set("q", v)
	}
	if v, _ := cmd.Flags().GetString("ids"); v != "" {
		params.Set("ids", v)
	}
	if v, _ := cmd.Flags().GetString("sources"); v != "" {
		params.Set("sources", v)
	}
	if v, _ := cmd.Flags().GetBool("listed"); v {
		params.Set("listed", "true")
	}
	if v, _ := cmd.Flags().GetBool("watchlist"); v {
		params.Set("watchlist", "true")
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

func renderArticles(f *output.Formatter, articles []api.Article, pagination *api.Pagination, cmdPath ...string) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"articles": articles}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		path := "nfn articles list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d articles", len(articles)), breadcrumbsFor(path))
	}

	columns := []string{"ID", "Title", "Country", "Source", "Published"}
	rows := make([][]string, len(articles))
	for i, a := range articles {
		source := ""
		if a.Source != nil {
			source = a.Source.Name
		}
		rows[i] = []string{a.ID, truncate(a.Title, 60), a.Country, source, a.PublishedAt}
	}
	f.Render(columns, rows)
	return nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func init() {
	articlesListCmd.Flags().String("country", "", "Filter by country code")
	articlesListCmd.Flags().String("category", "", "Filter by category")
	articlesListCmd.Flags().String("ticker", "", "Filter by company ticker")
	articlesListCmd.Flags().String("content-type", "", "Filter by content type")
	articlesListCmd.Flags().String("published-after", "", "Filter by published after date")
	articlesListCmd.Flags().String("published-before", "", "Filter by published before date")
	articlesListCmd.Flags().String("updated-after", "", "ISO 8601 datetime for incremental sync")
	articlesListCmd.Flags().String("q", "", "Search query")
	articlesListCmd.Flags().String("ids", "", "Comma-separated article IDs")
	articlesListCmd.Flags().String("sources", "", "Filter by source IDs (comma-separated, max 25)")
	articlesListCmd.Flags().Bool("listed", false, "Only show articles about listed companies")
	articlesListCmd.Flags().Bool("watchlist", false, "Only show articles about watchlisted companies")
	addPaginationFlags(articlesListCmd)
	addFieldsFlag(articlesListCmd)

	articlesCmd.AddCommand(articlesListCmd)
	articlesCmd.AddCommand(articlesGetCmd)
	rootCmd.AddCommand(articlesCmd)
}
