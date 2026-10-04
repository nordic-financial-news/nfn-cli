package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nordic-financial-news/nfn-cli/internal/output"
	"github.com/spf13/cobra"
)

// fieldVocabularies records the field names each feed's --fields accepts, taken
// from the OpenAPI description of that endpoint's fields parameter. Every name
// here was checked against the API, and nothing outside these sets is accepted,
// so rejecting an unknown name locally spends no request on a typo and lets
// --help enumerate the vocabulary instead of leaving a 400 to teach it.
//
// The events feed is the one endpoint whose spec description does not enumerate
// its fields; the set below is its EventSummary schema, confirmed field by field
// against the live API.
var fieldVocabularies = map[string][]string{
	"article": {
		"id", "title", "article_url", "published_at", "updated_at",
		"content_type", "source", "company_ids", "country", "category",
	},
	"story": {
		"id", "title", "summary", "published_at", "updated_at",
		"article_count", "article_ids", "country", "category", "company_ids",
	},
	"event": {
		"id", "title", "status", "significance", "event_type", "article_count",
		"first_reported_at", "last_article_at", "updated_at", "company_ids",
		"story_ids", "merged_into", "related_to",
	},
	"event_type": {"id", "slug", "label", "category", "summary"},
	"calendar_event": {
		"id", "title", "description", "event_type", "status", "date_precision",
		"fiscal_period", "scheduled_at", "timezone", "local_time",
		"scheduled_at_changed_at", "source_article_id", "company_id", "company",
		"country", "dividend", "updated_at",
	},
	"company":  {"id", "name", "slug", "ticker", "former_tickers"},
	"source":   {"id", "name", "domain", "country", "source_type"},
	"exchange": {"id", "name", "mic_code", "acronym", "country"},
	"index":    {"id", "name", "symbol", "pan_nordic", "exchange"},
}

// fieldsResourceAnnotation remembers which vocabulary a command's --fields
// belongs to. The flag is registered per command, so the annotation travels with
// it and both validation and the selection a render call reads find the right set.
const fieldsResourceAnnotation = "fieldsResource"

// addFieldsFlagFor registers --fields for a command and names the feed whose
// vocabulary it takes, so the help text can list the accepted names.
func addFieldsFlagFor(cmd *cobra.Command, resource string) {
	vocab, ok := fieldVocabularies[resource]
	if !ok {
		panic("addFieldsFlagFor: unknown resource " + resource)
	}
	cmd.Flags().String("fields", "", fmt.Sprintf(
		"Comma-separated API field names to include, which also narrows the table to those columns. "+
			"Accepts: %s. id is always returned", strings.Join(vocab, ", ")))
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[fieldsResourceAnnotation] = resource
}

// validateFields rejects a --fields name outside the command's vocabulary before
// a request is spent on it. The API answers an unknown name with a 400 that names
// it, so what this adds is the accepted set in the same breath, and the reminder
// that the table's headings are not those names.
func validateFields(cmd *cobra.Command) error {
	resource := cmd.Annotations[fieldsResourceAnnotation]
	if resource == "" {
		return nil
	}
	v, _ := cmd.Flags().GetString("fields")
	if strings.TrimSpace(v) == "" {
		return nil
	}
	vocab := fieldVocabularies[resource]
	allowed := make(map[string]bool, len(vocab))
	for _, name := range vocab {
		allowed[name] = true
	}

	var unknown []string
	for _, name := range strings.Split(v, ",") {
		name = strings.TrimSpace(name)
		if name == "" || allowed[name] {
			continue
		}
		unknown = append(unknown, fmt.Sprintf("%q", name))
	}
	if len(unknown) == 0 {
		return nil
	}
	verb := "is"
	if len(unknown) > 1 {
		verb = "are"
	}
	return fmt.Errorf(
		"--fields takes API field names, and %s %s not among them. This feed accepts: %s. "+
			"The table's column headings are display labels, not field names",
		strings.Join(unknown, ", "), verb, strings.Join(vocab, ", "))
}

