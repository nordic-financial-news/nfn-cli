package cmd

import (
	"testing"
)

func TestCommandTree(t *testing.T) {
	// Not parallel — rootCmd.Commands() sorts in place

	// Expected subcommand paths
	want := map[string]bool{
		"articles":            true,
		"articles list":       true,
		"articles get":        true,
		"companies":           true,
		"companies list":      true,
		"companies get":       true,
		"companies articles":  true,
		"companies stories":   true,
		"stories":             true,
		"stories list":        true,
		"stories get":         true,
		"categories":          true,
		"countries":           true,
		"countries list":      true,
		"countries get":       true,
		"sources":             true,
		"sources list":        true,
		"exchanges":           true,
		"exchanges list":      true,
		"exchanges get":       true,
		"exchanges companies": true,
		"indices":             true,
		"indices list":        true,
		"indices get":         true,
		"indices companies":   true,
		"search":              true,
		"auth":                true,
		"auth login":          true,
		"auth status":         true,
		"auth logout":         true,
		"doctor":              true,
		"version":             true,
		"completion":          true,
		"commands":            true,
		"watchlists":          true,
		"watchlists list":     true,
		"watchlists get":      true,
	}

	found := map[string]bool{}

	// Walk the command tree
	for _, cmd := range rootCmd.Commands() {
		found[cmd.Name()] = true
		for _, sub := range cmd.Commands() {
			found[cmd.Name()+" "+sub.Name()] = true
		}
	}

	for path := range want {
		if !found[path] {
			t.Errorf("missing command: %q", path)
		}
	}
}

func TestSkipAuthAnnotations(t *testing.T) {
	// Not parallel — rootCmd.Commands() sorts in place

	wantSkipAuth := map[string]bool{
		"login":      true,
		"status":     true,
		"logout":     true,
		"doctor":     true,
		"version":    true,
		"completion": true,
		"commands":   true,
	}

	// Check top-level commands
	for _, cmd := range rootCmd.Commands() {
		if wantSkipAuth[cmd.Name()] {
			if cmd.Annotations["skipAuth"] != "true" {
				t.Errorf("command %q should have skipAuth annotation", cmd.Name())
			}
		}
		// Check subcommands
		for _, sub := range cmd.Commands() {
			if wantSkipAuth[sub.Name()] {
				if sub.Annotations["skipAuth"] != "true" {
					t.Errorf("command %q %q should have skipAuth annotation", cmd.Name(), sub.Name())
				}
			}
		}
	}
}

func TestGlobalFlags(t *testing.T) {
	flags := []string{"format", "no-color", "api-url", "allow-custom-host"}
	for _, name := range flags {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing persistent flag: %q", name)
		}
	}
}
