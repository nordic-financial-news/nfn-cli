package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestRenderEnvelope(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", false, &buf)

	data := map[string]interface{}{"items": []string{"a", "b"}}
	breadcrumbs := []Breadcrumb{
		{Description: "Next step", Command: "nfn foo bar"},
	}

	if err := f.RenderEnvelope(data, "2 items", breadcrumbs); err != nil {
		t.Fatalf("RenderEnvelope: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !env.OK {
		t.Error("expected ok=true")
	}
	if env.Summary != "2 items" {
		t.Errorf("summary = %q, want %q", env.Summary, "2 items")
	}
	if len(env.Breadcrumbs) != 1 {
		t.Fatalf("breadcrumbs len = %d, want 1", len(env.Breadcrumbs))
	}
	if env.Breadcrumbs[0].Command != "nfn foo bar" {
		t.Errorf("breadcrumb command = %q", env.Breadcrumbs[0].Command)
	}
}

func TestRenderEnvelopeNilBreadcrumbs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", false, &buf)

	if err := f.RenderEnvelope("hello", "1 item", nil); err != nil {
		t.Fatalf("RenderEnvelope: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if env.Breadcrumbs == nil {
		t.Error("breadcrumbs should be empty slice, not nil")
	}
	if len(env.Breadcrumbs) != 0 {
		t.Errorf("breadcrumbs len = %d, want 0", len(env.Breadcrumbs))
	}
}

func TestRenderError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	f := NewFormatterWithWriter("json", false, &buf)

	if err := f.RenderError(errors.New("something broke")); err != nil {
		t.Fatalf("RenderError: %v", err)
	}

	var env ErrorEnvelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if env.OK {
		t.Error("expected ok=false")
	}
	if env.Error != "something broke" {
		t.Errorf("error = %q, want %q", env.Error, "something broke")
	}
	if env.Data != nil {
		t.Errorf("data = %v, want nil", env.Data)
	}
}
