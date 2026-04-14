package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

var companiesCmd = &cobra.Command{
	Use:   "companies",
	Short: "Manage companies",
}

var companiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List companies",
	Example: `  nfn companies list --limit 10
  nfn companies list --q "Volvo"
  nfn companies list --country SE --sector Financials
  nfn companies list --exchange XSTO --listed
  nfn companies list --is-active
  nfn companies list --watchlist
  nfn companies list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		if v, _ := cmd.Flags().GetString("q"); v != "" {
			params.Set("q", v)
		}
		if v, _ := cmd.Flags().GetString("country"); v != "" {
			params.Set("country", v)
		}
		if v, _ := cmd.Flags().GetString("exchange"); v != "" {
			params.Set("exchange", v)
		}
		if v, _ := cmd.Flags().GetString("sector"); v != "" {
			params.Set("sector", v)
		}
		if v, _ := cmd.Flags().GetBool("listed"); v {
			params.Set("listed", "true")
		}
		if v, _ := cmd.Flags().GetBool("watchlist"); v {
			params.Set("watchlist", "true")
		}
		if cmd.Flags().Changed("is-active") {
			v, _ := cmd.Flags().GetBool("is-active")
			if v {
				params.Set("is_active", "true")
			} else {
				params.Set("is_active", "false")
			}
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
			companies, _, err := client.ListAllCompanies(cmd.Context(), params)
			if err != nil {
				return err
			}
			return renderCompanies(f, companies, nil)
		}

		companies, pagination, _, err := client.ListCompanies(cmd.Context(), params)
		if err != nil {
			return err
		}
		return renderCompanies(f, companies, pagination)
	},
}

var companiesGetCmd = &cobra.Command{
	Use:   "get <identifier>",
	Short: "Get company details",
	Long:  "Get company details. The identifier can be a company ID or ticker symbol.",
	Example: `  nfn companies get VOLV-B
  nfn companies get 64hn2d7ul6kr --format json`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		company, _, err := client.GetCompany(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(company, fmt.Sprintf("company: %s", company.Name), breadcrumbsFor("nfn companies get"))
		}

		exchange := ""
		if company.Exchange != nil {
			exchange = fmt.Sprintf("%s (%s)", company.Exchange.Name, company.Exchange.MicCode)
		}

		active := "No"
		if company.IsActive {
			active = "Yes"
		}

		fields := []output.Field{
			{Key: "ID", Value: company.ID},
			{Key: "Name", Value: company.Name},
			{Key: "Ticker", Value: company.TickerSymbol},
			{Key: "Exchange", Value: exchange},
			{Key: "Slug", Value: company.Slug},
			{Key: "Active", Value: active},
		}
		if company.Country != nil {
			fields = append(fields, output.Field{Key: "Country", Value: fmt.Sprintf("%s (%s)", company.Country.Name, company.Country.ISO2Code)})
		}
		if company.Description != "" {
			fields = append(fields, output.Field{Key: "Description", Value: company.Description})
		}
		if company.Sector != "" {
			fields = append(fields, output.Field{Key: "Sector", Value: company.Sector})
		}
		if company.Website != "" {
			fields = append(fields, output.Field{Key: "Website", Value: company.Website})
		}
		if company.ParentCompany != nil {
			parent := company.ParentCompany.Name
			if company.ParentCompany.TickerSymbol != "" {
				parent += " (" + company.ParentCompany.TickerSymbol + ")"
			}
			fields = append(fields, output.Field{Key: "Parent Company", Value: parent})
		}
		if len(company.Subsidiaries) > 0 {
			subs := make([]string, len(company.Subsidiaries))
			for i, s := range company.Subsidiaries {
				if s.TickerSymbol != "" {
					subs[i] = s.Name + " (" + s.TickerSymbol + ")"
				} else {
					subs[i] = s.Name
				}
			}
			fields = append(fields, output.Field{Key: "Subsidiaries", Value: strings.Join(subs, ", ")})
		}
		if len(company.Indices) > 0 {
			indices := make([]string, len(company.Indices))
			for i, idx := range company.Indices {
				if idx.Symbol != "" {
					indices[i] = idx.Name + " (" + idx.Symbol + ")"
				} else {
					indices[i] = idx.Name
				}
			}
			fields = append(fields, output.Field{Key: "Indices", Value: strings.Join(indices, ", ")})
		}
		if len(company.StockListings) > 0 {
			listings := make([]string, len(company.StockListings))
			for i, l := range company.StockListings {
				primary := ""
				if l.Primary {
					primary = " [primary]"
				}
				listings[i] = fmt.Sprintf("%s @ %s%s", l.TickerSymbol, l.Exchange.MicCode, primary)
			}
			fields = append(fields, output.Field{Key: "Stock Listings", Value: strings.Join(listings, ", ")})
		}
		if company.ArticleCount > 0 {
			fields = append(fields, output.Field{Key: "Articles", Value: fmt.Sprintf("%d", company.ArticleCount)})
		}
		if company.StoryCount > 0 {
			fields = append(fields, output.Field{Key: "Stories", Value: fmt.Sprintf("%d", company.StoryCount)})
		}
		if r := company.Registry; r != nil {
			if r.Source != "" {
				fields = append(fields, output.Field{Key: "Registry Source", Value: r.Source})
			}
			if r.RegisteredName != "" {
				fields = append(fields, output.Field{Key: "Registered Name", Value: r.RegisteredName})
			}
			if r.CompanyForm != "" {
				fields = append(fields, output.Field{Key: "Company Form", Value: r.CompanyForm})
			}
			if r.City != "" {
				fields = append(fields, output.Field{Key: "City", Value: r.City})
			}
			if r.Founded != "" {
				fields = append(fields, output.Field{Key: "Founded", Value: r.Founded})
			}
			if r.Employees != nil {
				fields = append(fields, output.Field{Key: "Employees", Value: fmt.Sprintf("%d", *r.Employees)})
			}
			if r.Industry != "" {
				fields = append(fields, output.Field{Key: "Industry", Value: r.Industry})
			}
			if r.ShareCapital != nil && r.ShareCapital.Amount != nil {
				sc := fmt.Sprintf("%.0f", *r.ShareCapital.Amount)
				if r.ShareCapital.Currency != nil {
					sc += " " + *r.ShareCapital.Currency
				}
				fields = append(fields, output.Field{Key: "Share Capital", Value: sc})
			}
			if r.InstitutionalSector != "" {
				fields = append(fields, output.Field{Key: "Institutional Sector", Value: r.InstitutionalSector})
			}
			if r.Bankruptcy != nil && *r.Bankruptcy {
				fields = append(fields, output.Field{Key: "Bankruptcy", Value: "Yes"})
			}
			if r.UnderLiquidation != nil && *r.UnderLiquidation {
				fields = append(fields, output.Field{Key: "Under Liquidation", Value: "Yes"})
			}
			if r.DeregisteredAt != "" {
				fields = append(fields, output.Field{Key: "Deregistered At", Value: r.DeregisteredAt})
			}
			if r.DeregistrationReason != "" {
				fields = append(fields, output.Field{Key: "Deregistration Reason", Value: r.DeregistrationReason})
			}
		}

		f.RenderDetail(fields)
		return nil
	},
}

var companiesArticlesCmd = &cobra.Command{
	Use:   "articles <identifier>",
	Short: "List articles for a company",
	Example: `  nfn companies articles VOLV-B --limit 5
  nfn companies articles VOLV-B --primary-only`,
	Args:  cobra.ExactArgs(1),
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
		if v, _ := cmd.Flags().GetBool("primary-only"); v {
			params.Set("primary_only", "true")
		}

		all, _ := cmd.Flags().GetBool("all")
		if all {
			articles, _, err := client.ListAllCompanyArticles(cmd.Context(), args[0], params)
			if err != nil {
				return err
			}
			return renderArticles(f, articles, nil, "nfn companies articles")
		}

		articles, pagination, _, err := client.ListCompanyArticles(cmd.Context(), args[0], params)
		if err != nil {
			return err
		}
		return renderArticles(f, articles, pagination, "nfn companies articles")
	},
}

var companiesStoriesCmd = &cobra.Command{
	Use:   "stories <identifier>",
	Short: "List stories for a company",
	Example: `  nfn companies stories VOLV-B --limit 5`,
	Args:  cobra.ExactArgs(1),
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
			stories, _, err := client.ListAllCompanyStories(cmd.Context(), args[0], params)
			if err != nil {
				return err
			}
			return renderStories(f, stories, nil, "nfn companies stories")
		}

		stories, pagination, _, err := client.ListCompanyStories(cmd.Context(), args[0], params)
		if err != nil {
			return err
		}
		return renderStories(f, stories, pagination, "nfn companies stories")
	},
}

func renderCompanies(f *output.Formatter, companies []api.Company, pagination *api.Pagination, cmdPath ...string) error {
	if f.Format() == "json" {
		result := map[string]interface{}{"companies": companies}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		path := "nfn companies list"
		if len(cmdPath) > 0 {
			path = cmdPath[0]
		}
		return f.RenderEnvelope(result, fmt.Sprintf("%d companies", len(companies)), breadcrumbsFor(path))
	}

	columns := []string{"ID", "Name", "Ticker", "Exchange", "Active"}
	rows := make([][]string, len(companies))
	for i, c := range companies {
		exchange := ""
		if c.Exchange != nil {
			exchange = c.Exchange.MicCode
		}
		active := "No"
		if c.IsActive {
			active = "Yes"
		}
		rows[i] = []string{c.ID, c.Name, c.TickerSymbol, exchange, active}
	}
	f.Render(columns, rows)
	return nil
}

func init() {
	companiesListCmd.Flags().String("q", "", "Search company names and aliases")
	companiesListCmd.Flags().String("country", "", "Filter by country code (e.g. SE)")
	companiesListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (e.g. XSTO)")
	companiesListCmd.Flags().String("sector", "", "Filter by sector (e.g. Financials)")
	companiesListCmd.Flags().Bool("listed", false, "Only show listed companies")
	companiesListCmd.Flags().Bool("is-active", false, "Only show active companies")
	companiesListCmd.Flags().Bool("watchlist", false, "Only show watchlisted companies")
	addPaginationFlags(companiesListCmd)
	addFieldsFlag(companiesListCmd)

	addPaginationFlags(companiesArticlesCmd)
	companiesArticlesCmd.Flags().Bool("primary-only", false, "Only show articles where this is the primary company")

	addPaginationFlags(companiesStoriesCmd)

	companiesCmd.AddCommand(companiesListCmd)
	companiesCmd.AddCommand(companiesGetCmd)
	companiesCmd.AddCommand(companiesArticlesCmd)
	companiesCmd.AddCommand(companiesStoriesCmd)
	rootCmd.AddCommand(companiesCmd)
}
