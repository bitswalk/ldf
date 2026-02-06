package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// captureStdout captures stdout output during fn execution
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// =============================================================================
// PrintJSON Tests
// =============================================================================

func TestPrintJSON_Map(t *testing.T) {
	data := map[string]string{"key": "value"}
	out := captureStdout(t, func() {
		if err := PrintJSON(data); err != nil {
			t.Fatalf("PrintJSON error: %v", err)
		}
	})
	var result map[string]string
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result)
	}
}

func TestPrintJSON_Struct(t *testing.T) {
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	data := item{ID: "1", Name: "test"}
	out := captureStdout(t, func() {
		_ = PrintJSON(data)
	})
	if !strings.Contains(out, `"id": "1"`) {
		t.Errorf("expected id field in JSON, got %s", out)
	}
	if !strings.Contains(out, `"name": "test"`) {
		t.Errorf("expected name field in JSON, got %s", out)
	}
}

func TestPrintJSON_Indented(t *testing.T) {
	data := map[string]string{"key": "value"}
	out := captureStdout(t, func() {
		_ = PrintJSON(data)
	})
	if !strings.Contains(out, "  ") {
		t.Error("expected indented JSON output")
	}
}

// =============================================================================
// PrintYAML Tests
// =============================================================================

func TestPrintYAML_Map(t *testing.T) {
	data := map[string]string{"key": "value"}
	out := captureStdout(t, func() {
		if err := PrintYAML(data); err != nil {
			t.Fatalf("PrintYAML error: %v", err)
		}
	})
	if !strings.Contains(out, "key: value") {
		t.Errorf("expected YAML key: value, got %q", out)
	}
}

func TestPrintYAML_Struct(t *testing.T) {
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	data := item{ID: "1", Name: "test"}
	out := captureStdout(t, func() {
		_ = PrintYAML(data)
	})
	if !strings.Contains(out, "id:") {
		t.Errorf("expected id field in YAML, got %q", out)
	}
	if !strings.Contains(out, "name: test") {
		t.Errorf("expected name field in YAML, got %q", out)
	}
}

func TestPrintYAML_RespectsJsonTags(t *testing.T) {
	type item struct {
		SourceURL string `json:"source_url"`
	}
	data := item{SourceURL: "https://example.com"}
	out := captureStdout(t, func() {
		_ = PrintYAML(data)
	})
	if !strings.Contains(out, "source_url:") {
		t.Errorf("expected source_url (json tag), got %q", out)
	}
}

// =============================================================================
// PrintTable Tests
// =============================================================================

func TestPrintTable_BasicOutput(t *testing.T) {
	out := captureStdout(t, func() {
		PrintTable(
			[]string{"ID", "NAME"},
			[][]string{
				{"1", "alpha"},
				{"2", "beta"},
			},
		)
	})
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Errorf("expected headers in output, got %q", out)
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Errorf("expected row data in output, got %q", out)
	}
}

func TestPrintTable_EmptyRows(t *testing.T) {
	out := captureStdout(t, func() {
		PrintTable([]string{"ID", "NAME"}, [][]string{})
	})
	// Should still print headers
	if !strings.Contains(out, "ID") {
		t.Errorf("expected headers even with empty rows, got %q", out)
	}
}

func TestPrintTable_Alignment(t *testing.T) {
	out := captureStdout(t, func() {
		PrintTable(
			[]string{"ID", "NAME"},
			[][]string{
				{"1", "short"},
				{"100", "a much longer name"},
			},
		)
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}
}

// =============================================================================
// PrintMessage / PrintError Tests
// =============================================================================

func TestPrintMessage(t *testing.T) {
	out := captureStdout(t, func() {
		PrintMessage("hello world")
	})
	if strings.TrimSpace(out) != "hello world" {
		t.Errorf("expected 'hello world', got %q", out)
	}
}

func TestPrintError(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintError(fmt.Errorf("test error"))

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "test error") {
		t.Errorf("expected error message on stderr, got %q", buf.String())
	}
}
