package adapt

import (
	"bytes"
	"encoding/json"
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
