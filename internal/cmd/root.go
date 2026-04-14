package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/config"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type contextKey string

const (
	clientKey    contextKey = "client"
	formatterKey contextKey = "formatter"
)

// jsonMode tracks whether the current invocation is using JSON output,
// so Execute() can render errors as JSON envelopes.
var jsonMode bool

var rootCmd = &cobra.Command{
	Use:   "nfn",
	Short: "CLI for the Nordic Financial News API",
	Long: `Query Nordic financial news, stories, and companies from your terminal.

Output is a table when run interactively (TTY) and JSON when piped or
called by scripts/agents. Override with --format table or --format json.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		config.Init()

		formatExplicit := cmd.Flags().Changed("format")
		format, _ := cmd.Flags().GetString("format")
		if format == "" {
			format = config.GetFormat()
		}
		// Auto-switch to JSON when stdout is not a TTY (piped/scripted),
		// unless the user explicitly passed --format
		if !formatExplicit && !term.IsTerminal(int(os.Stdout.Fd())) {
			format = "json"
		}
		noColor, _ := cmd.Flags().GetBool("no-color")
		jsonMode = format == "json"

		formatter := output.NewFormatter(format, noColor)
		ctx := context.WithValue(cmd.Context(), formatterKey, formatter)

		// Skip auth for annotated commands
		if cmd.Annotations["skipAuth"] == "true" {
			cmd.SetContext(ctx)
			return nil
		}

		apiKey, err := config.GetAPIKey()
		if err != nil {
			return err
		}

		baseURL, err := resolveBaseURL(cmd)
		if err != nil {
			return err
		}

		api.Version = Version
		client := api.NewClient(apiKey, api.WithBaseURL(baseURL))
		ctx = context.WithValue(ctx, clientKey, client)
		cmd.SetContext(ctx)

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		if jsonMode {
			f := output.NewFormatter("json", false)
			_ = f.RenderError(err)
		} else {
			_, _ = fmt.Fprintln(rootCmd.ErrOrStderr(), err)
		}
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().String("format", "", "Output format: table or json")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	rootCmd.PersistentFlags().String("api-url", "", "Override API base URL")
	rootCmd.PersistentFlags().Bool("allow-custom-host", false, "Allow sending API key to a non-default host")
}
