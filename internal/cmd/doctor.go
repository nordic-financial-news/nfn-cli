package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/config"
	"github.com/spf13/cobra"
)

type checkResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

const (
	statusPass = "pass"
	statusFail = "fail"
	statusWarn = "warn"
	statusSkip = "skip"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check CLI health and configuration",
	Long:  "Runs diagnostic checks on your nfn configuration, API connectivity, authentication, and CLI version.",
	Annotations: map[string]string{
		"skipAuth": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := formatterFromContext(cmd)

		baseURL, err := resolveBaseURL(cmd)
		if err != nil {
			return err
		}

		var results []checkResult

		// 1. Config directory
		results = append(results, checkConfigDir())

		// 2. API key configured
		apiKey, keyResult := checkAPIKey()
		results = append(results, keyResult)

		// 3. API reachable
		results = append(results, checkAPIReachable(cmd.Context(), baseURL))

		// 4. API key valid (skip if no key)
		if apiKey != "" {
			results = append(results, checkAPIKeyValid(cmd.Context(), baseURL, apiKey))
		} else {
			results = append(results, checkResult{Name: "API key valid", Status: statusSkip, Detail: "no API key configured"})
		}

		// 5. CLI version
		results = append(results, checkCLIVersion())

		if f.Format() == "json" {
			return f.RenderEnvelope(results, doctorSummary(results), breadcrumbsFor("nfn doctor"))
		}

		noColor, _ := cmd.Flags().GetBool("no-color")
		for _, r := range results {
			icon := statusIcon(r.Status, noColor)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s", icon, r.Name)
			if r.Detail != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), " (%s)", r.Detail)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}
		return nil
	},
}

func checkConfigDir() checkResult {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	nfnDir := filepath.Join(configDir, "nfn")

	if _, err := os.Stat(nfnDir); err != nil {
		return checkResult{Name: "Config directory", Status: statusWarn, Detail: nfnDir + " not found"}
	}
	return checkResult{Name: "Config directory", Status: statusPass, Detail: nfnDir}
}

func checkAPIKey() (string, checkResult) {
	key, err := config.GetAPIKey()
	if err != nil {
		return "", checkResult{Name: "API key configured", Status: statusFail, Detail: "not set — run 'nfn auth login'"}
	}
	return key, checkResult{Name: "API key configured", Status: statusPass}
}

func checkAPIReachable(ctx context.Context, baseURL string) checkResult {
	client := api.NewClient("", api.WithBaseURL(baseURL))
	var result map[string]interface{}
	_, err := client.Get(ctx, "/health", nil, &result)
	if err != nil {
		return checkResult{Name: "API reachable", Status: statusFail, Detail: err.Error()}
	}
	return checkResult{Name: "API reachable", Status: statusPass}
}

func checkAPIKeyValid(ctx context.Context, baseURL, apiKey string) checkResult {
	client := api.NewClient(apiKey, api.WithBaseURL(baseURL))
	var meta map[string]interface{}
	resp, err := client.Get(ctx, "/meta", nil, &meta)
	if err != nil {
		return checkResult{Name: "API key valid", Status: statusFail, Detail: err.Error()}
	}
	detail := ""
	if resp.RateLimitInfo != nil {
		detail = fmt.Sprintf("%d/%d requests remaining", resp.RateLimitInfo.Remaining, resp.RateLimitInfo.Limit)
	}
	return checkResult{Name: "API key valid", Status: statusPass, Detail: detail}
}

func checkCLIVersion() checkResult {
	if Version == "dev" {
		return checkResult{Name: "CLI version", Status: statusSkip, Detail: "dev build"}
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get("https://api.github.com/repos/nordic-financial-news/nfn-cli/releases/latest")
	if err != nil {
		return checkResult{Name: "CLI version", Status: statusWarn, Detail: "could not check: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return checkResult{Name: "CLI version", Status: statusWarn, Detail: fmt.Sprintf("could not check (HTTP %d)", resp.StatusCode)}
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&release); err != nil {
		return checkResult{Name: "CLI version", Status: statusWarn, Detail: "could not parse response"}
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	if latest != current {
		return checkResult{Name: "CLI version", Status: statusWarn, Detail: fmt.Sprintf("update available: v%s → v%s", current, latest)}
	}
	return checkResult{Name: "CLI version", Status: statusPass, Detail: "v" + current}
}

func statusIcon(status string, noColor bool) string {
	if noColor {
		return status
	}
	switch status {
	case statusPass:
		return "\033[32mpass\033[0m"
	case statusFail:
		return "\033[31mfail\033[0m"
	case statusWarn:
		return "\033[33mwarn\033[0m"
	case statusSkip:
		return "\033[90mskip\033[0m"
	default:
		return status
	}
}

func doctorSummary(results []checkResult) string {
	pass, fail, warn := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case statusPass:
			pass++
		case statusFail:
			fail++
		case statusWarn:
			warn++
		}
	}
	if fail > 0 {
		return fmt.Sprintf("%d passed, %d failed, %d %s", pass, fail, warn, pluralize("warning", warn))
	}
	if warn > 0 {
		return fmt.Sprintf("%d passed, %d %s", pass, warn, pluralize("warning", warn))
	}
	return fmt.Sprintf("all %d checks passed", pass)
}

func pluralize(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
