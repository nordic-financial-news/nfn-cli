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
	Short: "Manage stories",
}

var storiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stories",
	Example: `  nfn stories list --limit 5
  nfn stories list --country SE --category "Economic Policy"
  nfn stories list --ticker VOLV-B
  nfn stories list --sources abc123,def456
  nfn stories list --listed
  nfn stories list --watchlist
  nfn stories list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := buildStoryParams(cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			stories, _, err := client.ListAllStories(cmd.Context(), params)
			if err != nil {
				return err
			}
			return renderStories(f, stories, nil)
		}

		stories, pagination, _, err := client.ListStories(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderStories(f, stories, pagination)
	},
}

var storiesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get story details",
	Example: `  nfn stories get abc123
  nfn stories get abc123 --format json`,
	Args:  cobra.ExactArgs(1),
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
		if story.Content != "" {
			fields = append(fields, output.Field{Key: "Content", Value: story.Content})
		}

		f.RenderDetail(fields)
		return nil
	},
}

func buildStoryParams(cmd *cobra.Command) url.Values {
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

func renderStories(f *output.Formatter, stories []api.Story, pagination *api.Pagination, cmdPath ...string) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"stories": stories}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		path := "nfn stories list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d stories", len(stories)), breadcrumbsFor(path))
	}

	columns := []string{"ID", "Title", "Country", "Articles", "Published"}
	rows := make([][]string, len(stories))
	for i, s := range stories {
		rows[i] = []string{s.ID, truncate(s.Title, 60), s.Country, fmt.Sprintf("%d", s.ArticleCount), s.PublishedAt}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	storiesListCmd.Flags().String("country", "", "Filter by country code")
	storiesListCmd.Flags().String("category", "", "Filter by category")
	storiesListCmd.Flags().String("ticker", "", "Filter by company ticker")
	storiesListCmd.Flags().String("sources", "", "Filter by source IDs (comma-separated, max 25)")
	storiesListCmd.Flags().Bool("listed", false, "Only show stories about listed companies")
	storiesListCmd.Flags().Bool("watchlist", false, "Only show stories about watchlisted companies")
	addPaginationFlags(storiesListCmd)
	addFieldsFlag(storiesListCmd)

	storiesCmd.AddCommand(storiesListCmd)
	storiesCmd.AddCommand(storiesGetCmd)
	rootCmd.AddCommand(storiesCmd)
}
