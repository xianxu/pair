package main

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/historyarchive"
	"testing"
)

func TestArchiveStreamingInventory(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := runStreamingSubcommand("archive", []string{"inventory"}, nil, &out, &stderr); code != 0 {
		t.Fatalf("code=%d error=%s", code, &stderr)
	}
	if _, err := historyarchive.DecodeInventory(out.Bytes()); err != nil {
		t.Fatal(err)
	}
}
