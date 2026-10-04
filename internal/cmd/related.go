package cmd

import "github.com/nordic-financial-news/nfn-cli/internal/output"

// commandBreadcrumbs maps command paths to their suggested follow-up commands.
// Used by both envelope breadcrumbs and the commands catalog.
var commandBreadcrumbs = map[string][]output.Breadcrumb{
	"nfn articles list": {
		{Description: "View article details", Command: "nfn articles get <id>"},
		{Description: "List stories", Command: "nfn stories list"},
		{Description: "List sources", Command: "nfn sources list"},
		{Description: "Search", Command: "nfn search <query>"},
	},
	"nfn articles get": {
		{Description: "List articles", Command: "nfn articles list"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "View the event this article reports", Command: "nfn events get <id>"},
	},
	"nfn companies list": {
		{Description: "View company details", Command: "nfn companies get <identifier>"},
	},
	"nfn companies get": {
		{Description: "List company articles", Command: "nfn companies articles <identifier>"},
		{Description: "List company stories", Command: "nfn companies stories <identifier>"},
		{Description: "List company events", Command: "nfn companies events <identifier>"},
		{Description: "List company calendar events", Command: "nfn calendar list --ticker <ticker>"},
		{Description: "List companies", Command: "nfn companies list"},
	},
	"nfn companies articles": {
		{Description: "View article details", Command: "nfn articles get <id>"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
	},
	"nfn companies stories": {
		{Description: "View story details", Command: "nfn stories get <id>"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
	},
	"nfn stories list": {
		{Description: "View story details", Command: "nfn stories get <id>"},
		{Description: "List articles", Command: "nfn articles list"},
		{Description: "List sources", Command: "nfn sources list"},
	},
	"nfn stories get": {
		{Description: "List stories", Command: "nfn stories list"},
		{Description: "View article details", Command: "nfn articles get <id>"},
		{Description: "View the event behind this story", Command: "nfn events get <id>"},
	},
	"nfn calendar list": {
		{Description: "View calendar event details", Command: "nfn calendar get <id>"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "List articles", Command: "nfn articles list"},
	},
	"nfn calendar get": {
		{Description: "List calendar events", Command: "nfn calendar list"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "View source article", Command: "nfn articles get <id>"},
	},
	"nfn events list": {
		{Description: "View event details", Command: "nfn events get <id>"},
		{Description: "List event types", Command: "nfn event-types"},
		{Description: "List articles", Command: "nfn articles list"},
		{Description: "List stories", Command: "nfn stories list"},
	},
	"nfn events get": {
		{Description: "View member article details", Command: "nfn articles get <id>"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "List commentary on this event", Command: "nfn events list --related-to <id>"},
		{Description: "List events", Command: "nfn events list"},
	},
	"nfn event-types": {
		{Description: "Filter events by type", Command: "nfn events list --event-type <slug>"},
		{Description: "List events", Command: "nfn events list"},
	},
	"nfn companies events": {
		{Description: "View event details", Command: "nfn events get <id>"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
	},
	"nfn categories": {
		{Description: "Filter articles by category", Command: "nfn articles list --category <id>"},
		{Description: "Filter stories by category", Command: "nfn stories list --category <id>"},
	},
	"nfn countries list": {
		{Description: "View country details", Command: "nfn countries get <code>"},
	},
	"nfn countries get": {
		{Description: "List exchanges in country", Command: "nfn exchanges list --country <code>"},
		{Description: "List articles from country", Command: "nfn articles list --country <code>"},
	},
	"nfn exchanges list": {
		{Description: "View exchange details", Command: "nfn exchanges get <mic>"},
	},
	"nfn exchanges get": {
		{Description: "List exchange companies", Command: "nfn exchanges companies <mic>"},
		{Description: "List exchanges", Command: "nfn exchanges list"},
	},
	"nfn exchanges companies": {
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "View exchange details", Command: "nfn exchanges get <mic>"},
	},
	"nfn indices list": {
		{Description: "View index details", Command: "nfn indices get <symbol>"},
	},
	"nfn indices get": {
		{Description: "List index companies", Command: "nfn indices companies <symbol>"},
		{Description: "List indices", Command: "nfn indices list"},
	},
	"nfn indices companies": {
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "View index details", Command: "nfn indices get <symbol>"},
	},
	"nfn sources list": {
		{Description: "Filter articles by source", Command: "nfn articles list --sources <ids>"},
		{Description: "Filter articles by source type", Command: "nfn articles list --source-type <type>"},
		{Description: "Filter stories by source", Command: "nfn stories list --sources <ids>"},
	},
	"nfn search": {
		{Description: "View article details", Command: "nfn articles get <id>"},
		{Description: "View company details", Command: "nfn companies get <identifier>"},
		{Description: "View story details", Command: "nfn stories get <id>"},
	},
	"nfn watchlists list": {
		{Description: "View watchlist details", Command: "nfn watchlists get <id>"},
	},
	"nfn watchlists get": {
		{Description: "List articles for this watchlist", Command: "nfn articles list --watchlist-id <id>"},
		{Description: "List stories for this watchlist", Command: "nfn stories list --watchlist-id <id>"},
		{Description: "List calendar events for this watchlist", Command: "nfn calendar list --watchlist-id <id>"},
		{Description: "List events for this watchlist", Command: "nfn events list --watchlist-id <id>"},
		{Description: "List watchlists", Command: "nfn watchlists list"},
	},
	"nfn doctor": {
		{Description: "Authenticate", Command: "nfn auth login"},
		{Description: "Check CLI version", Command: "nfn version"},
	},
}

// breadcrumbsFor returns the breadcrumbs for a given command path.
func breadcrumbsFor(cmdPath string) []output.Breadcrumb {
	if bc, ok := commandBreadcrumbs[cmdPath]; ok {
		return bc
	}
	return nil
}
