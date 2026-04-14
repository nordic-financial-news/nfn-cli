package cmd

import (
	"fmt"
	"os"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage API authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your API key",
	Long:  "Stores your API key in the system keyring. Get your key at https://nordicfinancialnews.com/account.",
	Example: `  nfn auth login`,
	Annotations: map[string]string{
		"skipAuth": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("Enter API key: ")
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("reading API key: %w", err)
		}

		key := string(keyBytes)
		if key == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		// Validate the key by hitting /meta
		baseURL, err := resolveBaseURL(cmd)
		if err != nil {
			return err
		}
		client := api.NewClient(key, api.WithBaseURL(baseURL))

		var meta map[string]interface{}
		_, err = client.Get(cmd.Context(), "/meta", nil, &meta)
		if err != nil {
			return fmt.Errorf("invalid API key: %w", err)
		}

		if err := config.SetAPIKey(key); err != nil {
			return fmt.Errorf("storing API key: %w", err)
		}

		fmt.Println("Authenticated successfully.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status and rate limits",
	Annotations: map[string]string{
		"skipAuth": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := formatterFromContext(cmd)

		apiKey, err := config.GetAPIKey()
		if err != nil {
			if f.Format() == "json" {
				return f.RenderEnvelope(map[string]interface{}{"authenticated": false}, "not authenticated", nil)
			}
			fmt.Println("Not authenticated. Run 'nfn auth login' to authenticate.")
			return nil
		}

		baseURL, err := resolveBaseURL(cmd)
		if err != nil {
			return err
		}
		client := api.NewClient(apiKey, api.WithBaseURL(baseURL))

		var meta map[string]interface{}
		resp, err := client.Get(cmd.Context(), "/meta", nil, &meta)
		if err != nil {
			return fmt.Errorf("checking auth status: %w", err)
		}

		if f.Format() == "json" {
			result := map[string]interface{}{"authenticated": true}
			if resp.RateLimitInfo != nil {
				result["rate_limit"] = map[string]interface{}{
					"remaining": resp.RateLimitInfo.Remaining,
					"limit":     resp.RateLimitInfo.Limit,
					"reset":     resp.RateLimitInfo.Reset,
				}
			}
			return f.RenderEnvelope(result, "authenticated", nil)
		}

		fmt.Println("Authenticated: yes")
		if resp.RateLimitInfo != nil {
			fmt.Printf("Rate limit: %d/%d remaining\n", resp.RateLimitInfo.Remaining, resp.RateLimitInfo.Limit)
			if !resp.RateLimitInfo.Reset.IsZero() {
				fmt.Printf("Resets at: %s\n", resp.RateLimitInfo.Reset.Format("15:04:05"))
			}
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored API key",
	Annotations: map[string]string{
		"skipAuth": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := formatterFromContext(cmd)

		if err := config.DeleteAPIKey(); err != nil {
			return err
		}

		if f.Format() == "json" {
			return f.RenderEnvelope(map[string]interface{}{"logged_out": true}, "logged out", nil)
		}
		fmt.Println("Logged out successfully.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}
