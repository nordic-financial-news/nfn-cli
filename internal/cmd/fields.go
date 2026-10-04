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
		"content_type", "source", "company_ids", "country", "category", "event_id",
	},
	"story": {
		"id", "title", "summary", "published_at", "updated_at",
		"article_count", "article_ids", "country", "category", "company_ids", "event_id",
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

// passThroughJSONKeys names the keys a --fields selection neither requests nor
// suppresses, because the API merges them onto a row after it has projected it.
// distance is the semantic search's ranking output rather than a field of any
// resource, and a projected row with the scores stripped out would be
// uninterpretable. Keys merged before the projection (company_sentiment,
// coverage_sentiment) are stripped by the API itself and need no entry here.
var passThroughJSONKeys = map[string]bool{"distance": true}

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

// selectedFields reports the fields a command asked for, or nil when --fields was
// not set.
//
// id is absent unless the caller named it. The API returns id either way, but
// what that should mean differs by consumer: the JSON keeps it, the table keeps
// its column, and the note about fields with no column must not name a field
// nobody asked about. Each decides for itself rather than reading a forced entry
// here as a request.
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
	for _, name := range strings.Split(v, ",") {
		if name = strings.TrimSpace(name); name != "" {
			want[name] = true
		}
	}
	if len(want) == 0 {
		return nil
	}
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
// fields survives on either. An id column survives unrequested, because the API
// returns id whether or not it was asked for and it is the handle every get
// command takes. A column with no field listed is never dropped, because nothing
// in the selection governs it.
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
			if want[name] || name == "id" {
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
		// A short row keeps a blank cell in place rather than being compacted,
		// so the cells that follow stay under their own headers.
		cells := make([]string, len(keep))
		for j, c := range keep {
			if c < len(row) {
				cells[j] = row[c]
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
		if requested && !shown[name] {
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

// narrowJSON keeps the keys the caller asked for and drops the rest, so a field
// that was not requested is absent from the payload instead of appearing as ""
// or null. The API has already projected the row; what this undoes is the zero
// value Go fills in for every struct field the projection left out, which would
// otherwise assert that a company is inactive or an exchange has no currency.
//
// id survives unrequested because the API returns it either way, and so does a
// key in passThroughJSONKeys.
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
			if !want[key] && key != "id" && !passThroughJSONKeys[key] {
				delete(row, key)
			}
		}
	}
	return rows, nil
}
