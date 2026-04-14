package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for nfn.

To load completions:

Bash:
  $ source <(nfn completion bash)
  # To load completions for each session, execute once:
  $ nfn completion bash > /etc/bash_completion.d/nfn

Zsh:
  $ source <(nfn completion zsh)
  # To load completions for each session, execute once:
  $ nfn completion zsh > "${fpath[1]}/_nfn"

Fish:
  $ nfn completion fish | source
  # To load completions for each session, execute once:
  $ nfn completion fish > ~/.config/fish/completions/nfn.fish
`,
	Annotations: map[string]string{
		"skipAuth": "true",
	},
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
