package launcher

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

// The command still runs through the real owned-process collector. Inject an
// independent collection failure after the native process result, rather than
// substituting a parser result or fabricating an ExitError.
func TestQueryCollectionFailuresCannotBecomeDiagnosticEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, diagnostic   string
		status             int
		inventory, tooLong bool
	}{
		{name: "empty-inventory", diagnostic: "No active zellij sessions found.", status: 1, inventory: true},
		{name: "absent-name", diagnostic: "There is no active session!", status: 1},
		{name: "validator", diagnostic: "error: session name must be less than 108 characters", status: 2, tooLong: true},
	} {
		for _, fault := range []string{"none", "collection", "collection-and-cancel"} {
			t.Run(tc.name+"/"+fault, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "zellij")
				// Fixed fixture strings contain no shell metacharacters.
				script := "#!/bin/sh\nprintf '%s\\n' '" + tc.diagnostic + "' >&2\nexit " + map[int]string{1: "1", 2: "2"}[tc.status] + "\n"
				if err := os.WriteFile(path, []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				run := func(cmd *exec.Cmd) error {
					err := zellijexec.Run(cmd)
					if fault == "none" {
						return err
					}
					if fault == "collection-and-cancel" {
						cancel()
					}
					return errors.Join(err, syscall.EIO)
				}
				var err error
				if tc.inventory {
					_, err = (ZellijSource{Path: path, queryRun: run}).runContext(ctx, "list-sessions", "--short")
				} else {
					err = (OSRuntime{nameQueryRun: run}).ProbeSessionNameContext(ctx, "pair-new")
				}
				if fault == "none" {
					if tc.tooLong {
						if !errors.Is(err, errSessionNameTooLong) {
							t.Fatalf("pure validator rejected: %v", err)
						}
					} else if err != nil {
						t.Fatalf("pure absence rejected: %v", err)
					}
					return
				}
				if !errors.Is(err, syscall.EIO) || errors.Is(err, errSessionNameTooLong) {
					t.Fatalf("collector failure became diagnostic authority: %v", err)
				}
				if fault == "collection-and-cancel" && !errors.Is(err, context.Canceled) {
					t.Fatalf("caller cancellation lost: %v", err)
				}
			})
		}
	}
}
