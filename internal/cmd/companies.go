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
	Short: "Browse companies",
}

var companiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List companies",
	Example: `  nfn companies list --limit 10
  nfn companies list --q "Volvo"
  nfn companies list --ticker VOLV-B,ERIC-B
  nfn companies list --company 64hn2d7ul6kr,7f3k9m2p1q8r
  nfn companies list --q "offshore wind turbine installation" --semantic
  nfn companies list --country SE --sector Financials
  nfn companies list --exchange XSTO,XCSE --listed
  nfn companies list --market SE
  nfn companies list --domicile FI
  nfn companies list --is-active
  nfn companies list --watchlist
  nfn companies list --all --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		if err := validateSemantic(cmd); err != nil {
			return err
		}

		params := buildCompanyParams(cmd)
		semantic := semanticRequested(cmd)
		opts := listOpts{order: orderNote(cmd, false)}
		if unfilteredCompanyList(params) {
			opts.order = "alphabetical; narrow with --q, --listed or --market"
		}

		all, _ := cmd.Flags().GetBool("all")
		if all {
			companies, resp, err := client.ListAllCompanies(cmd.Context(), params)
			if err != nil {
				return err
			}
			return renderCompanies(f, companies, nil, planLimitedFrom(resp), opts)
		}

		companies, pagination, resp, err := client.ListCompanies(cmd.Context(), params)
		if err != nil {
			return semanticError(err, semantic)
		}
		return renderCompanies(f, companies, pagination, planLimitedFrom(resp), opts)
	},
}

var companiesGetCmd = &cobra.Command{
	Use:   "get <identifier>",
	Short: "Get company details",
	Long:  "Get company details. The identifier can be a company ID or ticker symbol.",
	Example: `  nfn companies get VOLV-B
  nfn companies get 64hn2d7ul6kr --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		company, _, err := client.GetCompany(cmd.Context(), args[0])
		if err != nil {
			return companyNotFoundError(err, args[0])
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(company, fmt.Sprintf("company: %s", company.Name), breadcrumbsFor("nfn companies get"))
		}

		// Rows run from which company this is and where it trades, to what it
		// does and how it is related, to the registry record. The slug is left
		// to JSON: it is a URL key, not something a reader looks a company up by.
		listings := companyListings(company)
		listingKey := "Listing"
		if len(listings) > 1 {
			listingKey = "Listings"
		}
		fields := []output.Field{
			{Key: "ID", Value: company.ID},
			{Key: "Name", Value: company.Name},
			{Key: listingKey, Value: strings.Join(listings, "\n")},
		}
		if len(company.FormerTickers) > 0 {
			fields = append(fields, output.Field{Key: "Former Tickers", Value: strings.Join(company.FormerTickers, ", ")})
		}
		if !company.IsActive {
			fields = append(fields, output.Field{Key: "Status", Value: "Inactive"})
		}
		if company.Country != nil {
			fields = append(fields, output.Field{Key: "Country", Value: fmt.Sprintf("%s (%s)", company.Country.Name, company.Country.ISO2Code)})
		}
		if company.Sector != "" {
			fields = append(fields, output.Field{Key: "Sector", Value: company.Sector})
		}
		if company.Description != "" {
			fields = append(fields, output.Field{Key: "Description", Value: company.Description})
		}
		if company.Website != "" {
			fields = append(fields, output.Field{Key: "Website", Value: company.Website})
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
			fields = append(fields, output.Field{Key: "Indices", Value: strings.Join(indices, "\n")})
		}
		// Related companies lead with a handle 'nfn companies get' takes, since
		// most subsidiaries have no ticker to look them up by.
		if company.ParentCompany != nil {
			fields = append(fields, output.Field{Key: "Parent Company", Value: handleLines([]api.Company{*company.ParentCompany})})
		}
		if len(company.Subsidiaries) > 0 {
			fields = append(fields, output.Field{Key: "Subsidiaries", Value: handleLines(company.Subsidiaries)})
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
		for _, line := range companyNextSteps(company) {
			f.Println(line)
		}
		return nil
	},
}

