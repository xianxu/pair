package main

import (
	"os"
	"testing"
)

func TestParentNameReadsThisTestsParent(t *testing.T) {
	if got := parentName(os.Getpid()); got == "" {
		t.Fatal("could not read this process's own command name")
	}
	if got := parentName(1 << 30); got != "" {
		t.Fatalf("nonexistent pid named %q", got)
	}
}
