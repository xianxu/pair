package termcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestTermCellDiagnosticUsesManagedBoundedJSON(t *testing.T) {
	root := t.TempDir()
	log, err := openTermDiagnostics(root)
	if err != nil {
		t.Fatal(err)
	}
	log.observe(terminal.Observation{Kind: "feed", Data: []byte("private frame must not be logged")})
	detail := []byte(`{"reason":"control-content","content_codepoint_values":["U+001B"],"content_bytes":1}`)
	log.observe(terminal.Observation{Kind: "cell-degraded", EndpointID: "child-1", Data: detail})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "term-diagnostics.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("records=%d: %s", len(lines), raw)
	}
	var record struct {
		Event    string          `json:"event"`
		Endpoint string          `json:"endpoint"`
		Cell     json.RawMessage `json:"cell"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.Event != "cell-degraded" || record.Endpoint != "child-1" || !strings.Contains(string(record.Cell), "U+001B") {
		t.Fatalf("lost trigger evidence: %s", raw)
	}
	if strings.Contains(string(raw), "private frame") || strings.ContainsRune(string(raw), '\x1b') {
		t.Fatal("raw screen/control leaked")
	}
	if len(raw) > 8192 {
		t.Fatalf("record bytes=%d", len(raw))
	}
}

func TestTermDiagnosticRejectsUnboundedOrMalformedEvidence(t *testing.T) {
	root := t.TempDir()
	log, err := openTermDiagnostics(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{[]byte(`{"truncated":`), []byte(`{"x":"` + strings.Repeat("x", 9000) + `"}`)} {
		log.observe(terminal.Observation{Kind: "cell-degraded", Data: data})
	}
	log.observe(terminal.Observation{Kind: "cell-degraded", Data: []byte(`{"reason":"safe"}`)})
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "term-diagnostics.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "\n") != 1 || !strings.Contains(string(raw), "safe") {
		t.Fatalf("invalid records admitted: %q", raw)
	}
}

func TestTermDiagnosticFailureDoesNotAffectCaller(t *testing.T) {
	log, err := openTermDiagnostics(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	log.observe(terminal.Observation{Kind: "cell-degraded", Data: []byte(`{"reason":"late"}`)})
	// Observation is evidence-only, including after sink teardown.
	if _, err := openTermDiagnostics(""); err == nil {
		t.Fatal("empty root permits current-directory writes")
	}
}
