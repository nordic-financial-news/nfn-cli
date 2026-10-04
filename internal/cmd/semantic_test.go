package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/spf13/cobra"
)

// semanticTestCmd builds a command carrying the flags validateSemantic looks
// at, so the checks can be exercised without a live command tree.
func semanticTestCmd(withUpdatedAfter bool) *cobra.Command {
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().String("q", "", "")
	cmd.Flags().String("cursor", "", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("sort", "", "")
	if withUpdatedAfter {
		cmd.Flags().String("updated-after", "", "")
	}
	addSemanticFlag(cmd)
	return cmd
}

func TestValidateSemantic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		withUpdatedAfter bool
		set              map[string]string
		wantErr          string
	}{
		{
			name: "not requested, nothing is checked",
			set:  map[string]string{"cursor": "abc"},
		},
		{
			name: "valid semantic request",
			set:  map[string]string{"semantic": "true", "q": "defence contract wins"},
		},
		{
			name:    "semantic without a query",
			set:     map[string]string{"semantic": "true"},
			wantErr: "needs a query",
		},
		{
			name:    "semantic with a blank query",
			set:     map[string]string{"semantic": "true", "q": "   "},
			wantErr: "needs a query",
		},
		{
			name:    "query below the minimum length",
			set:     map[string]string{"semantic": "true", "q": "a"},
			wantErr: "between 2 and 200 characters",
		},
		{
			name:    "query above the maximum length",
			set:     map[string]string{"semantic": "true", "q": strings.Repeat("x", 201)},
			wantErr: "between 2 and 200 characters",
		},
		{
			name: "query at the maximum length is accepted",
			set:  map[string]string{"semantic": "true", "q": strings.Repeat("x", 200)},
		},
		{
			// Length is counted in runes, so a 200-character Nordic query is
			// not rejected for its byte count.
			name: "multibyte query counted in runes",
			set:  map[string]string{"semantic": "true", "q": strings.Repeat("ä", 200)},
		},
		{
			name:    "semantic with a cursor",
			set:     map[string]string{"semantic": "true", "q": "battery plants", "cursor": "abc"},
			wantErr: "--cursor cannot be combined with --semantic",
		},
		{
			// --sort is refused for a different reason than the paging flags:
			// semantic mode ranks by closeness, so an explicit order would
			// override the ranking it exists to produce. The API rejects the
			// pair outright, so catching it here saves a round trip.
			name:    "semantic with sort",
			set:     map[string]string{"semantic": "true", "q": "battery plants", "sort": "latest"},
			wantErr: "--sort cannot be combined with --semantic",
		},
		{
			name:    "semantic with sort explains the ranking clash",
			set:     map[string]string{"semantic": "true", "q": "battery plants", "sort": "latest"},
			wantErr: "defines its own ranking",
		},
		{
			name:    "semantic with all",
			set:     map[string]string{"semantic": "true", "q": "battery plants", "all": "true"},
			wantErr: "--all cannot be combined with --semantic",
		},
		{
			name:             "semantic with updated-after",
			withUpdatedAfter: true,
			set:              map[string]string{"semantic": "true", "q": "battery plants", "updated-after": "2026-01-01T00:00:00Z"},
			wantErr:          "--updated-after cannot be combined with --semantic",
		},
		{
			// companies has neither --updated-after nor --sort; the check must
			// skip a flag the command never registered rather than panic.
			name: "missing updated-after flag is skipped",
			set:  map[string]string{"semantic": "true", "q": "battery plants"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := semanticTestCmd(tt.withUpdatedAfter)
			for k, v := range tt.set {
				if err := cmd.Flags().Set(k, v); err != nil {
					t.Fatalf("setting %s: %v", k, err)
				}
			}

			err := validateSemantic(cmd)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateSemantic() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateSemantic() = nil, want an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateSemantic() = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// An unset --all or --cursor must not trip the conflict check; only a flag the
// caller actually passed counts.
func TestValidateSemanticIgnoresUnsetFlags(t *testing.T) {
	t.Parallel()

	cmd := semanticTestCmd(true)
	_ = cmd.Flags().Set("semantic", "true")
	_ = cmd.Flags().Set("q", "grid capacity investment")

	if err := validateSemantic(cmd); err != nil {
		t.Errorf("validateSemantic() = %v, want nil", err)
	}
}

func TestSemanticError(t *testing.T) {
	t.Parallel()

	unavailable := &api.APIError{
		StatusCode: 503,
		Type:       "https://nordicfinancialnews.com/problems/semantic-unavailable",
		Detail:     "Semantic search unavailable",
	}
	rateLimited := &api.APIError{StatusCode: 429, Detail: "Too many requests", RetryAfter: 30 * time.Minute}
	forbidden := &api.APIError{StatusCode: 403, Detail: "Forbidden"}
	notRolledOut := &api.APIError{StatusCode: 400, Detail: "Unknown parameter 'mode'"}
	badQuery := &api.APIError{StatusCode: 400, Detail: "q must be between 2 and 200 characters"}

	tests := []struct {
		name     string
		err      error
		semantic bool
		want     string
	}{
		{name: "nil error", err: nil, semantic: true, want: ""},
		{
			name:     "not a semantic request, passes through",
			err:      unavailable,
			semantic: false,
			want:     "Semantic search unavailable",
		},
		{
			name:     "semantic unavailable is explained",
			err:      unavailable,
			semantic: true,
			want:     "drop --semantic",
		},
		{
			name:     "rate limit mentions the semantic budget",
			err:      rateLimited,
			semantic: true,
			want:     "120 per hour",
		},
		{
			name:     "rate limit reports the retry window",
			err:      rateLimited,
			semantic: true,
			want:     "retry in 30m0s",
		},
		{
			// Semantic rolls out per key; a key outside it must be told to drop
			// --semantic, not shown a parameter name it never typed.
			name:     "key outside the rollout gets an actionable message",
			err:      notRolledOut,
			semantic: true,
			want:     "not enabled for this API key yet",
		},
		{
			name:     "rollout message names the flag the user typed",
			err:      notRolledOut,
			semantic: true,
			want:     "Drop --semantic",
		},
		{
			name:     "an ordinary 400 is not mistaken for the rollout",
			err:      badQuery,
			semantic: true,
			want:     "q must be between 2 and 200 characters",
		},
		{
			name:     "unrelated API errors pass through",
			err:      forbidden,
			semantic: true,
			want:     "Forbidden",
		},
		{
			name:     "non-API errors pass through",
			err:      errors.New("dial tcp: connection refused"),
			semantic: true,
			want:     "dial tcp: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := semanticError(tt.err, tt.semantic)
			if tt.want == "" {
				if got != nil {
					t.Fatalf("semanticError() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("semanticError() = nil, want an error containing %q", tt.want)
			}
			if !strings.Contains(got.Error(), tt.want) {
				t.Errorf("semanticError() = %q, want it to contain %q", got.Error(), tt.want)
			}
		})
	}
}

// A 429 with no Retry-After still explains the budget, just without a window.
func TestSemanticErrorWithoutRetryAfter(t *testing.T) {
	t.Parallel()

	err := semanticError(&api.APIError{StatusCode: 429, Detail: "Too many requests"}, true)
	if err == nil {
		t.Fatal("semanticError() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "120 per hour") {
		t.Errorf("semanticError() = %q, want it to mention the semantic budget", err.Error())
	}
	if strings.Contains(err.Error(), "retry in") {
		t.Errorf("semanticError() = %q, want no retry window when none was advertised", err.Error())
	}
}

func TestFormatDistance(t *testing.T) {
	t.Parallel()

	if got := formatDistance(nil); got != "" {
		t.Errorf("formatDistance(nil) = %q, want an empty string", got)
	}

	tests := []struct {
		value float64
		want  string
	}{
		{value: 0, want: "0.0000"},
		{value: 0.1842, want: "0.1842"},
		{value: 1, want: "1.0000"},
	}
	for _, tt := range tests {
		v := tt.value
		if got := formatDistance(&v); got != tt.want {
			t.Errorf("formatDistance(%v) = %q, want %q", v, got, tt.want)
		}
	}
}

// The four list commands offering --semantic must all forward mode=semantic,
// and must not set it when the flag is absent.
func TestSemanticParamMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		build    func(*cobra.Command) map[string][]string
		register func(*cobra.Command)
		source   *cobra.Command
	}{
		{
			name:   "articles",
			source: articlesListCmd,
			register: func(c *cobra.Command) {
				c.Flags().String("q", "", "")
				c.Flags().Bool("semantic", false, "")
			},
			build: func(c *cobra.Command) map[string][]string { return buildArticleParams(c) },
		},
		{
			name:   "stories",
			source: storiesListCmd,
			register: func(c *cobra.Command) {
				c.Flags().String("q", "", "")
				c.Flags().Bool("semantic", false, "")
			},
			build: func(c *cobra.Command) map[string][]string { return buildStoryParams(c) },
		},
		{
			name:   "companies",
			source: companiesListCmd,
			register: func(c *cobra.Command) {
				c.Flags().String("q", "", "")
				c.Flags().Bool("semantic", false, "")
			},
			build: func(c *cobra.Command) map[string][]string { return buildCompanyParams(c) },
		},
		{
			name:   "events",
			source: eventsListCmd,
			register: func(c *cobra.Command) {
				c.Flags().String("q", "", "")
				c.Flags().Bool("semantic", false, "")
			},
			build: func(c *cobra.Command) map[string][]string { return buildEventParams(c) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" sets mode when semantic", func(t *testing.T) {
			t.Parallel()
			cmd := *tt.source
			cmd.ResetFlags()
			tt.register(&cmd)
			_ = cmd.Flags().Set("q", "battery gigafactory")
			_ = cmd.Flags().Set("semantic", "true")

			params := tt.build(&cmd)
			if got := params["mode"]; len(got) != 1 || got[0] != "semantic" {
				t.Errorf("params[mode] = %v, want [semantic]", got)
			}
			if got := params["q"]; len(got) != 1 || got[0] != "battery gigafactory" {
				t.Errorf("params[q] = %v, want [battery gigafactory]", got)
			}
		})

		t.Run(tt.name+" omits mode without semantic", func(t *testing.T) {
			t.Parallel()
			cmd := *tt.source
			cmd.ResetFlags()
			tt.register(&cmd)
			_ = cmd.Flags().Set("q", "battery gigafactory")

			params := tt.build(&cmd)
			if _, ok := params["mode"]; ok {
				t.Errorf("params[mode] = %v, want it to be absent", params["mode"])
			}
		})
	}
}
