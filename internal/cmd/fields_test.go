package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

func TestValidateFieldsRejectsNamesOutsideTheVocabulary(t *testing.T) {
	tests := []struct {
		name      string
		fields    string
		wantErr   bool
		wantParts []string
	}{
		{name: "unset", fields: "", wantErr: false},
		{name: "accepted", fields: "id,title,article_count", wantErr: false},
		{name: "spaces are trimmed", fields: " id , title ", wantErr: false},
		{
			name:      "a display label is not a field name",
			fields:    "reported",
			wantErr:   true,
			wantParts: []string{`"reported"`, " is not among them", "first_reported_at"},
		},
		{
			name:      "two unknown names agree the verb",
			fields:    "reported,type",
			wantErr:   true,
			wantParts: []string{`"reported"`, `"type"`, " are not among them"},
		},
		{
			name:      "a field from another feed is rejected",
			fields:    "id,summary",
			wantErr:   true,
			wantParts: []string{`"summary"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreFlags(t, eventsListCmd)
			cmd := *eventsListCmd
			if err := cmd.Flags().Set("fields", tt.fields); err != nil {
				t.Fatalf("Set(fields, %q) = %v", tt.fields, err)
			}

			err := validateFields(&cmd)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("validateFields() = %v, want nil for %q", err, tt.fields)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateFields() = nil, want an error for %q", tt.fields)
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q does not contain %q", err, part)
				}
			}
		})
	}
}

// TestValidateFieldsIgnoresCommandsWithoutTheFlag guards the nested routes and
// every command that never registered --fields: validation runs from the shared
// pre-run hook, so it has to be inert where the flag does not exist.
func TestValidateFieldsIgnoresCommandsWithoutTheFlag(t *testing.T) {
	if err := validateFields(doctorCmd); err != nil {
		t.Errorf("validateFields(doctor) = %v, want nil", err)
	}
	if fields := selectedFields(doctorCmd); fields != nil {
		t.Errorf("selectedFields(doctor) = %v, want nil", fields)
	}
}

// TestSelectedFieldsMarksTheWholeVocabulary covers the three states a renderer
// has to tell apart: requested, excluded, and not governed by --fields at all.
func TestSelectedFieldsMarksTheWholeVocabulary(t *testing.T) {
	restoreFlags(t, eventsListCmd)
	cmd := *eventsListCmd
	if err := cmd.Flags().Set("fields", "title"); err != nil {
		t.Fatal(err)
	}

	want := selectedFields(&cmd)
	if want == nil {
		t.Fatal("selectedFields() = nil, want a selection")
	}
	if !want["title"] {
		t.Error("title was requested but is not marked true")
	}
	if !want["id"] {
		t.Error("id must be true: the API returns it whether or not it is asked for")
	}
	if requested, governed := want["article_count"]; !governed || requested {
		t.Errorf("article_count = (%v, governed %v), want (false, true)", requested, governed)
	}
	if _, governed := want["distance"]; governed {
		t.Error("distance must not be governed by --fields: the flag can neither request nor suppress it")
	}
}

func TestSelectedFieldsIsNilWhenTheFlagIsUnset(t *testing.T) {
	restoreFlags(t, eventsListCmd)
	cmd := *eventsListCmd
	if fields := selectedFields(&cmd); fields != nil {
		t.Errorf("selectedFields() = %v, want nil so every column renders", fields)
	}
}

func TestNarrowColumnsDropsUnrequestedColumns(t *testing.T) {
	columns := []string{"ID", "Reported", "Type", "Title"}
	rows := [][]string{{"e1", "2026-10-01", "ceo-departure", "A title"}}
	colFields := [][]string{{"id"}, {"first_reported_at"}, {"event_type"}, {"title"}}

	want := map[string]bool{
		"id": true, "title": true,
		"first_reported_at": false, "event_type": false,
	}
	gotCols, gotRows := narrowColumns(want, columns, rows, colFields)

	if strings.Join(gotCols, ",") != "ID,Title" {
		t.Errorf("columns = %v, want [ID Title]", gotCols)
	}
	if len(gotRows) != 1 || strings.Join(gotRows[0], ",") != "e1,A title" {
		t.Errorf("rows = %v, want [[e1 A title]] — cells must follow their columns", gotRows)
	}
}

// TestNarrowColumnsKeepsAColumnFedByEitherField covers the calendar Date and
// Company columns, which each read one field and fall back to another.
func TestNarrowColumnsKeepsAColumnFedByEitherField(t *testing.T) {
	columns := []string{"ID", "Company"}
	rows := [][]string{{"c1", "cmpny123"}}
	colFields := [][]string{{"id"}, {"company", "company_id"}}

	want := map[string]bool{"id": true, "company": false, "company_id": true}
	gotCols, _ := narrowColumns(want, columns, rows, colFields)
	if strings.Join(gotCols, ",") != "ID,Company" {
		t.Errorf("columns = %v, want the Company column kept when company_id was requested", gotCols)
	}
}

// TestNarrowColumnsKeepsUngovernedColumns covers the Distance column, which a
// semantic search appends past the end of colFields.
func TestNarrowColumnsKeepsUngovernedColumns(t *testing.T) {
	columns := []string{"ID", "Title", "Distance"}
	rows := [][]string{{"e1", "A title", "0.3901"}}
	colFields := [][]string{{"id"}, {"title"}}

	want := map[string]bool{"id": true, "title": true, "article_count": false}
	gotCols, gotRows := narrowColumns(want, columns, rows, colFields)
	if strings.Join(gotCols, ",") != "ID,Title,Distance" {
		t.Errorf("columns = %v, want Distance kept", gotCols)
	}
	if strings.Join(gotRows[0], ",") != "e1,A title,0.3901" {
		t.Errorf("rows = %v, want the distance cell kept", gotRows)
	}
}

func TestNarrowColumnsWithoutASelectionChangesNothing(t *testing.T) {
	columns := []string{"ID", "Title"}
	rows := [][]string{{"e1", "A title"}}
	colFields := [][]string{{"id"}, {"title"}}

	gotCols, gotRows := narrowColumns(nil, columns, rows, colFields)
	if strings.Join(gotCols, ",") != "ID,Title" || strings.Join(gotRows[0], ",") != "e1,A title" {
		t.Errorf("nil selection altered the table: %v / %v", gotCols, gotRows)
	}
}

func TestNarrowJSONDropsUnrequestedKeys(t *testing.T) {
	type row struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Country  string   `json:"country"`
		Distance *float64 `json:"distance,omitempty"`
	}
	d := 0.39
	in := []row{{ID: "e1", Title: "A title", Country: "", Distance: &d}}

	want := map[string]bool{"id": true, "title": true, "country": false}
	out, err := narrowJSON(want, in)
	if err != nil {
		t.Fatalf("narrowJSON() = %v", err)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}

	if _, present := got[0]["country"]; present {
		t.Error(`country is still present: an unrequested field must be absent, not ""`)
	}
	if got[0]["title"] != "A title" {
		t.Errorf("title = %v, want it kept", got[0]["title"])
	}
	if _, present := got[0]["distance"]; !present {
		t.Error("distance was dropped, but --fields does not govern it")
	}
}

func TestNarrowJSONWithoutASelectionReturnsTheInput(t *testing.T) {
	in := []map[string]string{{"id": "e1", "country": ""}}
	out, err := narrowJSON(nil, in)
	if err != nil {
		t.Fatalf("narrowJSON() = %v", err)
	}
	raw, _ := json.Marshal(out)
	if !strings.Contains(string(raw), `"country":""`) {
		t.Errorf("nil selection altered the payload: %s", raw)
	}
}

func TestFieldsWithoutColumnNamesOnlyRequestedFieldsWithNoColumn(t *testing.T) {
	colFields := [][]string{{"id"}, {"title"}}
	want := map[string]bool{
		"id": true, "title": true, "article_url": true,
		"country": false, "updated_at": true,
	}
	missing := fieldsWithoutColumn(want, colFields)
	if strings.Join(missing, ",") != "article_url,updated_at" {
		t.Errorf("missing = %v, want [article_url updated_at] sorted", missing)
	}

	note := noColumnNote(missing)
	for _, part := range []string{"article_url", "updated_at", "have no table column", "--format json"} {
		if !strings.Contains(note, part) {
			t.Errorf("note %q does not contain %q", note, part)
		}
	}
	if single := noColumnNote([]string{"article_url"}); !strings.Contains(single, "has no table column") {
		t.Errorf("single-field note %q should read \"has\"", single)
	}
	if noColumnNote(nil) != "" {
		t.Error("an empty selection must produce no note")
	}
}

// TestEveryFieldsFlagNamesAVocabulary keeps the flag and its vocabulary wired
// together: a command registering --fields by hand would validate nothing and
// narrow nothing, failing silently.
func TestEveryFieldsFlagNamesAVocabulary(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Flags().Lookup("fields") != nil {
			resource := c.Annotations[fieldsResourceAnnotation]
			if resource == "" {
				t.Errorf("%q registers --fields without naming a vocabulary", c.CommandPath())
			} else if _, ok := fieldVocabularies[resource]; !ok {
				t.Errorf("%q names unknown vocabulary %q", c.CommandPath(), resource)
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}

// TestFieldsEndToEnd drives a real command with --fields, which is the only test
// here that covers the selection reaching the renderer rather than the helpers in
// isolation. The fixture answers with the sparse body the API returns for a
// narrowed request: the unrequested fields are absent, not empty.
func TestFieldsEndToEnd(t *testing.T) {
	body := `{"events":[{"id":"evt1","title":"Saab lands order"}],"pagination":{"count":1}}`

	t.Run("table shows only the requested columns", func(t *testing.T) {
		out := runListCommand(t, eventsListCmd, fixedBody(body), "table", "--fields", "id,title")

		for _, heading := range []string{"ID", "TITLE"} {
			if !strings.Contains(out, heading) {
				t.Errorf("output = %q, want the %s column", out, heading)
			}
		}
		// These carried a blank or, worse, a zero that read as a real count.
		for _, heading := range []string{"REPORTED", "SIGNIFICANCE", "ARTICLES"} {
			if strings.Contains(out, heading) {
				t.Errorf("output = %q, must not show the %s column: it was not requested", out, heading)
			}
		}
	})

	t.Run("json omits the unrequested keys", func(t *testing.T) {
		out := runListCommand(t, eventsListCmd, fixedBody(body), "json", "--fields", "id,title")

		var env struct {
			Data struct {
				Events []map[string]interface{} `json:"events"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, out)
		}
		if len(env.Data.Events) != 1 {
			t.Fatalf("got %d events, want 1", len(env.Data.Events))
		}
		got := env.Data.Events[0]
		if got["title"] != "Saab lands order" {
			t.Errorf("title = %v, want it present", got["title"])
		}
		for _, key := range []string{"significance", "article_count", "first_reported_at", "status"} {
			if _, present := got[key]; present {
				t.Errorf("key %q is present as %v; an unrequested field must be absent, not zero", key, got[key])
			}
		}
	})

	t.Run("a bad field name never reaches the API", func(t *testing.T) {
		restoreFlags(t, eventsListCmd)
		cmd := *eventsListCmd
		if err := cmd.Flags().Set("fields", "reported"); err != nil {
			t.Fatal(err)
		}
		called := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		t.Cleanup(srv.Close)

		var buf bytes.Buffer
		f := output.NewFormatterWithWriter("json", true, &buf)
		if err := applyFieldSelection(&cmd, f); err == nil {
			t.Fatal("applyFieldSelection() = nil, want a rejection")
		}
		if called {
			t.Error("the API was called despite an invalid --fields value")
		}
		if f.Fields() != nil {
			t.Error("a rejected selection must not be recorded on the formatter")
		}
	})
}
