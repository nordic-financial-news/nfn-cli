package output

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

func TestEllipsize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{name: "fits unchanged", s: "Volvo", width: 10, want: "Volvo"},
		{name: "exact fit unchanged", s: "Volvo", width: 5, want: "Volvo"},
		{name: "cut with one-cell ellipsis", s: "Volvo Cars", width: 6, want: "Volvo…"},
		{name: "no space left before the ellipsis", s: "Volvo Cars", width: 7, want: "Volvo…"},
		{name: "multibyte runes count as cells", s: "Lantmännen Q2", width: 8, want: "Lantmän…"},
		{name: "width one is just the mark", s: "Volvo", width: 1, want: "…"},
		{name: "wide characters count two cells", s: "日本語テキスト", width: 6, want: "日本…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Ellipsize(tt.s, tt.width); got != tt.want {
				t.Errorf("Ellipsize(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestFitTable(t *testing.T) {
	t.Parallel()

	columns := []string{"ID", "Type", "Title", "Articles"}
	rows := [][]string{{"h2k9rqz4m7wt", "earnings-report", strings.Repeat("x", 70), "7"}}
	layout := Layout{Flex: "Title", Drop: []string{"Articles", "Type"}}

	t.Run("a table that fits is untouched", func(t *testing.T) {
		t.Parallel()
		gotCols, gotRows := fitTable(columns, rows, 200, layout)
		if !reflect.DeepEqual(gotCols, columns) || !reflect.DeepEqual(gotRows, rows) {
			t.Errorf("fitTable at 200 = %v %v, want the input unchanged", gotCols, gotRows)
		}
	})

	t.Run("the flex column shrinks before anything is dropped", func(t *testing.T) {
		t.Parallel()
		gotCols, gotRows := fitTable(columns, rows, 100, layout)
		if !reflect.DeepEqual(gotCols, columns) {
			t.Fatalf("columns = %v, want all four kept", gotCols)
		}
		if w := tableWidth(cellWidths(gotRows[0])); w > 100 {
			t.Errorf("row is %d cells wide, want at most 100", w)
		}
		if !strings.HasSuffix(gotRows[0][2], "…") {
			t.Errorf("title = %q, want it ellipsized", gotRows[0][2])
		}
	})

	t.Run("columns drop in order once the flex column hits its floor", func(t *testing.T) {
		t.Parallel()
		gotCols, gotRows := fitTable(columns, rows, 70, layout)
		if want := []string{"ID", "Title"}; !reflect.DeepEqual(gotCols, want) {
			t.Fatalf("columns = %v, want %v", gotCols, want)
		}
		if gotRows[0][0] != "h2k9rqz4m7wt" {
			t.Errorf("id = %q, want it whole", gotRows[0][0])
		}
		if w := tableWidth(cellWidths(gotRows[0])); w > 70 {
			t.Errorf("row is %d cells wide, want at most 70", w)
		}
	})

	t.Run("an unlisted column is never dropped", func(t *testing.T) {
		t.Parallel()
		gotCols, gotRows := fitTable(columns, rows, 20, layout)
		if gotCols[0] != "ID" || gotRows[0][0] != "h2k9rqz4m7wt" {
			t.Errorf("fitTable at 20 = %v %v, want the ID column kept whole", gotCols, gotRows)
		}
		if got := text.StringWidthWithoutEscSequences(gotRows[0][1]); got != flexMin {
			t.Errorf("title is %d cells, want the %d-cell minimum", got, flexMin)
		}
	})

	t.Run("a lower floor keeps columns that a default floor drops", func(t *testing.T) {
		t.Parallel()
		gotCols, _ := fitTable(columns, rows, 70, Layout{Flex: "Title", Drop: layout.Drop, Floor: 20})
		if !reflect.DeepEqual(gotCols, columns) {
			t.Errorf("columns = %v, want all four kept", gotCols)
		}
	})

	t.Run("no flex column means no change", func(t *testing.T) {
		t.Parallel()
		gotCols, _ := fitTable(columns, rows, 20, Layout{Flex: "Summary", Drop: []string{"Type"}})
		if !reflect.DeepEqual(gotCols, columns) {
			t.Errorf("columns = %v, want the input unchanged", gotCols)
		}
	})
}

func cellWidths(row []string) []int {
	widths := make([]int, len(row))
	for i, c := range row {
		widths[i] = text.StringWidthWithoutEscSequences(c)
	}
	return widths
}

// A long value must wrap inside the table. Clipping it at the edge cut words
// and IDs in half, so a copied ID no longer resolved.
func TestRenderDetailTableStaysWithinWidth(t *testing.T) {
	t.Parallel()

	ids := []string{"ldzcvhhkeiux", "k11gfg1wh80v", "bnl92fe0np98"}
	fields := []Field{
		{Key: "Primary Disclosure", Value: "ldzcvhhkeiux"},
		{Key: "Summary", Value: strings.Repeat("Lantmännen reported adjusted operating profit. ", 6)},
		{Key: "Article IDs", Value: strings.Join(ids, "\n")},
	}

	for _, width := range []int{60, 80, 100} {
		var buf bytes.Buffer
		renderDetailTableWidth(&buf, fields, true, width)
		out := buf.String()
		for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if w := text.StringWidthWithoutEscSequences(line); w > width {
				t.Errorf("width %d: line is %d cells: %q", width, w, line)
			}
		}
		// Each ID keeps its own line rather than being reflowed into a paragraph.
		for _, id := range ids {
			if !strings.Contains(out, "| "+id+" ") {
				t.Errorf("width %d: %s is not on a line of its own\n%s", width, id, out)
			}
		}
	}
}

