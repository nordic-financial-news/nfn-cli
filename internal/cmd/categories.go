package cmd

import (
	"fmt"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List article categories",
	Long:  "List all available article categories. Use category names to filter articles and stories.",
	Example: `  nfn categories
  nfn categories --format json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		categories, pagination, _, err := client.ListCategories(cmd.Context(), nil)
		if err != nil {
			return err
		}
		return renderCategories(f, categories, pagination)
	},
}

func renderCategories(f *output.Formatter, categories []api.CategoryDetail, pagination *api.Pagination) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"categories": categories}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d categories", len(categories)), breadcrumbsFor("nfn categories"))
	}

	columns := []string{"ID", "Name"}
	rows := make([][]string, len(categories))
	for i, c := range categories {
		rows[i] = []string{c.ID, c.Name}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	rootCmd.AddCommand(categoriesCmd)
}
