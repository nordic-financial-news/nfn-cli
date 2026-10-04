package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// restoreFlags resets a command's flags to their defaults once the test ends.
// Copying a cobra.Command does NOT isolate its flags — the copy shares the
// original's *pflag.FlagSet — so parsing flags below mutates the package-level
// command for the rest of the test binary.
func restoreFlags(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	t.Cleanup(func() {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	})
}

// fixedBody serves the same JSON body for every request.
func fixedBody(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

// runListCommand drives a real list command end to end: the command's RunE
// calls the real API client against a test server, so this covers the whole
// path from HTTP response through ListPage to the rendered output. It is the
// seam that the api-layer and renderer-layer tests each cover only one side of.
//
// These tests must not call t.Parallel: they mutate shared cobra flag state.
func runListCommand(t *testing.T, cmd *cobra.Command, handler http.HandlerFunc, format string, args ...string) string {
	t.Helper()
	return runCommandWithArgs(t, cmd, handler, format, nil, args...)
}

// runCommandWithArgs is runListCommand for a command that takes positional
// arguments, such as a detail route or a nested company route.
func runCommandWithArgs(t *testing.T, cmd *cobra.Command, handler http.HandlerFunc, format string, cmdArgs []string, args ...string) string {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	ctx := context.WithValue(context.Background(), formatterKey, output.NewFormatterWithWriter(format, true, &buf))
	ctx = context.WithValue(ctx, clientKey, api.NewClient("key", api.WithBaseURL(srv.URL)))

	restoreFlags(t, cmd)

	c := *cmd
	c.SetContext(ctx)
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("parsing flags: %v", err)
	}
	if err := c.RunE(&c, cmdArgs); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	return buf.String()
}

func TestStoriesList_PlanLimitedEndToEnd(t *testing.T) {
	body := `{"stories":[{"id":"s1","title":"Capped story","country":"SE","article_count":1,` +
		`"published_at":"2026-08-07T00:00:00Z"}],"pagination":{"count":1},"plan_limited":true}`

	t.Run("json surfaces the flag", func(t *testing.T) {
		out := runListCommand(t, storiesListCmd, fixedBody(body), "json", "--limit", "25")

		var env struct {
			Summary string `json:"summary"`
			Data    struct {
				Stories     []api.Story `json:"stories"`
				PlanLimited *bool       `json:"plan_limited"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, out)
		}
		if len(env.Data.Stories) != 1 {
			t.Fatalf("got %d stories, want 1", len(env.Data.Stories))
		}
		if env.Data.PlanLimited == nil || !*env.Data.PlanLimited {
			t.Fatalf("data.plan_limited = %v, want true\ngot:\n%s", env.Data.PlanLimited, out)
		}
		if !strings.Contains(env.Summary, "limited by your plan") {
			t.Errorf("summary = %q, want it to mention the plan limit", env.Summary)
		}
	})

	t.Run("table shows the note", func(t *testing.T) {
		out := runListCommand(t, storiesListCmd, fixedBody(body), "table", "--limit", "25")
		if !strings.Contains(out, planLimitedNote) {
			t.Errorf("table output missing the plan-limited note\ngot:\n%s", out)
		}
	})

	t.Run("unrestricted response stays clean", func(t *testing.T) {
		plain := `{"stories":[{"id":"s1","title":"A story"}],"pagination":{"count":1}}`
		out := runListCommand(t, storiesListCmd, fixedBody(plain), "json", "--limit", "25")
		if strings.Contains(out, "plan_limited") {
			t.Errorf("plan_limited should be absent\ngot:\n%s", out)
		}
	})
}

func TestArticlesList_PlanLimitedEndToEnd(t *testing.T) {
	body := `{"articles":[{"id":"a1","title":"Capped article","country":"SE"}],` +
		`"pagination":{"count":1},"plan_limited":true}`
	out := runListCommand(t, articlesListCmd, fixedBody(body), "json", "--limit", "25")

	if !strings.Contains(out, `"plan_limited": true`) {
		t.Errorf("articles output missing plan_limited\ngot:\n%s", out)
	}
}

func TestCalendarList_PlanLimitedEndToEnd(t *testing.T) {
	body := `{"calendar_events":[{"id":"c1","title":"Capped event","event_type":"earnings_report",` +
		`"status":"scheduled","company_id":"co1"}],"pagination":{"count":1},"plan_limited":true}`
	out := runListCommand(t, calendarListCmd, fixedBody(body), "json", "--limit", "25")

	if !strings.Contains(out, `"plan_limited": true`) {
		t.Errorf("calendar output missing plan_limited\ngot:\n%s", out)
	}
}

// --all follows cursors; the flag must survive aggregation across pages even
// when the final page doesn't repeat it. Live free-plan responses behave this
// way: the cap sits inside pagination rather than terminating it.
func TestStoriesList_PlanLimitedEndToEndWithAll(t *testing.T) {
	var page int
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page++
		if page == 1 {
			_, _ = w.Write([]byte(`{"stories":[{"id":"s1"}],"pagination":{"count":1,"next_cursor":"n"},"plan_limited":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"stories":[{"id":"s2"}],"pagination":{"count":1}}`))
	}

	out := runListCommand(t, storiesListCmd, handler, "json", "--all")

	if !strings.Contains(out, `"plan_limited": true`) {
		t.Errorf("--all output missing plan_limited\ngot:\n%s", out)
	}
	if page < 2 {
		t.Errorf("expected --all to fetch multiple pages, got %d", page)
	}
}
