package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nordic-financial-news/nfn-cli/internal/api"
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

// TestSelectedFieldsReportsOnlyWhatWasAskedFor pins the contract the three
// consumers read: the map holds the caller's request and nothing else, so no
// consumer can mistake a field the API happens to return for one that was asked
// about.
func TestSelectedFieldsReportsOnlyWhatWasAskedFor(t *testing.T) {
	restoreFlags(t, eventsListCmd)
	cmd := *eventsListCmd
	if err := cmd.Flags().Set("fields", "title, significance ,"); err != nil {
		t.Fatal(err)
	}

	want := selectedFields(&cmd)
	if want == nil {
		t.Fatal("selectedFields() = nil, want a selection")
	}
	if len(want) != 2 || !want["title"] || !want["significance"] {
		t.Errorf("selectedFields() = %v, want exactly title and significance, whitespace trimmed", want)
	}
	if want["id"] {
		t.Error("id must not be marked requested: nobody asked for it")
	}
	if want["article_count"] {
		t.Error("an unrequested vocabulary field must not be marked requested")
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

// TestNarrowColumnsKeepsAColumnFedByEitherField covers the calendar When and
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

// TestNarrowJSONDropsKeysOutsideTheVocabulary covers the shape that made a
// narrowed payload lie: a struct field the vocabulary does not name and the json
// tag does not mark omitempty. The API projected it away, so Go's zero value is
// all that is left, and emitting it asserts that the company is inactive and has
// no exchange.
func TestNarrowJSONDropsKeysOutsideTheVocabulary(t *testing.T) {
	type row struct {
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		IsActive bool      `json:"is_active"`
		Exchange *struct{} `json:"exchange"`
		Distance *float64  `json:"distance,omitempty"`
	}
	d := 0.39
	in := []row{{ID: "c1", Name: "Volvo", Distance: &d}}

	out, err := narrowJSON(map[string]bool{"name": true}, in)
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

	for _, key := range []string{"is_active", "exchange"} {
		if _, present := got[0][key]; present {
			t.Errorf("%s survived narrowing, asserting a value the API never returned", key)
		}
	}
	if got[0]["name"] != "Volvo" {
		t.Errorf("name = %v, want it kept", got[0]["name"])
	}
	if got[0]["id"] != "c1" {
		t.Error("id was dropped, but the API returns it whether or not it was asked for")
	}
	if _, present := got[0]["distance"]; !present {
		t.Error("distance was dropped: the API merges it after projecting, so --fields cannot suppress it")
	}
}

// TestNarrowColumnsKeepsTheIDColumnUnrequested: id comes back either way and is
// the argument every get command takes, so its column is worth keeping.
func TestNarrowColumnsKeepsTheIDColumnUnrequested(t *testing.T) {
	columns := []string{"ID", "Title", "Reported"}
	rows := [][]string{{"e1", "A title", "2026-10-01"}}
	colFields := [][]string{{"id"}, {"title"}, {"first_reported_at"}}

	gotCols, gotRows := narrowColumns(map[string]bool{"title": true}, columns, rows, colFields)
	if strings.Join(gotCols, ",") != "ID,Title" {
		t.Errorf("columns = %v, want [ID Title]", gotCols)
	}
	if strings.Join(gotRows[0], ",") != "e1,A title" {
		t.Errorf("rows = %v, want the id cell kept", gotRows)
	}
}

// TestNarrowColumnsPadsShortRows: a row shorter than the header set keeps a blank
// in place, so the cells after it stay under their own headers instead of sliding
// left under the wrong ones.
func TestNarrowColumnsPadsShortRows(t *testing.T) {
	columns := []string{"ID", "Title", "Status", "Reported"}
	rows := [][]string{{"e1", "A title", "confirmed", "2026-10-01"}, {"e2"}}
	colFields := [][]string{{"id"}, {"title"}, {"status"}, {"first_reported_at"}}

	gotCols, gotRows := narrowColumns(map[string]bool{"title": true, "status": true}, columns, rows, colFields)
	if strings.Join(gotCols, ",") != "ID,Title,Status" {
		t.Fatalf("columns = %v, want Reported dropped so the pad path is reached", gotCols)
	}
	if len(gotRows[1]) != 3 {
		t.Fatalf("short row = %v, want 3 cells to match the 3 kept columns", gotRows[1])
	}
	if gotRows[1][0] != "e2" || gotRows[1][1] != "" || gotRows[1][2] != "" {
		t.Errorf("short row = %v, want [e2 \"\" \"\"]", gotRows[1])
	}
}

// TestFieldsWithoutColumnReportsIDWhenNoColumnShowsIt covers nfn event-types,
// whose table is Slug/Label/Category/Summary. --fields id narrows that to nothing,
// and without the note the caller gets an empty table and no reason for it.
func TestFieldsWithoutColumnReportsIDWhenNoColumnShowsIt(t *testing.T) {
	colFields := [][]string{{"slug"}, {"label"}, {"category"}, {"summary"}}

	missing := fieldsWithoutColumn(map[string]bool{"id": true}, colFields)
	if strings.Join(missing, ",") != "id" {
		t.Errorf("missing = %v, want [id]: this table has no id column", missing)
	}
	if note := noColumnNote(missing); !strings.Contains(note, "id has no table column") {
		t.Errorf("note = %q, want it to name id", note)
	}

	// The reverse: an unrequested id must never be named, or every narrowed
	// event-types table would report a field nobody asked about.
	if m := fieldsWithoutColumn(map[string]bool{"label": true}, colFields); len(m) != 0 {
		t.Errorf("missing = %v, want none: id was not requested", m)
	}
}

// TestEveryVocabularyFieldCanSurviveTheRoundTrip keeps each vocabulary and the
// struct it narrows in step. A name the API accepts but the struct has no json
// tag for is dropped on unmarshal, so --fields would advertise a field and then
// return a payload without it.
func TestEveryVocabularyFieldCanSurviveTheRoundTrip(t *testing.T) {
	structs := map[string]interface{}{
		"article":        api.Article{},
		"story":          api.Story{},
		"event":          api.Event{},
		"event_type":     api.EventTypeDetail{},
		"calendar_event": api.CalendarEvent{},
		"company":        api.Company{},
		"source":         api.Source{},
		"exchange":       api.ExchangeDetail{},
		"index":          api.StockIndex{},
	}
	for resource := range fieldVocabularies {
		if _, ok := structs[resource]; !ok {
			t.Errorf("vocabulary %q has no struct listed here, so nothing checks it", resource)
		}
	}

	for resource, v := range structs {
		tags := map[string]bool{}
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
			if name != "" && name != "-" {
				tags[name] = true
			}
		}
		for _, field := range fieldVocabularies[resource] {
			if !tags[field] {
				t.Errorf("%s vocabulary offers %q, but %T has no json tag for it, so the value is dropped",
					resource, field, v)
			}
		}
	}
}

// TestEveryFieldsCommandNarrowsItsJSON is the structural guard: a renderer that
// narrows its table but forgets narrowJSON looks right in a terminal and keeps
// emitting zero values to the agents the envelope exists for. Three renderers
// shipped that way. The mock returns id plus one field, so every other key in
// the payload can only be a zero value Go filled in.
func TestEveryFieldsCommandNarrowsItsJSON(t *testing.T) {
	tests := []struct {
		name       string
		cmd        *cobra.Command
		collection string
		fields     string
		// extra is the one requested field beyond id, and the value the mock returns.
		extraKey, extraVal string
	}{
		{"articles", articlesListCmd, "articles", "id,title", "title", "Saab lands order"},
		{"stories", storiesListCmd, "stories", "id,title", "title", "A story"},
		{"events", eventsListCmd, "events", "id,title", "title", "An event"},
		{"event-types", eventTypesCmd, "event_types", "id,label", "label", "CEO Departure"},
		{"calendar", calendarListCmd, "calendar_events", "id,title", "title", "Q3 report"},
		{"companies", companiesListCmd, "companies", "id,name", "name", "Volvo"},
		{"sources", sourcesListCmd, "sources", "id,name", "name", "Yle"},
		{"exchanges", exchangesListCmd, "exchanges", "id,name", "name", "Nasdaq Stockholm"},
		{"indices", indicesListCmd, "indices", "id,name", "name", "OMXS30"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			row, err := json.Marshal(map[string]string{"id": "x1", tc.extraKey: tc.extraVal})
			if err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(`{%q:[%s],"pagination":{"count":1}}`, tc.collection, row)
			out := runListCommand(t, tc.cmd, fixedBody(body), "json", "--fields", tc.fields)

			var env struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal([]byte(out), &env); err != nil {
				t.Fatalf("output is not valid JSON: %v\ngot:\n%s", err, out)
			}
			var rows []map[string]interface{}
			if err := json.Unmarshal(env.Data[tc.collection], &rows); err != nil {
				t.Fatalf("decoding %s: %v\ngot:\n%s", tc.collection, err, out)
			}
			if len(rows) != 1 {
				t.Fatalf("got %d rows, want 1", len(rows))
			}

			requested := map[string]bool{"id": true, tc.extraKey: true}
			for key, value := range rows[0] {
				if !requested[key] && !passThroughJSONKeys[key] {
					t.Errorf("key %q is present as %v; it was not requested, so it can only be a zero value", key, value)
				}
			}
			if rows[0][tc.extraKey] != tc.extraVal {
				t.Errorf("%s = %v, want %q kept", tc.extraKey, rows[0][tc.extraKey], tc.extraVal)
			}
		})
	}
}

// TestArticleAndStoryVocabulariesAcceptEventID: the OpenAPI fields description
// for both feeds omits event_id, but the server's ALLOWED_ARTICLE_FIELDS and
// ALLOWED_STORY_FIELDS accept it. The vocabularies were built from the spec and
// inherited the omission, which made the CLI reject a field the API serves.
// Trimming either vocabulary back to the spec would reintroduce that.
func TestArticleAndStoryVocabulariesAcceptEventID(t *testing.T) {
	for _, resource := range []string{"article", "story"} {
		found := false
		for _, name := range fieldVocabularies[resource] {
			if name == "event_id" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s vocabulary is missing event_id, which the API accepts", resource)
		}
	}
}
