package cmd

import (
	"strings"
	"testing"
)

// Search ranks each section by relevance, so a dated section says so rather
// than reading as a broken date sort.
func TestSearch_RankingNote(t *testing.T) {
	body := `{"results":{"companies":[{"id":"h22z9xvlw07k","name":"CTEK AB","ticker":"CTEK"}],` +
		`"articles":[{"id":"lqaylm8mqb6y","title":"Konecranes order","published_at":"2026-03-19T09:00:00Z"}],` +
		`"stories":[{"id":"yptbjb4ovfgb","title":"Lyten completes acquisition","published_at":"2026-02-27T07:09:00Z"}]}}`

	out := runCommandWithArgs(t, searchCmd, fixedBody(body), "table", []string{"battery"})

	for _, want := range []string{"1 article (ranked by relevance)", "1 story (ranked by relevance)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "1 company (ranked") {
		t.Errorf("the undated companies section carries a ranking note\n%s", out)
	}
}
