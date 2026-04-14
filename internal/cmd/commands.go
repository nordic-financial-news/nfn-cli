package cmd

import (
	"fmt"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type commandInfo struct {
	Command     string            `json:"command"`
	Description string            `json:"description"`
	Flags       []flagInfo        `json:"flags"`
	Arguments   string            `json:"arguments"`
	Examples    []string          `json:"examples,omitempty"`
	Related     []output.Breadcrumb `json:"related,omitempty"`
}

type flagInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

type commandsCatalog struct {
	Version     string        `json:"version"`
	GlobalFlags []flagInfo    `json:"global_flags"`
	Commands    []commandInfo `json:"commands"`
}

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all commands as structured JSON for agent consumption",
	Long:  "Outputs the full command catalog as JSON, including flags, arguments, examples, and related commands. Designed for AI agents and scripts.",
	Annotations: map[string]string{
		"skipAuth": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := formatterFromContext(cmd)
		filter, _ := cmd.Flags().GetString("command")

		catalog := buildCatalog(filter)

		return f.RenderEnvelope(catalog, catalogSummary(catalog), nil)
	},
}

func buildCatalog(filter string) commandsCatalog {
	catalog := commandsCatalog{
		Version:     Version,
		GlobalFlags: extractGlobalFlags(),
		Commands:    []commandInfo{},
	}

	walkCommands(rootCmd, filter, &catalog.Commands)
	return catalog
}

func extractGlobalFlags() []flagInfo {
	var flags []flagInfo
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, flagInfo{
			Name:        f.Name,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})
	return flags
}

func walkCommands(parent *cobra.Command, filter string, out *[]commandInfo) {
	for _, cmd := range parent.Commands() {
		if cmd.Hidden || cmd.Name() == "help" || cmd.Name() == "commands" {
			continue
		}

		fullPath := cmd.CommandPath()

		// If the command is runnable, add it
		if cmd.RunE != nil || cmd.Run != nil {
			if filter == "" || matchesFilter(fullPath, filter) {
				*out = append(*out, buildCommandInfo(cmd, fullPath))
			}
		}

		// Recurse into subcommands
		walkCommands(cmd, filter, out)
	}
}

func matchesFilter(fullPath, filter string) bool {
	// Match either full path ("nfn articles list") or relative ("articles list")
	return fullPath == filter || strings.TrimPrefix(fullPath, "nfn ") == filter
}

func buildCommandInfo(cmd *cobra.Command, fullPath string) commandInfo {
	info := commandInfo{
		Command:     fullPath,
		Description: cmd.Short,
		Flags:       extractLocalFlags(cmd),
		Arguments:   extractArguments(cmd),
	}

	if cmd.Example != "" {
		for _, line := range strings.Split(strings.TrimSpace(cmd.Example), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				info.Examples = append(info.Examples, line)
			}
		}
	}

	if bc := breadcrumbsFor(fullPath); bc != nil {
		info.Related = bc
	}

	return info
}

func extractLocalFlags(cmd *cobra.Command) []flagInfo {
	var flags []flagInfo
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, flagInfo{
			Name:        f.Name,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})
	if flags == nil {
		flags = []flagInfo{}
	}
	return flags
}

func extractArguments(cmd *cobra.Command) string {
	use := cmd.Use
	if idx := strings.Index(use, " "); idx != -1 {
		return use[idx+1:]
	}
	return ""
}

func catalogSummary(catalog commandsCatalog) string {
	return fmt.Sprintf("%d commands", len(catalog.Commands))
}

func init() {
	commandsCmd.Flags().String("command", "", "Filter to a specific command (e.g. \"articles list\")")
	rootCmd.AddCommand(commandsCmd)
}
