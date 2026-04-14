package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}

	if err := WriteJSON(&buf, data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should be indented
	if !strings.Contains(output, "  ") {
		t.Error("output should be indented")
	}
	// Should end with newline (json.Encoder.Encode adds trailing newline)
	if !strings.HasSuffix(output, "\n") {
		t.Error("output should end with newline")
	}
	// Should contain the data
	if !strings.Contains(output, `"name": "test"`) {
		t.Errorf("unexpected output: %s", output)
	}
}