// A wrapped list item indents under its own text, so it does not read as the
// start of the next item.
func TestWrapLinesHangsBulletedItems(t *testing.T) {
	t.Parallel()
	got := wrapLines(Bullet+"Orders fell 9.5% from a year earlier\n"+Bullet+"Short", 20)
	for i, line := range strings.Split(got, "\n") {
		if w := text.StringWidthWithoutEscSequences(line); w > 20 {
			t.Errorf("line %d is %d cells: %q", i, w, line)
		}
		if !strings.HasPrefix(line, Bullet) && !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d = %q, want a bullet or a hanging indent", i, line)
		}
	}
	if n := strings.Count(got, Bullet); n != 2 {
		t.Errorf("got %d bullets, want 2\n%s", n, got)
	}
}

func TestRenderFitSkipsAnEmptyTable(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	f := NewFormatterWithWriter("table", true, &buf)
	f.RenderFit([]string{"ID", "Title"}, nil, Layout{Flex: "Title"})
	if buf.Len() != 0 {
		t.Errorf("output = %q, want nothing for an empty table", buf.String())
	}
}

func TestProse(t *testing.T) {
	t.Parallel()
	got := prose("Intro line.\n## Targets\n- Margin 12%\n* Growth 20%\n\n### Outlook\nUnchanged.")
	want := "Intro line.\n\nTARGETS\n" + Bullet + "Margin 12%\n" + Bullet + "Growth 20%\n\nOUTLOOK\nUnchanged."
	if got != want {
		t.Errorf("prose() = %q, want %q", got, want)
	}
}

func TestProseEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, in, want string }{
		{"a # without a space is text", "#1 supplier in the Nordics", "#1 supplier in the Nordics"},
		{"CRLF line endings", "Revenue rose.\r\n## Outlook\r\nUnchanged.\r\n", "Revenue rose.\n\nOUTLOOK\nUnchanged."},
		{"bold, links and code", "**Revenue** rose, see [the report](https://example.com/q3) and `EBITDA`", "Revenue rose, see the report (https://example.com/q3) and EBITDA"},
		{"single * and _ are left alone", "5 * 3 and snake_case", "5 * 3 and snake_case"},
		{"bullet text is cleaned too", "- __Margin__ 12%", Bullet + "Margin 12%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := prose(tt.in); got != tt.want {
				t.Errorf("prose(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWrapLinesHangsNumberedItems(t *testing.T) {
	t.Parallel()
	got := wrapLines("12. Orders fell 9.5% from a year earlier", 20)
	for _, line := range strings.Split(got, "\n")[1:] {
		if !strings.HasPrefix(line, "    ") {
			t.Errorf("continuation %q not indented under the item text\n%s", line, got)
		}
	}
}

func TestRenderProseWritesNothingAsJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	NewFormatterWithWriter("json", true, &buf).RenderProse("## Heading\nBody")
	if buf.Len() != 0 {
		t.Errorf("output = %q, want nothing on a JSON formatter", buf.String())
	}
}
