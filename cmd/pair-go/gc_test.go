package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGCStreamingRouteUsesExplicitRoot(t *testing.T) {
	var out, errout bytes.Buffer
	code := runStreamingSubcommand("gc", []string{"--root", t.TempDir(), "--json"}, strings.NewReader(""), &out, &errout)
	if code != 0 || !strings.Contains(out.String(), "migration_complete") {
		t.Fatalf("%d %s %s", code, &out, &errout)
	}
}
