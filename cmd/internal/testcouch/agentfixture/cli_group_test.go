package agentfixture

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestCLILeaderExitCannotAcknowledgeLiveDescendant(t *testing.T) {
	for _, tc := range []struct{ name, redirect string }{
		{"inherited-pipes", ""},
		{"closed-pipes", " >/dev/null 2>&1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "descendant-pid")
			result := runCLI(t.Context(), "/bin/sh", Command{ID: "leader-exited", Timeout: 200 * time.Millisecond, Args: []string{"-c", `sleep 30` + tc.redirect + ` & printf '%s' "$!" > "$1"; exit 0`, "fixture", marker}})
			raw, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			pid, err := strconv.Atoi(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			generation := procutil.StrictIdentity(strconv.Itoa(pid))
			t.Cleanup(func() {
				if generation != "" && procutil.StrictIdentity(strconv.Itoa(pid)) == generation {
					if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
						t.Error(err)
					}
				}
			})
			if err := syscall.Kill(pid, 0); err == nil && !strings.Contains(result.Error, "cleanup uncertain") {
				t.Fatalf("request acknowledged while owned descendant is alive: pid=%d result=%+v", pid, result)
			}
		})
	}
}
