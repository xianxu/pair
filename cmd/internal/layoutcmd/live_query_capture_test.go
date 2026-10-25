package layoutcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestLiveQueryCaptureKeepsRawStreamsSeparate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte("#!/bin/sh\n"+zellijtest.ShellCase(t, "layout.raw-query-capture")+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(dir, "native"))
	var trace bytes.Buffer
	rt := &fullscreenLiveRuntime{session: "private", caller: "3", queryLog: func(format string, args ...any) { fmt.Fprintf(&trace, format, args...) }}
	raw, err := rt.ListPanesJSONContext(context.Background())
	if err != nil || string(raw) != `[{"id":3` {
		t.Fatalf("stdout=%q error=%v", raw, err)
	}
	if !strings.Contains(trace.String(), `stdout="[{\"id\":3"`) || !strings.Contains(trace.String(), `stderr="fixture warning"`) || !strings.Contains(trace.String(), "list-panes") {
		t.Fatalf("raw native evidence absent or merged: %q", trace.String())
	}
}
