package cmd

import (
	"fmt"
	"net/url"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var indicesCmd = &cobra.Command{
	Use:   "indices",
	Short: "Manage stock indices",
}

var indicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stock indices",
	Example: `  nfn indices list
  nfn indices list --exchange XSTO
  nfn indices list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		if v, _ := cmd.Flags().GetString("exchange"); v != "" {
			params.Set("exchange", v)
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

		all, _ := cmd.Flags().GetBool("all")
		if all {
			indices, _, err := client.ListAllIndices(cmd.Context(), params)
			if err != nil {
				return err
			}
			return renderIndices(f, indices, nil)
		}

		indices, pagination, _, err := client.ListIndices(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderIndices(f, indices, pagination)
	},
}

var indicesGetCmd = &cobra.Command{
	Use:   "get <id-or-symbol>",
	Short: "Get stock index details",
	Long:  "Returns detailed information about a stock index. Look up by ID or symbol (e.g. OMXS30).",
	Example: `  nfn indices get OMXS30
  nfn indices get sn98w2gu83sf --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		index, _, err := client.GetIndex(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(index, fmt.Sprintf("index: %s", index.Name), breadcrumbsFor("nfn indices get"))
		}

		exchange := ""
		if index.Exchange != nil {
			exchange = fmt.Sprintf("%s (%s)", index.Exchange.Name, index.Exchange.MicCode)
		}

		fields := []output.Field{
			{Key: "ID", Value: index.ID},
			{Key: "Name", Value: index.Name},
			{Key: "Symbol", Value: index.Symbol},
			{Key: "Exchange", Value: exchange},
			{Key: "Pan-Nordic", Value: fmt.Sprintf("%t", index.PanNordic)},
		}
		if index.Description != "" {
			fields = append(fields, output.Field{Key: "Description", Value: index.Description})
		}
		if index.CompaniesCount > 0 {
			fields = append(fields, output.Field{Key: "Companies", Value: fmt.Sprintf("%d", index.CompaniesCount)})
		}

		f.RenderDetail(fields)
		return nil
	},
}

var indicesCompaniesCmd = &cobra.Command{
	Use:   "companies <id-or-symbol>",
	Short: "List companies in a stock index",
	Example: `  nfn indices companies OMXS30
  nfn indices companies OMXS30 --limit 10
  nfn indices companies OMXS30 --all --format json`,
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
			companies, _, err := client.ListAllIndexCompanies(cmd.Context(), args[0], params)
			if err != nil {
				return err
			}
			return renderCompanies(f, companies, nil, "nfn indices companies")
		}

		companies, pagination, _, err := client.ListIndexCompanies(cmd.Context(), args[0], params)
		if err != nil {
			return err
		}
		return renderCompanies(f, companies, pagination, "nfn indices companies")
	},
}

func renderIndices(f *output.Formatter, indices []api.StockIndex, pagination *api.Pagination) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"indices": indices}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d indices", len(indices)), breadcrumbsFor("nfn indices list"))
	}

	columns := []string{"ID", "Name", "Symbol", "Exchange", "Pan-Nordic"}
	rows := make([][]string, len(indices))
	for i, idx := range indices {
		exchange := ""
		if idx.Exchange != nil {
			exchange = idx.Exchange.MicCode
		}
		panNordic := ""
		if idx.PanNordic {
			panNordic = "yes"
		}
		rows[i] = []string{idx.ID, idx.Name, idx.Symbol, exchange, panNordic}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	indicesListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (e.g. XSTO)")
	addPaginationFlags(indicesListCmd)
	addFieldsFlag(indicesListCmd)

	addPaginationFlags(indicesCompaniesCmd)

	indicesCmd.AddCommand(indicesListCmd)
	indicesCmd.AddCommand(indicesGetCmd)
	indicesCmd.AddCommand(indicesCompaniesCmd)
	rootCmd.AddCommand(indicesCmd)
}
