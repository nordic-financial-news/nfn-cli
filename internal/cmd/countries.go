package cmd

import (
	"fmt"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var countriesCmd = &cobra.Command{
	Use:   "countries",
	Short: "Manage supported countries",
}

var countriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all supported countries",
	Long:  "List all supported Nordic countries. Use ISO2 codes or slugs with other commands.",
	Example: `  nfn countries list
  nfn countries list --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		countries, pagination, _, err := client.ListCountries(cmd.Context(), nil)
		if err != nil {
			return err
		}
		return renderCountries(f, countries, pagination)
	},
}

var countriesGetCmd = &cobra.Command{
	Use:   "get <id-or-code>",
	Short: "Get country details",
	Long:  "Returns details for a country, including its exchanges. Look up by ID, ISO2 code (e.g. SE), or slug (e.g. sweden).",
	Example: `  nfn countries get SE
  nfn countries get sweden`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		country, _, err := client.GetCountry(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(country, fmt.Sprintf("country: %s", country.Name), breadcrumbsFor("nfn countries get"))
		}

		fields := []output.Field{
			{Key: "ID", Value: country.ID},
			{Key: "Name", Value: country.Name},
			{Key: "ISO2 Code", Value: country.ISO2Code},
		}
		if len(country.Exchanges) > 0 {
			exchanges := make([]string, len(country.Exchanges))
			for i, e := range country.Exchanges {
				exchanges[i] = fmt.Sprintf("%s (%s)", e.Name, e.MicCode)
			}
			fields = append(fields, output.Field{Key: "Exchanges", Value: strings.Join(exchanges, ", ")})
		}
		f.RenderDetail(fields)
		return nil
	},
}

func renderCountries(f *output.Formatter, countries []api.CountryDetail, pagination *api.Pagination) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"countries": countries}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d countries", len(countries)), breadcrumbsFor("nfn countries list"))
	}

	columns := []string{"Code", "Name"}
	rows := make([][]string, len(countries))
	for i, c := range countries {
		rows[i] = []string{c.ISO2Code, c.Name}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	countriesCmd.AddCommand(countriesListCmd)
	countriesCmd.AddCommand(countriesGetCmd)
	rootCmd.AddCommand(countriesCmd)
}
