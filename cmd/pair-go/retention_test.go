package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRetentionStreamingRunner(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	t.Setenv("PAIR_TAG", "test")
	t.Setenv("PAIR_SCOPE_KEY", "")
	var out, err bytes.Buffer
	code := runStreamingSubcommand("retention", []string{"invalid-action"}, strings.NewReader(""), &out, &err)
	if code != 1 || strings.Contains(err.String(), "no runner wired") {
		t.Fatalf("retention not wired: %d %s", code, err.String())
	}
}
