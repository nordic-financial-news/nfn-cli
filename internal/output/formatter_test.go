package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatter_Render_JSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", true, &buf)

	columns := []string{"ID", "Name"}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}
	f.Render(columns, rows)

	var got []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}
	if got[0]["ID"] != "1" || got[0]["Name"] != "Alice" {
		t.Errorf("got[0] = %v, want ID=1, Name=Alice", got[0])
	}
	if got[1]["ID"] != "2" || got[1]["Name"] != "Bob" {
		t.Errorf("got[1] = %v, want ID=2, Name=Bob", got[1])
	}
}

func TestFormatter_Render_JSON_EmptyRows(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", true, &buf)

	f.Render([]string{"ID"}, [][]string{})

	var got []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d items, want 0", len(got))
	}
}

func TestFormatter_RenderDetail_JSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", true, &buf)

	fields := []Field{
		{Key: "Title", Value: "Test Article"},
		{Key: "Country", Value: "SE"},
	}
	f.RenderDetail(fields)

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["Title"] != "Test Article" {
		t.Errorf("Title = %q, want %q", got["Title"], "Test Article")
	}
	if got["Country"] != "SE" {
		t.Errorf("Country = %q, want %q", got["Country"], "SE")
	}
}

func TestFormatter_RenderJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", true, &buf)

	data := map[string]interface{}{
		"articles": []map[string]string{
			{"id": "1"},
		},
	}
	if err := f.RenderJSON(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"articles"`) {
		t.Error("output missing 'articles' key")
	}
	// Verify indentation
	if !strings.Contains(output, "  ") {
		t.Error("output should be indented")
	}
}

func TestFormatter_Render_Table(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("table", true, &buf)

	columns := []string{"ID", "Title"}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}
	f.Render(columns, rows)

	output := buf.String()
	for _, want := range []string{"ID", "TITLE", "Alice", "Bob"} {
		if !strings.Contains(strings.ToUpper(output), strings.ToUpper(want)) {
			t.Errorf("table output missing %q\ngot:\n%s", want, output)
		}
	}
}

func TestFormatter_RenderDetail_Table(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("table", true, &buf)

	fields := []Field{
		{Key: "Title", Value: "My Article"},
		{Key: "Country", Value: "NO"},
	}
	f.RenderDetail(fields)

	output := buf.String()
	if !strings.Contains(output, "Title") {
		t.Error("detail output missing 'Title' key")
	}
	if !strings.Contains(output, "My Article") {
		t.Error("detail output missing 'My Article' value")
	}
}
