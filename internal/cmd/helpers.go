package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/config"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

const trustedHost = "nordicfinancialnews.com"

// resolveBaseURL returns the API base URL from the --api-url flag or config,
// and validates that it uses HTTPS and points to a trusted host.
func resolveBaseURL(cmd *cobra.Command) (string, error) {
	baseURL, _ := cmd.Flags().GetString("api-url")
	if baseURL == "" {
		baseURL = config.GetBaseURL()
	}
	if !strings.HasPrefix(baseURL, "https://") {
		return "", fmt.Errorf("API URL must use HTTPS (got %q)", baseURL)
	}

	if !isTrustedHost(baseURL) {
		allowCustom, _ := cmd.Flags().GetBool("allow-custom-host")
		if !allowCustom {
			return "", fmt.Errorf(
				"API URL host is not %s — to send your API key to a custom host, pass --allow-custom-host",
				trustedHost,
			)
		}
	}

	return baseURL, nil
}

// isTrustedHost returns true if the URL's host is nordicfinancialnews.com
// or a subdomain of it.
func isTrustedHost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == trustedHost || strings.HasSuffix(host, "."+trustedHost)
}

func apiClientFromContext(cmd *cobra.Command) *api.Client {
	return cmd.Context().Value(clientKey).(*api.Client)
}

func formatterFromContext(cmd *cobra.Command) *output.Formatter {
	return cmd.Context().Value(formatterKey).(*output.Formatter)
}

// watchlistFilterError clarifies the error from a list request when a watchlist
// filter was applied. A watchlist-filtered request returns 403 when the API key
// lacks the read:watchlist scope; surface that as an actionable message. When no
// watchlist filter was used (watchlistID == "") or the error is unrelated, the
// original error passes through unchanged.
func watchlistFilterError(err error, watchlistID string) error {
	if err == nil || watchlistID == "" {
		return err
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.IsForbidden() {
		return fmt.Errorf("watchlist filtering requires the read:watchlist scope on your API key (the API returned 403 Forbidden)")
	}
	return err
}

func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 0, "Maximum number of results per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	cmd.Flags().Bool("all", false, "Fetch all pages of results")
}

// contentFilterFlags maps CLI flag names to their query-param names for the
// content filters shared across the article and story feeds and the nested
// company routes. A command participates simply by registering the flag; only
// flags a command both registered and the user set are copied, so a route never
// sends a filter it does not support (e.g. source-type/content-type stay off the
// story routes, which do not register them).
var contentFilterFlags = []struct{ flag, param string }{
	{"q", "q"},
	{"sort", "sort"},
	{"country", "country"},
	{"category", "category"},
	{"sources", "sources"},
	{"source-type", "source_type"},
	{"content-type", "content_type"},
	{"published-after", "published_after"},
	{"published-before", "published_before"},
	{"updated-after", "updated_after"},
	{"fields", "fields"},
}

// setContentFilters copies each registered, non-empty content filter flag from
// cmd into params, translating the flag name to its query-param name.
func setContentFilters(params url.Values, cmd *cobra.Command) {
	for _, f := range contentFilterFlags {
		if cmd.Flags().Lookup(f.flag) == nil {
			continue
		}
		if v, _ := cmd.Flags().GetString(f.flag); v != "" {
			params.Set(f.param, v)
		}
	}
}

// setPaginationParams copies the --limit and --cursor flags into params.
func setPaginationParams(params url.Values, cmd *cobra.Command) {
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		params.Set("limit", fmt.Sprintf("%d", v))
	}
	if v, _ := cmd.Flags().GetString("cursor"); v != "" {
		params.Set("cursor", v)
	}
}

// addCompanyNewsFilterFlags registers the content filters shared by the nested
// company article and story routes (item 9 filter parity). Callers that also
// need source-type/content-type (articles only) register those separately.
func addCompanyNewsFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("q", "", "Search query (relevance-ranked single page; add --sort latest for newest-first with cursor paging)")
	cmd.Flags().String("sort", "", "Sort order (e.g. latest)")
	cmd.Flags().String("country", "", "Filter by country code (comma-separated ISO2, e.g. SE,NO)")
	cmd.Flags().String("category", "", "Filter by category ID from nfn categories (comma-separated for several; unknown IDs are ignored)")
	cmd.Flags().String("sources", "", "Filter by source IDs (comma-separated, max 25)")
	cmd.Flags().String("published-after", "", "Filter by published after date")
	cmd.Flags().String("published-before", "", "Filter by published before date")
	cmd.Flags().String("updated-after", "", "ISO 8601 datetime for incremental sync")
}

// semanticMinQuery and semanticMaxQuery bound the query the API accepts in
// semantic mode. Checking them here turns a wasted round trip into an
// immediate message, and semantic requests come from a small hourly budget.
const (
	semanticMinQuery = 2
	semanticMaxQuery = 200
)

func addMarketSegmentFlag(cmd *cobra.Command) {
	cmd.Flags().String("market-segment", "", "Filter by the primary listing's cap segment (comma-separated: large_cap, mid_cap, small_cap, first_north_premier, first_north_growth). Nasdaq Nordic markets only")
}

func addSemanticFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("semantic", false, "Match --q by meaning rather than by keyword, ranking results by embedding similarity; requires --q (2-200 characters) and returns a single page with a distance on each result")
}

// semanticRequested reports whether --semantic was asked for on a command that
// offers it.
func semanticRequested(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("semantic")
	return v
}

// semanticConflicts are the flags the API refuses alongside mode=semantic,
// each with the reason it cannot apply. A semantic response is a single
// distance-ranked page, so the flags that walk pages or ask for incremental
// slices have nothing to act on, and --sort would override the ranking that
// semantic mode exists to produce.
var semanticConflicts = []struct{ flag, because string }{
	{"cursor", "semantic results are a single page ranked by closeness, with no cursor to follow"},
	{"all", "semantic results are a single page ranked by closeness, with no cursor to follow"},
	{"updated-after", "semantic results are a single page ranked by closeness, with no cursor to follow"},
	{"sort", "semantic mode defines its own ranking by closeness to the query"},
}

// validateSemantic enforces the constraints the API places on semantic mode,
// before a request is spent against the hourly semantic budget.
func validateSemantic(cmd *cobra.Command) error {
	if !semanticRequested(cmd) {
		return nil
	}

	q, _ := cmd.Flags().GetString("q")
	if strings.TrimSpace(q) == "" {
		return fmt.Errorf("--semantic needs a query to match against: add --q \"...\"")
	}
	if n := len([]rune(q)); n < semanticMinQuery || n > semanticMaxQuery {
		return fmt.Errorf("--semantic requires --q between %d and %d characters (got %d)", semanticMinQuery, semanticMaxQuery, n)
	}

	for _, c := range semanticConflicts {
		f := cmd.Flags().Lookup(c.flag)
		if f != nil && f.Changed {
			return fmt.Errorf("--%s cannot be combined with --semantic: %s", c.flag, c.because)
		}
	}

	return nil
}

// semanticError rewrites the two failures specific to semantic search into
// messages that say what to do next. Other errors pass through unchanged.
func semanticError(err error, semantic bool) error {
	if err == nil || !semantic {
		return err
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	switch {
	case apiErr.IsUnknownParameter("mode"):
		// Semantic search rolls out per API key, so a key outside the rollout
		// gets its "mode" parameter rejected. Name --semantic, which the caller
		// actually typed, rather than the mode parameter it sets for them.
		return fmt.Errorf("semantic search is not enabled for this API key yet (the API rejected the 'mode' parameter that --semantic sets). Drop --semantic to run the same query as a keyword search")
	case apiErr.IsSemanticUnavailable():
		return fmt.Errorf("semantic search is temporarily unavailable (the API returned 503). It never falls back to keyword matching, so either retry shortly or drop --semantic to run the same query lexically")
	case apiErr.IsRateLimited():
		msg := "rate limited by the API (429). Semantic requests draw on their own budget of 120 per hour per API key, shared across every semantic-mode resource and separate from the standard rate limit, so this may be that budget rather than the usual one"
		if apiErr.RetryAfter > 0 {
			msg += fmt.Sprintf(" — retry in %s", apiErr.RetryAfter.Round(time.Second))
		}
		return errors.New(msg)
	}

	return err
}

// formatDistance renders a semantic distance for table output. The API rounds
// to four decimal places and lower is closer; a lexical result has none.
func formatDistance(d *float64) string {
	if d == nil {
		return ""
	}
	return strconv.FormatFloat(*d, 'f', 4, 64)
}

// planLimitedNote is printed under table output when the caller's plan hid
// records from the result, so a capped result isn't mistaken for a short one.
const planLimitedNote = "Note: results limited by your plan — some records were hidden."

// planLimitedFrom returns the plan_limited flag carried on an API response,
// tolerating a nil response.
func planLimitedFrom(resp *api.Response) *bool {
	if resp == nil {
		return nil
	}
	return resp.PlanLimited
}

// isPlanLimited reports whether the API flagged the result as plan-limited. A
// nil flag means the API said nothing, which is not the same as false.
func isPlanLimited(planLimited *bool) bool {
	return planLimited != nil && *planLimited
}

// addPlanLimited adds the plan_limited flag to a JSON result map when the API
// sent one. Absent stays absent so consumers can tell "not restricted" from
// "this endpoint doesn't report it".
func addPlanLimited(result map[string]interface{}, planLimited *bool) {
	if planLimited != nil {
		result["plan_limited"] = *planLimited
	}
}

// planLimitedSummary appends a truncation warning to an envelope summary, since
// human-facing wrappers often read only the summary line.
func planLimitedSummary(summary string, planLimited *bool) string {
	if !isPlanLimited(planLimited) {
		return summary
	}
	return summary + " (limited by your plan — some records were hidden)"
}

// paginationJSON returns a JSON-friendly pagination map, or nil if there's no next page.
func paginationJSON(p *api.Pagination) map[string]interface{} {
	if p == nil {
		return map[string]interface{}{"count": 0, "next_cursor": nil}
	}
	var cursor interface{}
	if p.NextCursor != "" {
		cursor = p.NextCursor
	}
	return map[string]interface{}{"count": p.Count, "next_cursor": cursor}
}