// selectedFields reports which of a feed's fields a command asked for, or nil
// when --fields was not set.
//
// Every name in the feed's vocabulary is present in the result, false when it was
// not requested, because a renderer needs to tell three cases apart: a field that
// was requested, one the selection excluded, and a key the selection does not
// govern at all. A semantic distance is the third case: --fields cannot ask for
// it and cannot suppress it, so it must survive narrowing untouched.
//
// id is always true: the API returns it whether or not it was asked for.
func selectedFields(cmd *cobra.Command) map[string]bool {
	resource := cmd.Annotations[fieldsResourceAnnotation]
	if resource == "" {
		return nil
	}
	v, _ := cmd.Flags().GetString("fields")
	if strings.TrimSpace(v) == "" {
		return nil
	}
	want := make(map[string]bool, len(fieldVocabularies[resource]))
	for _, name := range fieldVocabularies[resource] {
		want[name] = false
	}
	for _, name := range strings.Split(v, ",") {
		if name = strings.TrimSpace(name); name != "" {
			want[name] = true
		}
	}
	want["id"] = true
	return want
}

// applyFieldSelection validates --fields and records the resulting selection on
// the formatter, which is where every render call reads it from. Keeping both
// halves together means a command cannot end up validating its selection while
// rendering everything, or narrowing against a selection nothing checked.
func applyFieldSelection(cmd *cobra.Command, f *output.Formatter) error {
	if err := validateFields(cmd); err != nil {
		return err
	}
	f.SetFields(selectedFields(cmd))
	return nil
}

// narrowColumns drops the columns whose data a --fields selection left out of the
// response, rather than printing them empty. An empty cell reads as "no value",
// which is a different claim from "not requested", and for a boolean column it
// reads as a flat "No".
//
// colFields gives the field names behind each column, in the same order. A column
// survives if any of its fields was requested, so a column fed by either of two
// fields survives on either. A column with no field listed is never dropped,
// because nothing in the selection governs it.
func narrowColumns(want map[string]bool, columns []string, rows [][]string, colFields [][]string) ([]string, [][]string) {
	if want == nil {
		return columns, rows
	}
	keep := make([]int, 0, len(columns))
	for i := range columns {
		if i >= len(colFields) || len(colFields[i]) == 0 {
			keep = append(keep, i)
			continue
		}
		for _, name := range colFields[i] {
			if want[name] {
				keep = append(keep, i)
				break
			}
		}
	}
	if len(keep) == len(columns) {
		return columns, rows
	}

	narrowed := make([]string, len(keep))
	for i, c := range keep {
		narrowed[i] = columns[c]
	}
	out := make([][]string, len(rows))
	for i, row := range rows {
		cells := make([]string, 0, len(keep))
		for _, c := range keep {
			if c < len(row) {
				cells = append(cells, row[c])
			}
		}
		out[i] = cells
	}
	return narrowed, out
}

// fieldsWithoutColumn names the fields a caller requested that the table has no
// column for. The data is in the JSON, so the note points there rather than
// letting a narrowed table read as the whole answer.
func fieldsWithoutColumn(want map[string]bool, colFields [][]string) []string {
	if want == nil {
		return nil
	}
	shown := map[string]bool{}
	for _, names := range colFields {
		for _, name := range names {
			shown[name] = true
		}
	}
	var missing []string
	for name, requested := range want {
		// id is returned rather than asked for, and every table leads with it,
		// so it is never the thing a caller is missing.
		if requested && name != "id" && !shown[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

// noColumnNote tells the caller where the fields they asked for actually are.
func noColumnNote(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	has, them := "have", "them"
	if len(missing) == 1 {
		has, them = "has", "it"
	}
	return fmt.Sprintf("Note: %s %s no table column. Use --format json to see %s.",
		strings.Join(missing, ", "), has, them)
}

// narrowJSON drops the keys a --fields selection left out, so a field that was
// not requested is absent from the payload instead of appearing as "" or null.
// The CLI serializes whole structs, so without this a caller cannot tell a field
// it did not ask for from one the API reports as empty.
//
// Only keys the selection governs are removed. A key the vocabulary does not
// name — a semantic distance, say — is passed through, since --fields neither
// requests nor suppresses it.
func narrowJSON(want map[string]bool, v interface{}) (interface{}, error) {
	if want == nil {
		return v, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		for key := range row {
			if requested, governed := want[key]; governed && !requested {
				delete(row, key)
			}
		}
	}
	return rows, nil
}
