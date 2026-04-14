package output

import (
	"io"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120
	}
	return w
}

func renderTable(w io.Writer, columns []string, rows [][]string, noColor bool) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetAllowedRowLength(termWidth())

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

	if !noColor && isTTY() {
		t.SetStyle(table.StyleRounded)
	} else {
		t.SetStyle(table.StyleDefault)
	}

	t.Render()
}

func renderDetailTable(w io.Writer, fields []Field, noColor bool) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetAllowedRowLength(termWidth())
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 2, WidthMax: termWidth() - 20},
	})

	for _, f := range fields {
		t.AppendRow(table.Row{f.Key, f.Value})
	}

	if !noColor && isTTY() {
		t.SetStyle(table.StyleRounded)
	} else {
		t.SetStyle(table.StyleDefault)
	}

	t.Render()
}