// companyListings is one line per listing, naming the venue in full where the
// payload carries it and marking the primary one when there is a choice.
func companyListings(c *api.Company) []string {
	if len(c.StockListings) > 0 {
		lines := make([]string, len(c.StockListings))
		for i, l := range c.StockListings {
			lines[i] = l.TickerSymbol + " · " + exchangeLabel(&l.Exchange)
			if l.Primary && len(c.StockListings) > 1 {
				lines[i] += " · primary"
			}
		}
		return lines
	}
	if c.TickerSymbol == "" {
		return []string{"Not listed"}
	}
	lines := []string{c.TickerSymbol}
	if c.Exchange != nil {
		lines[0] += " · " + exchangeLabel(c.Exchange)
	}
	for _, l := range c.AlsoListedOn {
		lines = append(lines, l.TickerSymbol+" · "+l.MicCode)
	}
	return lines
}

func exchangeLabel(e *api.Exchange) string {
	if e.Name == "" {
		return e.MicCode
	}
	return fmt.Sprintf("%s (%s)", e.Name, e.MicCode)
}

// companyNextSteps points at the feeds a company page is the way into, with
// the counts the detail payload carries.
func companyNextSteps(c *api.Company) []string {
	handle := companyHandle(*c)
	steps := []nextStep{{"Events:", "nfn companies events " + handle}}
	if c.ArticleCount > 0 {
		steps = append(steps, nextStep{countNoun(c.ArticleCount, articleNoun) + ":", "nfn companies articles " + handle})
	}
	if c.StoryCount > 0 {
		steps = append(steps, nextStep{countNoun(c.StoryCount, storyNoun) + ":", "nfn companies stories " + handle})
	}
	if c.TickerSymbol != "" {
		steps = append(steps, nextStep{"Scheduled dates:", "nfn calendar list --ticker " + c.TickerSymbol})
	}
	return formatNextSteps(steps)
}

var companiesArticlesCmd = &cobra.Command{
	Use:   "articles <identifier>",
	Short: "List articles for a company",
	Example: `  nfn companies articles VOLV-B --limit 5
  nfn companies articles VOLV-B --primary-only
  nfn companies articles VOLV-B --category <category-id> --published-after 2026-01-01
  nfn companies articles VOLV-B --q "battery" --sort latest --all`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		setContentFilters(params, cmd)
		setPaginationParams(params, cmd)
		if v, _ := cmd.Flags().GetBool("primary-only"); v {
			params.Set("primary_only", "true")
		}

		all, _ := cmd.Flags().GetBool("all")
		if all {
			articles, resp, err := client.ListAllCompanyArticles(cmd.Context(), args[0], params)
			if err != nil {
				return companyNotFoundError(err, args[0])
			}
			return renderArticles(f, articles, nil, planLimitedFrom(resp), listOpts{path: "nfn companies articles", order: orderNote(cmd, true)})
		}

		articles, pagination, resp, err := client.ListCompanyArticles(cmd.Context(), args[0], params)
		if err != nil {
			return companyNotFoundError(err, args[0])
		}
		return renderArticles(f, articles, pagination, planLimitedFrom(resp), listOpts{path: "nfn companies articles", order: orderNote(cmd, true)})
	},
}

var companiesStoriesCmd = &cobra.Command{
	Use:   "stories <identifier>",
	Short: "List stories for a company",
	Example: `  nfn companies stories VOLV-B --limit 5
  nfn companies stories VOLV-B --category <category-id> --published-after 2026-01-01
  nfn companies stories VOLV-B --q "battery" --sort latest --all`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		setContentFilters(params, cmd)
		setPaginationParams(params, cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			stories, resp, err := client.ListAllCompanyStories(cmd.Context(), args[0], params)
			if err != nil {
				return companyNotFoundError(err, args[0])
			}
			return renderStories(f, stories, nil, planLimitedFrom(resp), listOpts{path: "nfn companies stories", order: orderNote(cmd, true)})
		}

		stories, pagination, resp, err := client.ListCompanyStories(cmd.Context(), args[0], params)
		if err != nil {
			return companyNotFoundError(err, args[0])
		}
		return renderStories(f, stories, pagination, planLimitedFrom(resp), listOpts{path: "nfn companies stories", order: orderNote(cmd, true)})
	},
}

