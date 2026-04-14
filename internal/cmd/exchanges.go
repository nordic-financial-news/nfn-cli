package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var exchangesCmd = &cobra.Command{
	Use:   "exchanges",
	Short: "Manage exchanges",
}

var exchangesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List exchanges",
	Example: `  nfn exchanges list
  nfn exchanges list --country SE
  nfn exchanges list --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		if v, _ := cmd.Flags().GetString("country"); v != "" {
			params.Set("country", v)
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

		exchanges, pagination, _, err := client.ListExchanges(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderExchanges(f, exchanges, pagination)
	},
}

var exchangesGetCmd = &cobra.Command{
	Use:   "get <id-or-mic>",
	Short: "Get exchange details",
	Long:  "Returns detailed information about a stock exchange. Look up by ID or MIC code (e.g. XSTO).",
	Example: `  nfn exchanges get XSTO
  nfn exchanges get 64hn2d7ul6kr --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		exchange, _, err := client.GetExchange(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(exchange, fmt.Sprintf("exchange: %s", exchange.Name), breadcrumbsFor("nfn exchanges get"))
		}

		country := ""
		if exchange.Country != nil {
			country = exchange.Country.Name
		}

		fields := []output.Field{
			{Key: "ID", Value: exchange.ID},
			{Key: "Name", Value: exchange.Name},
			{Key: "MIC Code", Value: exchange.MicCode},
			{Key: "Country", Value: country},
			{Key: "Currency", Value: exchange.Currency},
			{Key: "City", Value: exchange.City},
			{Key: "Timezone", Value: exchange.Timezone},
		}
		if exchange.Acronym != "" {
			fields = append(fields, output.Field{Key: "Acronym", Value: exchange.Acronym})
		}
		if len(exchange.Indices) > 0 {
			indices := make([]string, len(exchange.Indices))
			for i, idx := range exchange.Indices {
				if idx.Symbol != "" {
					indices[i] = fmt.Sprintf("%s (%s)", idx.Name, idx.Symbol)
				} else {
					indices[i] = idx.Name
				}
			}
			fields = append(fields, output.Field{Key: "Indices", Value: strings.Join(indices, ", ")})
		}

		f.RenderDetail(fields)
		return nil
	},
}

var exchangesCompaniesCmd = &cobra.Command{
	Use:   "companies <id-or-mic>",
	Short: "List companies on an exchange",
	Example: `  nfn exchanges companies XSTO --limit 10
  nfn exchanges companies XSTO --all`,
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

		all, _ := cmd.Flags().GetBool("all")
		if all {
			companies, _, err := client.ListAllExchangeCompanies(cmd.Context(), args[0], params)
			if err != nil {
				return err
			}
			return renderCompanies(f, companies, nil, "nfn exchanges companies")
		}

		companies, pagination, _, err := client.ListExchangeCompanies(cmd.Context(), args[0], params)
		if err != nil {
			return err
		}
		return renderCompanies(f, companies, pagination, "nfn exchanges companies")
	},
}

func renderExchanges(f *output.Formatter, exchanges []api.ExchangeDetail, pagination *api.Pagination) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"exchanges": exchanges}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d exchanges", len(exchanges)), breadcrumbsFor("nfn exchanges list"))
	}

	columns := []string{"ID", "Name", "MIC Code", "Country"}
	rows := make([][]string, len(exchanges))
	for i, e := range exchanges {
		country := ""
		if e.Country != nil {
			country = e.Country.Name
		}
		rows[i] = []string{e.ID, e.Name, e.MicCode, country}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	exchangesListCmd.Flags().String("country", "", "Filter by country ISO2 code (e.g. SE)")
	exchangesListCmd.Flags().Int("limit", 0, "Maximum number of results per page")
	exchangesListCmd.Flags().String("cursor", "", "Pagination cursor")
	addFieldsFlag(exchangesListCmd)

	addPaginationFlags(exchangesCompaniesCmd)

	exchangesCmd.AddCommand(exchangesListCmd)
	exchangesCmd.AddCommand(exchangesGetCmd)
	exchangesCmd.AddCommand(exchangesCompaniesCmd)
	rootCmd.AddCommand(exchangesCmd)
}
