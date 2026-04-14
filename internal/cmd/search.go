package cmd

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search articles, stories, and companies",
	Example: `  nfn search "Volvo"
  nfn search "battery" --type companies
  nfn search "IPO" --limit 5 --format json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		client := apiClientFromContext(cmd)
		f := formatterFromContext(cmd)

		params := url.Values{}
		params.Set("q", args[0])
		if v, _ := cmd.Flags().GetString("type"); v != "" {
			params.Set("type", v)
		}
		if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
			params.Set("limit", fmt.Sprintf("%d", v))
		}

		stop := startSpinner("Searching...")
		results, _, err := client.Search(cmd.Context(), params)
		stop()
		if err != nil {
			return err
		}

		if f.Format() == "json" {
			total := len(results.Companies) + len(results.Articles) + len(results.Stories)
			return f.RenderEnvelope(results, fmt.Sprintf("%d results", total), breadcrumbsFor("nfn search"))
		}

		printed := false
		if len(results.Companies) > 0 {
			f.Println("Companies:")
			if err := renderCompanies(f, results.Companies, nil); err != nil {
				return err
			}
			f.Println()
			printed = true
		}
		if len(results.Articles) > 0 {
			f.Println("Articles:")
			if err := renderArticles(f, results.Articles, nil); err != nil {
				return err
			}
			f.Println()
			printed = true
		}
		if len(results.Stories) > 0 {
			f.Println("Stories:")
			if err := renderStories(f, results.Stories, nil); err != nil {
				return err
			}
			f.Println()
			printed = true
		}

		if !printed {
			f.Println("No results found.")
		}

		return nil
	},
}

func startSpinner(message string) func() {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				_, _ = fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			default:
				_, _ = fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], message)
				i++
				select {
				case <-done:
					_, _ = fmt.Fprintf(os.Stderr, "\r\033[K")
					return
				case <-waitMillis(80):
				}
			}
		}
	}()
	return func() { close(done) }
}

func waitMillis(ms int) <-chan time.Time {
	return time.After(time.Duration(ms) * time.Millisecond)
}

func init() {
	searchCmd.Flags().String("type", "", "Filter by result type (articles, stories, companies)")
	searchCmd.Flags().Int("limit", 0, "Maximum number of results")
	rootCmd.AddCommand(searchCmd)
}
