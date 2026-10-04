package output

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// defaultWidth is the width assumed when neither the terminal nor $COLUMNS
// says otherwise.
const defaultWidth = 120

// flexFloor is the width below which the flexible column stops shrinking and
// the droppable columns go instead: a title cut shorter than this no longer
// identifies the row. flexMin is the narrowest it is ever drawn, once nothing
// is left to drop.
const (
	flexFloor = 40
	flexMin   = 8
)

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// termWidth is the width a table is fitted to: the terminal's when writing to
// one, then $COLUMNS, which is how a pipe or a script states the width it
// wants, then defaultWidth.
func termWidth(w io.Writer) int {
	if file, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(file.Fd())); err == nil && width > 0 {
			return width
		}
	}
	if width, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && width > 0 {
		return width
	}
	return defaultWidth
}

// Layout says how a list table fits its width. Flex names the column that
// takes whatever width the others leave, truncated to fit rather than
// clipped. Drop names columns, least important first, removed whole before
// Flex is cut below flexFloor. A column named in neither keeps its full width,
// which is what keeps an ID column intact: it is the handle every get command
// takes, so it is never a candidate.
type Layout struct {
	Flex string
	Drop []string
}

// tableWidth is the rendered width of a bordered row with these column widths:
// each column carries a space either side and a separator, plus the closing
// border.
func tableWidth(widths []int) int {
	total := 1
	for _, w := range widths {
		total += w + 3
	}
	return total
}

// fitTable applies a Layout to a table at the given width, returning the
// columns and rows to render. Without a Flex column present it changes nothing.
func fitTable(columns []string, rows [][]string, width int, layout Layout) ([]string, [][]string) {
	natural := make([]int, len(columns))
	for i, c := range columns {
		natural[i] = text.StringWidthWithoutEscSequences(c)
	}
	for _, row := range rows {
		for i := 0; i < len(row) && i < len(natural); i++ {
			if w := text.StringWidthWithoutEscSequences(row[i]); w > natural[i] {
				natural[i] = w
			}
		}
	}

	flex := indexOf(columns, layout.Flex)
	if flex < 0 {
		return columns, rows
	}

	keep := make([]bool, len(columns))
	for i := range keep {
		keep[i] = true
	}
	room := func() int {
		var widths []int
		for i, k := range keep {
			if k && i != flex {
				widths = append(widths, natural[i])
			}
		}
		// The flex column's own padding and separator are part of the row.
		return width - tableWidth(widths) - 3
	}

	for _, name := range layout.Drop {
		if room() >= flexFloor || room() >= natural[flex] {
			break
		}
		if i := indexOf(columns, name); i >= 0 && i != flex {
			keep[i] = false
		}
	}

	limit := room()
	if limit < flexMin {
		limit = flexMin
	}

	var outCols []string
	for i, c := range columns {
		if keep[i] {
			outCols = append(outCols, c)
		}
	}
	outRows := make([][]string, len(rows))
	for r, row := range rows {
		cells := make([]string, 0, len(outCols))
		for i, k := range keep {
			if !k {
				continue
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if i == flex {
				cell = Ellipsize(cell, limit)
			}
			cells = append(cells, cell)
		}
		outRows[r] = cells
	}
	return outCols, outRows
}

// Ellipsize shortens s to at most width display cells, marking the cut with a
// single-cell ellipsis.
func Ellipsize(s string, width int) string {
	if text.StringWidthWithoutEscSequences(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	// Cut by display cells, not runes: a wide character takes two.
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := text.RuneWidth(r)
		if used+w > width-1 {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return strings.TrimRight(b.String(), " ") + "…"
}

// wrapLines soft-wraps each line of a cell on its own, so a value laid out one
// item per line keeps that layout instead of being reflowed into a paragraph.
func wrapLines(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = text.WrapSoft(line, width)
	}
	return strings.Join(lines, "\n")
}

func indexOf(columns []string, name string) int {
	if name == "" {
		return -1
	}
	for i, c := range columns {
		if strings.EqualFold(c, name) {
			return i
		}
	}
	return -1
}

func tableStyle(noColor bool) table.Style {
	if !noColor && isTTY() {
		return table.StyleRounded
	}
	return table.StyleDefault
}

func renderTable(w io.Writer, columns []string, rows [][]string, noColor bool, layout *Layout) {
	width := termWidth(w)
	if layout != nil {
		columns, rows = fitTable(columns, rows, width, *layout)
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetAllowedRowLength(width)

	header := make(table.Row, len(columns))
	for i, c := range columns {
		header[i] = c
	}
	t.AppendHeader(header)

	for _, row := range rows {
		r := make(table.Row, len(row))
		for i, v := range row {
			r[i] = v
		}
		t.AppendRow(r)
	}

	t.SetStyle(tableStyle(noColor))
	t.Render()
}

func renderDetailTable(w io.Writer, fields []Field, noColor bool) {
	renderDetailTableWidth(w, fields, noColor, termWidth(w))
}

func renderDetailTableWidth(w io.Writer, fields []Field, noColor bool, width int) {
	keyWidth := 0
	for _, f := range fields {
		if kw := text.StringWidthWithoutEscSequences(f.Key); kw > keyWidth {
			keyWidth = kw
		}
	}
	// The value column gets what the key column and the borders leave, so a
	// long value wraps inside the table instead of running off its edge, where
	// it would be clipped mid-word or mid-ID.
	valueWidth := width - tableWidth([]int{keyWidth, 0})
	if valueWidth < 20 {
		valueWidth = 20
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 2, WidthMax: valueWidth, WidthMaxEnforcer: wrapLines},
	})

	for _, f := range fields {
		t.AppendRow(table.Row{f.Key, f.Value})
	}

	t.SetStyle(tableStyle(noColor))
	t.Render()
}
