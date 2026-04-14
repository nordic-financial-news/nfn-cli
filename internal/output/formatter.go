package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Formatter handles rendering output in different formats.
type Formatter struct {
	format  string
	writer  io.Writer
	noColor bool
}

// NewFormatter creates a new Formatter that writes to stdout.
func NewFormatter(format string, noColor bool) *Formatter {
	return &Formatter{
		format:  format,
		writer:  os.Stdout,
		noColor: noColor,
	}
}

// NewFormatterWithWriter creates a Formatter that writes to the given writer.
func NewFormatterWithWriter(format string, noColor bool, w io.Writer) *Formatter {
	return &Formatter{
		format:  format,
		writer:  w,
		noColor: noColor,
	}
}

// Format returns the current output format.
func (f *Formatter) Format() string {
	return f.format
}

// Render outputs columnar data as a table or JSON.
func (f *Formatter) Render(columns []string, rows [][]string) {
	if f.format == "json" {
		f.renderRowsAsJSON(columns, rows)
		return
	}
	renderTable(f.writer, columns, rows, f.noColor)
}

// RenderDetail outputs key-value pairs as a table or JSON.
func (f *Formatter) RenderDetail(fields []Field) {
	if f.format == "json" {
		f.renderFieldsAsJSON(fields)
		return
	}
	renderDetailTable(f.writer, fields, f.noColor)
}

// RenderJSON outputs any value as JSON.
func (f *Formatter) RenderJSON(v interface{}) error {
	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Field is a key-value pair for detail views.
type Field struct {
	Key   string
	Value string
}

func (f *Formatter) renderRowsAsJSON(columns []string, rows [][]string) {
	result := make([]map[string]string, len(rows))
	for i, row := range rows {
		m := make(map[string]string)
		for j, col := range columns {
			if j < len(row) {
				m[col] = row[j]
			}
		}
		result[i] = m
	}
	_ = f.RenderJSON(result)
}

func (f *Formatter) renderFieldsAsJSON(fields []Field) {
	m := make(map[string]string)
	for _, field := range fields {
		m[field.Key] = field.Value
	}
	_ = f.RenderJSON(m)
}

// Println prints a line to the formatter's writer.
func (f *Formatter) Println(a ...interface{}) {
	_, _ = fmt.Fprintln(f.writer, a...)
}

// Printf prints formatted output to the formatter's writer.
func (f *Formatter) Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(f.writer, format, a...)
}
