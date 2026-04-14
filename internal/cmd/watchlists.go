package cmd

import (
	"fmt"
	"net/url"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var watchlistsCmd = &cobra.Command{
	Use:   "watchlists",
	Short: "Manage watchlists",
}

var watchlistsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your watchlists",
	Example: `  nfn watchlists list
  nfn watchlists list --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		watchlists, _, err := client.ListWatchlists(cmd.Context())
		if err != nil {
			return err
		}
		return renderWatchlists(f, watchlists)
	},
}

var watchlistsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get watchlist details and companies",
	Example: `  nfn watchlists get abc123
  nfn watchlists get abc123 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
			params.Set("limit", fmt.Sprintf("%d", v))
		}
		if v, _ := cmd.Flags().GetString("cursor"); v != "" {
			params.Set("cursor", v)
		}

		watchlist, companies, pagination, _, err := client.GetWatchlist(cmd.Context(), args[0], params)
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			result := map[string]interface{}{
				"watchlist":  watchlist,
				"companies":  companies,
				"pagination": paginationJSON(pagination),
			}
			return f.RenderEnvelope(result, fmt.Sprintf("watchlist: %s", watchlist.Name), breadcrumbsFor("nfn watchlists get"))
		}

		fields := []output.Field{
			{Key: "ID", Value: watchlist.ID},
			{Key: "Name", Value: watchlist.Name},
			{Key: "Companies", Value: fmt.Sprintf("%d", watchlist.CompanyCount)},
		}
		f.RenderDetail(fields)

		if len(companies) > 0 {
			f.Println()
			return renderCompanies(f, companies, nil)
		}

		return nil
	},
}

func renderWatchlists(f *output.Formatter, watchlists []api.Watchlist) error {
	if f.Format() == "json" {
		return f.RenderEnvelope(map[string]interface{}{"watchlists": watchlists}, fmt.Sprintf("%d watchlists", len(watchlists)), breadcrumbsFor("nfn watchlists list"))
	}

	columns := []string{"ID", "Name", "Companies"}
	rows := make([][]string, len(watchlists))
	for i, w := range watchlists {
		rows[i] = []string{w.ID, w.Name, fmt.Sprintf("%d", w.CompanyCount)}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	addPaginationFlags(watchlistsGetCmd)

	watchlistsCmd.AddCommand(watchlistsListCmd)
	watchlistsCmd.AddCommand(watchlistsGetCmd)
	rootCmd.AddCommand(watchlistsCmd)
}