var companiesEventsCmd = &cobra.Command{
	Use:   "events <identifier>",
	Short: "List events for a company",
	Long: `List events for a company, most recently reported first.

The path fixes the company, so the filters that name a different company or
listing — --ticker, --company, --watchlist-id, --listed, --exchange, --market,
--domicile, --index and --sector — are not offered here. Use 'nfn events list'
with --company to filter the full collection by any of them, and to search one
company's events by meaning with --semantic.`,
	Example: `  nfn companies events VOLV-B --limit 5
  nfn companies events VOLV-B --significance notable
  nfn companies events VOLV-B --event-type earnings-report
  nfn companies events VOLV-B --window 30d
  nfn companies events 64hn2d7ul6kr --all --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		if err := validateEventType(cmd); err != nil {
			return err
		}

		params := buildEventParams(cmd)

		all, _ := cmd.Flags().GetBool("all")
		if all {
			events, resp, err := client.ListAllCompanyEvents(cmd.Context(), args[0], params)
			if err != nil {
				return eventTypeError(companyNotFoundError(err, args[0]))
			}
			return renderEvents(f, events, nil, planLimitedFrom(resp), truncatedFrom(resp), "nfn companies events")
		}

		events, pagination, resp, err := client.ListCompanyEvents(cmd.Context(), args[0], params)
		if err != nil {
			return eventTypeError(companyNotFoundError(err, args[0]))
		}
		return renderEvents(f, events, pagination, planLimitedFrom(resp), truncatedFrom(resp), "nfn companies events")
	},
}

// unfilteredCompanyList reports a request for every company, which the API
// answers alphabetically across listed and private companies alike.
func unfilteredCompanyList(params url.Values) bool {
	for key := range params {
		switch key {
		case "limit", "cursor", "fields":
		default:
			return false
		}
	}
	return true
}

func buildCompanyParams(cmd *cobra.Command) url.Values {
	params := url.Values{}

	if v, _ := cmd.Flags().GetString("q"); v != "" {
		params.Set("q", v)
	}
	if v, _ := cmd.Flags().GetString("ticker"); v != "" {
		params.Set("ticker", v)
	}
	if v, _ := cmd.Flags().GetString("company"); v != "" {
		params.Set("company", v)
	}
	if v, _ := cmd.Flags().GetBool("semantic"); v {
		params.Set("mode", "semantic")
	}
	if v, _ := cmd.Flags().GetString("country"); v != "" {
		params.Set("country", v)
	}
	if v, _ := cmd.Flags().GetString("market"); v != "" {
		params.Set("market", v)
	}
	if v, _ := cmd.Flags().GetString("domicile"); v != "" {
		params.Set("domicile", v)
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

	return params
}

func renderCompanies(f *output.Formatter, companies []api.Company, pagination *api.Pagination, planLimited *bool, opts listOpts) error {
	if f.Format() == "json" {
		rows, err := narrowJSON(f.Fields(), companies)
		if err != nil {
			return err
		}
		result := map[string]interface{}{"companies": rows}
		if pagination != nil {
			result["pagination"] = paginationJSON(pagination)
		}
		addPlanLimited(result, planLimited)
		summary := planLimitedSummary(countNoun(len(companies), companyNoun), planLimited)
		return f.RenderEnvelope(result, summary, breadcrumbsFor(opts.pathOr("nfn companies list")))
	}

	// Inactive companies are rare, so a Status column on every page would be a
	// column of blanks; it appears only when the page holds one. Only a semantic
	// search returns distances, so that column likewise appears only with them.
	showStatus, showDistance := false, false
	for _, c := range companies {
		showStatus = showStatus || !c.IsActive
		showDistance = showDistance || c.Distance != nil
	}

	columns := []string{"ID", "Name", "Listing"}
	colFields := [][]string{{"id"}, {"name"}, {"ticker"}}
	if showStatus {
		columns = append(columns, "Status")
		colFields = append(colFields, []string{"is_active"})
	}
	if showDistance {
		columns = append(columns, "Distance")
		colFields = append(colFields, nil)
	}
	rows := make([][]string, len(companies))
	for i, c := range companies {
		row := []string{c.ID, c.Name, companyListing(c)}
		if showStatus {
			status := ""
			if !c.IsActive {
				status = "inactive"
			}
			row = append(row, status)
		}
		if showDistance {
			row = append(row, formatDistance(c.Distance))
		}
		rows[i] = row
	}
	columns, rows = narrowColumns(f.Fields(), columns, rows, colFields)
	f.RenderFit(columns, rows, output.Layout{Flex: "Name", Drop: []string{"Listing"}})
	if note := noColumnNote(fieldsWithoutColumn(f.Fields(), colFields)); note != "" {
		f.Println(note)
	}
	listFooter{
		noun:       companyNoun,
		getCmd:     "nfn companies get",
		pagination: pagination,
		notes:      opts.notes(),
		emptyHint:  "Try fewer filters, or --semantic to match --q by meaning.",
	}.print(f, len(companies))
	if isPlanLimited(planLimited) {
		f.Println(planLimitedNote)
	}
	return nil
}

// companyListing is a company's main listing as "VOLV-B · XSTO", or "unlisted",
// so a list tells the tradable companies from the subsidiaries and private
// firms a name search also finds.
func companyListing(c api.Company) string {
	switch {
	case c.TickerSymbol == "":
		return "unlisted"
	case c.Exchange == nil || c.Exchange.MicCode == "":
		return c.TickerSymbol
	default:
		return c.TickerSymbol + " · " + c.Exchange.MicCode
	}
}

func init() {
	companiesListCmd.Flags().String("q", "", "Search company names and aliases")
	companiesListCmd.Flags().String("ticker", "", "Look up companies by ticker (comma-separated, max 25, e.g. VOLV-B,ERIC-B): an exact filter that resolves former and exchange-suffixed tickers; every value must resolve or the request errors")
	companiesListCmd.Flags().String("company", "", "Look up companies by ID (comma-separated, max 25): an exact filter; every value must resolve or the request errors")
	companiesListCmd.Flags().String("country", "", "Filter by legal domicile country code (comma-separated ISO2, e.g. SE,NO); --domicile is the canonical name for the same filter")
	companiesListCmd.Flags().String("market", "", "Filter by listing-venue country (comma-separated ISO2, e.g. SE,NO): companies with an active equity listing on an exchange in these countries, including foreign-domiciled issuers")
	companiesListCmd.Flags().String("domicile", "", "Filter by legal domicile country (comma-separated ISO2, e.g. SE,NO): reaches unlisted companies, excludes foreign-domiciled issuers listed locally")
	companiesListCmd.Flags().String("exchange", "", "Filter by exchange MIC code (comma-separated, e.g. XSTO,XCSE); requires an active listing on that exchange")
	companiesListCmd.Flags().String("sector", "", "Filter by sector (e.g. Financials)")
	companiesListCmd.Flags().Bool("listed", false, "Only show companies with an active stock listing")
	companiesListCmd.Flags().Bool("is-active", false, "Only show active companies")
	companiesListCmd.Flags().Bool("watchlist", false, "Only show watchlisted companies")
	addPaginationFlags(companiesListCmd)
	addFieldsFlagFor(companiesListCmd, "company")
	addSemanticFlag(companiesListCmd)

	addPaginationFlags(companiesArticlesCmd)
	companiesArticlesCmd.Flags().Bool("primary-only", false, "Only show articles where this is the primary company")
	addCompanyNewsFilterFlags(companiesArticlesCmd)
	companiesArticlesCmd.Flags().String("source-type", "", "Filter by source type (comma-separated: news_publication, wire_service, press_release, government, trade_publication, blog, stock_exchange, research)")
	companiesArticlesCmd.Flags().String("content-type", "", "Filter by content type")
	addFieldsFlagFor(companiesArticlesCmd, "article")

	addPaginationFlags(companiesStoriesCmd)
	addCompanyNewsFilterFlags(companiesStoriesCmd)
	addFieldsFlagFor(companiesStoriesCmd, "story")

	addEventFilterFlags(companiesEventsCmd)
	addPaginationFlags(companiesEventsCmd)
	addFieldsFlagFor(companiesEventsCmd, "event")

	companiesCmd.AddCommand(companiesListCmd)
	companiesCmd.AddCommand(companiesGetCmd)
	companiesCmd.AddCommand(companiesArticlesCmd)
	companiesCmd.AddCommand(companiesStoriesCmd)
	companiesCmd.AddCommand(companiesEventsCmd)
	rootCmd.AddCommand(companiesCmd)
}

var companyNoun = [2]string{"company", "companies"}
