package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildCatalogIncludesAllCommands(t *testing.T) {
	catalog := buildCatalog("")

	// Expect all runnable commands to appear
	wantCommands := []string{
		"nfn articles list",
		"nfn articles get",
		"nfn companies list",
		"nfn companies get",
		"nfn companies articles",
		"nfn companies stories",
		"nfn stories list",
		"nfn stories get",
		"nfn categories",
		"nfn countries list",
		"nfn countries get",
		"nfn sources list",
		"nfn exchanges list",
		"nfn exchanges get",
		"nfn exchanges companies",
		"nfn indices list",
		"nfn indices get",
		"nfn indices companies",
		"nfn search",
		"nfn watchlists list",
		"nfn watchlists get",
		"nfn auth login",
		"nfn auth status",
		"nfn auth logout",
		"nfn doctor",
		"nfn version",
		"nfn completion",
	}

	found := map[string]bool{}
	for _, cmd := range catalog.Commands {
		found[cmd.Command] = true
	}

	for _, want := range wantCommands {
		if !found[want] {
			t.Errorf("missing command in catalog: %q", want)
		}
	}
}

func TestBuildCatalogFilter(t *testing.T) {
	catalog := buildCatalog("articles list")

	if len(catalog.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(catalog.Commands))
	}
	if catalog.Commands[0].Command != "nfn articles list" {
		t.Errorf("command = %q, want %q", catalog.Commands[0].Command, "nfn articles list")
	}
}

func TestBuildCatalogFilterFullPath(t *testing.T) {
	catalog := buildCatalog("nfn articles list")

	if len(catalog.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(catalog.Commands))
	}
}

func TestBuildCatalogFlagsExtracted(t *testing.T) {
	catalog := buildCatalog("articles list")
	if len(catalog.Commands) == 0 {
		t.Fatal("no commands found")
	}

	cmd := catalog.Commands[0]
	flagNames := map[string]bool{}
	for _, f := range cmd.Flags {
		flagNames[f.Name] = true
	}

	for _, want := range []string{"country", "category", "ticker", "limit", "cursor", "all"} {
		if !flagNames[want] {
			t.Errorf("missing flag %q in articles list", want)
		}
	}
}

func TestBuildCatalogExamplesExtracted(t *testing.T) {
	catalog := buildCatalog("articles list")
	if len(catalog.Commands) == 0 {
		t.Fatal("no commands found")
	}

	if len(catalog.Commands[0].Examples) == 0 {
		t.Error("expected examples for articles list")
	}
}

func TestBuildCatalogGlobalFlags(t *testing.T) {
	catalog := buildCatalog("")

	flagNames := map[string]bool{}
	for _, f := range catalog.GlobalFlags {
		flagNames[f.Name] = true
	}

	for _, want := range []string{"format", "no-color", "api-url", "allow-custom-host"} {
		if !flagNames[want] {
			t.Errorf("missing global flag %q", want)
		}
	}
}

func TestBuildCatalogRelatedCommands(t *testing.T) {
	catalog := buildCatalog("articles list")
	if len(catalog.Commands) == 0 {
		t.Fatal("no commands found")
	}

	if len(catalog.Commands[0].Related) == 0 {
		t.Error("expected related commands for articles list")
	}
}

func TestExtractArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		use  string
		want string
	}{
		{"list", ""},
		{"get <id>", "<id>"},
		{"get <identifier>", "<identifier>"},
		{"search <query>", "<query>"},
		{"companies <id-or-mic>", "<id-or-mic>"},
	}

	for _, tt := range tests {
		cmd := &cobra.Command{Use: tt.use}
		got := extractArguments(cmd)
		if got != tt.want {
			t.Errorf("extractArguments(%q) = %q, want %q", tt.use, got, tt.want)
		}
	}
}

func TestCatalogSummary(t *testing.T) {
	t.Parallel()

	catalog := commandsCatalog{
		Commands: make([]commandInfo, 25),
	}
	got := catalogSummary(catalog)
	if got != "25 commands" {
		t.Errorf("catalogSummary() = %q, want %q", got, "25 commands")
	}
}
