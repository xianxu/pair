package launcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A PID file is display metadata, not authority to signal a process. The
// wrapper's lease tears down its child; legacy cleanup has a separate sample
// and generation-fenced predicate.
func TestTitleCleanupDoesNotSignalPIDFile(t *testing.T) {
	for _, contextual := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "contextual"}[contextual], func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "signal-attempt")
			killPath := filepath.Join(root, "kill")
			if err := os.WriteFile(killPath, []byte("#!/bin/sh\n: > '"+marker+"'\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", root)
			pidPath := filepath.Join(root, "title-pid-work")
			if err := os.WriteFile(pidPath, []byte("2147483646\n"), 0600); err != nil {
				t.Fatal(err)
			}
			rt := NewOSRuntime(root, "/pair")
			if contextual {
				if err := rt.KillTitlePollerContext(context.Background(), pidPath); err != nil {
					t.Fatal(err)
				}
			} else {
				rt.KillTitlePoller("work")
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("pidfile alone authorized a signal attempt")
			}
			if raw, err := os.ReadFile(pidPath); err != nil || string(raw) != "2147483646\n" {
				t.Fatal("launcher modified metadata owned by a title generation")
			}
		})
	}
}
