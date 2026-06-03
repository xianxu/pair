package adapt

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedTime is a deterministic clock for line-format assertions.
var fixedTime = time.Date(2026, 6, 3, 10, 22, 1, 0, time.UTC)

func TestLogLineFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "pair-wrap", "codex")
	l.now = func() time.Time { return fixedTime }

	l.Log(2, "overlay-detect", NearMiss, "prompt-shaped, no detector matched: 'Do you want to continue?'")

	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("line not newline-terminated: %q", out)
	}

	var got event
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("line is not valid JSON: %v\n%s", err, out)
	}
	want := event{
		TS:      "2026-06-03T10:22:01Z",
		Comp:    "pair-wrap",
		Agent:   "codex",
		Aspect:  2,
		Signal:  "overlay-detect",
		Outcome: "near-miss",
		Detail:  "prompt-shaped, no detector matched: 'Do you want to continue?'",
	}
	if got != want {
		t.Errorf("event mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestNilLoggerIsNoOp(t *testing.T) {
	var l *Logger
	// Must not panic.
	l.Log(1, "return-remap", Fired, "x")
	if err := l.Close(); err != nil {
		t.Errorf("nil Close returned %v, want nil", err)
	}
}

func TestDetailTruncatedToValidUTF8(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "pair-wrap", "agy")
	l.now = func() time.Time { return fixedTime }

	// 100 multi-byte runes (300 bytes) — must truncate without splitting one,
	// keeping the JSON valid.
	long := strings.Repeat("é", 100)
	l.Log(2, "overlay-detect", NearMiss, long)

	var got event
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("truncated line is not valid JSON: %v", err)
	}
	if len(got.Detail) > maxDetail {
		t.Errorf("detail %d bytes, want <= %d", len(got.Detail), maxDetail)
	}
	if !utf8Valid(got.Detail) {
		t.Errorf("truncation split a rune: %q", got.Detail)
	}
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}
	return true
}

func TestTruncateEdges(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc", "abc"}, // n > len → whole
		{strings.Repeat("a", 200), strings.Repeat("a", 200)}, // exactly n → whole
		{strings.Repeat("a", 201), strings.Repeat("a", 200)}, // 1 over → cut to n
	}
	for _, c := range cases {
		if got := truncate(c.in, maxDetail); got != c.want {
			t.Errorf("truncate(%d bytes) = %d bytes, want %d", len(c.in), len(got), len(c.want))
		}
	}
	// A cut landing mid-rune backs up to a boundary and stays valid UTF-8.
	got := truncate(strings.Repeat("é", maxDetail), maxDetail) // 2*maxDetail bytes
	if len(got) > maxDetail || !utf8Valid(got) {
		t.Errorf("mid-rune truncate: len=%d valid=%v", len(got), utf8Valid(got))
	}
}

// TestGoldenMatchesFixture pins the Go emitter to the shared cross-emitter
// fixture (tests/adapt-golden.expected). The shell + Lua emitters are checked
// against the same file in tests/adapt-schema-test.sh, so this is the Go leg of
// the "same schema, three languages" contract — the internal package can't be
// imported from tests/, so the Go assertion lives here.
func TestGoldenMatchesFixture(t *testing.T) {
	line := marshalEvent(fixedTime, "golden", "codex", 2, "overlay-detect", NearMiss, "press > to continue? (y/n)")
	// Normalize the only field that legitimately varies (ts) by string
	// replacement, preserving field order (a map round-trip would re-sort keys).
	got := strings.TrimRight(string(line), "\n")
	got = strings.Replace(got, fixedTime.Format(time.RFC3339), "TS", 1)

	raw, err := os.ReadFile("../../../tests/adapt-golden.expected")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	want := strings.TrimSpace(string(raw))
	if got != want {
		t.Errorf("Go emitter diverged from cross-emitter fixture:\n got %s\nwant %s", got, want)
	}
}

func TestOpenNoTagReturnsNilNoOp(t *testing.T) {
	t.Setenv("PAIR_TAG", "")
	l := Open("pair-wrap", "codex")
	if l != nil {
		t.Fatalf("Open with empty PAIR_TAG should return nil, got %v", l)
	}
	l.Log(1, "x", Fired, "y") // must not panic
	if err := l.Close(); err != nil {
		t.Errorf("nil Close = %v, want nil", err)
	}
}

func TestOpenAppendsAcrossOpens(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PAIR_TAG", "t1")
	t.Setenv("PAIR_DATA_DIR", dir)

	l1 := Open("pair-wrap", "codex")
	if l1 == nil {
		t.Fatal("Open returned nil with PAIR_TAG set")
	}
	l1.Log(1, "return-remap", Fired, "first")
	_ = l1.Close()

	// A second Open (e.g. a different process) must append, not truncate.
	l2 := Open("pair-slug", "codex")
	l2.Log(4, "slug-parse", NearMiss, "second")
	_ = l2.Close()

	data, err := os.ReadFile(filepath.Join(dir, "adapt-t1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(strings.TrimSpace(string(data)), "\n") + 1
	if lines != 2 {
		t.Fatalf("want 2 appended lines, got %d:\n%s", lines, data)
	}
}
