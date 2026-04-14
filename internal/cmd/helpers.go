package cmd

import (
	"fmt"
	"net/url"
	"strings"

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

func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 0, "Maximum number of results per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	cmd.Flags().Bool("all", false, "Fetch all pages of results")
}

func addFieldsFlag(cmd *cobra.Command) {
	cmd.Flags().String("fields", "", "Comma-separated list of fields to include")
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
