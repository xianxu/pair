package main

import (
	"os"
	"testing"
)

func TestParentNameReadsAKnownProcess(t *testing.T) {
	// The shim asks about its parent; any live pid exercises the same lookup.
	if got := parentName(os.Getpid()); got == "" {
		t.Fatal("could not read this process's own command name")
	}
	if got := parentName(1 << 30); got != "" {
		t.Fatalf("nonexistent pid named %q", got)
	}
}
