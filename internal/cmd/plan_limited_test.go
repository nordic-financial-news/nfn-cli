package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
	"github.com/nordic-financial-news/nfn-cli/internal/output"
)

func planLimitedPtr(b bool) *bool { return &b }

// planLimitedEnvelope is the subset of the JSON envelope these tests assert on.
type planLimitedEnvelope struct {
	Summary string `json:"summary"`
	Data    struct {
		PlanLimited *bool `json:"plan_limited"`
	} `json:"data"`
}

func decodePlanLimitedEnvelope(t *testing.T, b []byte) planLimitedEnvelope {
	t.Helper()
	var env planLimitedEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, string(b))
	}
	return env
}

// renderFn adapts each list renderer to a common shape so every command that
// can receive plan_limited is covered by the same assertions.
type renderFn func(f *output.Formatter, planLimited *bool) error

func planLimitedRenderers() map[string]renderFn {
	return map[string]renderFn{
		"stories": func(f *output.Formatter, pl *bool) error {
			return renderStories(f, []api.Story{{ID: "s1", Title: "A story"}}, nil, pl, listOpts{})
		},
		"articles": func(f *output.Formatter, pl *bool) error {
			return renderArticles(f, []api.Article{{ID: "a1", Title: "An article"}}, nil, pl, listOpts{})
		},
		"calendar events": func(f *output.Formatter, pl *bool) error {
			return renderCalendarEvents(f, []api.CalendarEvent{{ID: "c1", Title: "An event"}}, nil, pl, false)
		},
		"companies": func(f *output.Formatter, pl *bool) error {
			return renderCompanies(f, []api.Company{{ID: "co1", Name: "A company"}}, nil, pl, listOpts{})
		},
	}
}

func TestRenderers_PlanLimitedJSON(t *testing.T) {
	t.Parallel()

	for name, render := range planLimitedRenderers() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			f := output.NewFormatterWithWriter("json", true, &buf)

			if err := render(f, planLimitedPtr(true)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			env := decodePlanLimitedEnvelope(t, buf.Bytes())
			if env.Data.PlanLimited == nil {
				t.Fatalf("data.plan_limited is missing\ngot:\n%s", buf.String())
			}
			if !*env.Data.PlanLimited {
				t.Errorf("data.plan_limited = false, want true")
			}
			if !strings.Contains(env.Summary, "limited by your plan") {
				t.Errorf("summary = %q, want it to mention the plan limit", env.Summary)
			}
		})
	}
}

// The API omits plan_limited when nothing was restricted, so the CLI must omit
// it too — emitting false would claim the result is complete on endpoints that
// never report the field.
func TestRenderers_PlanLimitedAbsentIsOmitted(t *testing.T) {
	t.Parallel()

	for name, render := range planLimitedRenderers() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			f := output.NewFormatterWithWriter("json", true, &buf)

			if err := render(f, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if strings.Contains(buf.String(), "plan_limited") {
				t.Errorf("plan_limited should be absent from output\ngot:\n%s", buf.String())
			}
			env := decodePlanLimitedEnvelope(t, buf.Bytes())
			if strings.Contains(env.Summary, "limited by your plan") {
				t.Errorf("summary = %q, should not mention a plan limit", env.Summary)
			}
		})
	}
}

func TestRenderers_PlanLimitedFalseIsEmitted(t *testing.T) {
	t.Parallel()

	for name, render := range planLimitedRenderers() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			f := output.NewFormatterWithWriter("json", true, &buf)

			if err := render(f, planLimitedPtr(false)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			env := decodePlanLimitedEnvelope(t, buf.Bytes())
			if env.Data.PlanLimited == nil {
				t.Fatalf("data.plan_limited is missing, want explicit false\ngot:\n%s", buf.String())
			}
			if *env.Data.PlanLimited {
				t.Errorf("data.plan_limited = true, want false")
			}
			if strings.Contains(env.Summary, "limited by your plan") {
				t.Errorf("summary = %q, should not mention a plan limit", env.Summary)
			}
		})
	}
}

func TestRenderers_PlanLimitedTableNote(t *testing.T) {
	t.Parallel()

	for name, render := range planLimitedRenderers() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var limited bytes.Buffer
			if err := render(output.NewFormatterWithWriter("table", true, &limited), planLimitedPtr(true)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(limited.String(), planLimitedNote) {
				t.Errorf("table output missing the plan-limited note\ngot:\n%s", limited.String())
			}

			var unlimited bytes.Buffer
			if err := render(output.NewFormatterWithWriter("table", true, &unlimited), nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(unlimited.String(), planLimitedNote) {
				t.Errorf("table output should not show the note when unset\ngot:\n%s", unlimited.String())
			}
		})
	}
}

func TestPlanLimitedFrom(t *testing.T) {
	t.Parallel()

	if got := planLimitedFrom(nil); got != nil {
		t.Errorf("planLimitedFrom(nil) = %v, want nil", *got)
	}
	if got := planLimitedFrom(&api.Response{}); got != nil {
		t.Errorf("planLimitedFrom(empty response) = %v, want nil", *got)
	}
	got := planLimitedFrom(&api.Response{PlanLimited: planLimitedPtr(true)})
	if got == nil || !*got {
		t.Errorf("planLimitedFrom(limited response) = %v, want true", got)
	}
}

func TestIsPlanLimited(t *testing.T) {
	t.Parallel()

	if isPlanLimited(nil) {
		t.Error("isPlanLimited(nil) = true, want false")
	}
	if isPlanLimited(planLimitedPtr(false)) {
		t.Error("isPlanLimited(false) = true, want false")
	}
	if !isPlanLimited(planLimitedPtr(true)) {
		t.Error("isPlanLimited(true) = false, want true")
	}
}
