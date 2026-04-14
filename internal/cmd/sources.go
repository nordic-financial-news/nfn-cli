package cmd

import (
	"fmt"
	"net/url"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage news sources",
}

var sourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List news sources",
	Long:  "List news sources that the Nordic Financial News API indexes. Use the returned IDs with --sources on the articles and stories commands.",
	Example: `  nfn sources list
  nfn sources list --limit 10
  nfn sources list --all --format json
  nfn sources list --fields id,name,domain`,
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
		if v, _ := cmd.Flags().GetString("fields"); v != "" {
			params.Set("fields", v)
		}

		all, _ := cmd.Flags().GetBool("all")
		if all {
			sources, _, err := client.ListAllSources(cmd.Context(), params)
			if err != nil {
				return err
			}
			return renderSources(f, sources, nil)
		}

		sources, pagination, _, err := client.ListSources(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderSources(f, sources, pagination)
	},
}

func renderSources(f *output.Formatter, sources []api.Source, pagination *api.Pagination) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"sources": sources}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d sources", len(sources)), breadcrumbsFor("nfn sources list"))
	}

	columns := []string{"ID", "Name", "Domain", "Country"}
	rows := make([][]string, len(sources))
	for i, s := range sources {
		rows[i] = []string{s.ID, s.Name, s.Domain, s.Country}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	addPaginationFlags(sourcesListCmd)
	addFieldsFlag(sourcesListCmd)

	sourcesCmd.AddCommand(sourcesListCmd)
	rootCmd.AddCommand(sourcesCmd)
}
