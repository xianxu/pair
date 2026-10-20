package wrapcmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// This child holds the actual session-watch command immediately before its
// retention acquisition. A pipe, rather than host load, chooses the ordering.
func watcherLifetimeHelper() {
	ready := os.Getenv("PAIR_715_WATCHER_READY")
	if ready == "" || len(os.Args) < 3 || os.Args[1] != "session-watch" {
		return
	}
	if os.Getenv("PAIR_715_IGNORE_TERM") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}
	if os.Getenv("PAIR_715_MODE") == "descendant-ignores-term" {
		child := exec.Command("/bin/sh", "-c", "trap '' TERM; printf r; exec sleep 60")
		ready, err := child.StdoutPipe()
		if err != nil || child.Start() != nil {
			os.Exit(4)
		}
		if _, err := io.ReadFull(ready, make([]byte, 1)); err != nil {
			os.Exit(4)
		}
		_ = ready.Close()
	}
	if err := os.WriteFile(ready, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		os.Exit(2)
	}
	if os.Getenv("PAIR_715_MODE") == "already-exited" {
		os.Exit(0)
	}
	gate := os.NewFile(4, "watcher-startup-gate")
	if _, err := io.ReadFull(gate, make([]byte, 1)); err != nil {
		os.Exit(3)
	}
	_ = gate.Close()
	os.Exit(sessionwatch.RunCLI(os.Args[2:], os.Getenv, os.Stderr))
}

func TestWrapJoinsWatcherBeforeReturning(t *testing.T) {
	for _, mode := range []string{"normal", "ignores-term", "descendant-ignores-term", "already-exited", "start-failure"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			scope, tag := "0123456789abcdef", "watcher-lifetime"
			data := filepath.Join(root, "data", "repos", scope)
			if err := os.MkdirAll(data, 0700); err != nil {
				t.Fatal(err)
			}
			ready, nativeEnd := filepath.Join(root, "watcher-ready"), filepath.Join(root, "native-end")
			testpair.ApplyEnvironment(t, root, "PAIR_DATA_DIR="+data, "PAIR_TAG="+tag,
				"PAIR_SCOPE_KEY="+scope, "PAIR_LAUNCH_ORDINAL=1", "PAIR_715_WATCHER_READY="+ready, "PAIR_715_MODE="+mode)
			isolateNotificationSockets(t)
			if mode == "ignores-term" {
				t.Setenv("PAIR_715_IGNORE_TERM", "1")
			}
			paths, err := artifactpath.ResolveScoped(data, tag)
			if err != nil {
				t.Fatal(err)
			}
			launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch,
				ScopeKey: scope, Tag: tag, Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(paths.Ledger(), append(launch, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			native := filepath.Join(root, "codex")
			script := "#!/bin/sh\nwhile [ ! -f \"$PAIR_715_WATCHER_READY\" ]; do sleep 0.01; done\necho ended > \"$PAIR_715_NATIVE_END\"\n"
			if mode == "start-failure" {
				script = "#!/bin/sh\necho ended > \"$PAIR_715_NATIVE_END\"\n"
			}
			if err := os.WriteFile(native, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PAIR_715_NATIVE_END", nativeEnd)
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			defer writer.Close()
			prior := startWatcherProcess
			defer func() { startWatcherProcess = prior }()
			var watcher *ownedGroup
			starts := 0
			defer func() {
				if watcher != nil {
					_ = watcher.Wait()
				}
			}()
			startWatcherProcess = func(argv, env []string, files ...*os.File) (*ownedGroup, error) {
				starts++
				if mode == "start-failure" {
					argv = append([]string{filepath.Join(root, "missing-watcher")}, argv[1:]...)
				}
				var err error
				watcher, err = realStartWatcherProcess(argv, env, append(files, reader)...)
				if err == nil && mode == "already-exited" {
					waitDetachCondition(t, "watcher exit before wrapper continues", func() bool {
						return !groupHasLiveMember(watcher.cmd.Process.Pid)
					})
				}
				return watcher, err
			}
			var stderr bytes.Buffer
			code := Run([]string{native}, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
			if code != 0 {
				t.Fatalf("wrap exited %d: %s", code, &stderr)
			}
			returned := time.Now()
			info, err := os.Stat(nativeEnd)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s native exit marker to wrapper return: %s", mode, returned.Sub(info.ModTime()))
			if starts != 1 {
				t.Fatalf("watcher starts = %d, want one", starts)
			}
			if mode == "start-failure" {
				if watcher != nil {
					t.Fatal("failed start retained a watcher")
				}
				return
			}
			if watcher == nil {
				t.Fatal("watcher was not started")
			}
			if watcher.cmd.ProcessState != nil {
				if groupHasLiveMember(watcher.cmd.Process.Pid) {
					t.Fatal("reaped watcher left a live writer in its group")
				}
				return
			}
			// RED evidence: only a still-unjoined direct child can be here. Keep
			// its PID unreaped, release startup, and observe the real CLI lease.
			if _, err := writer.Write([]byte{1}); err != nil {
				t.Fatal(err)
			}
			waitDetachCondition(t, "late session-watch retention acquisition", func() bool {
				owners, _ := filepath.Glob(filepath.Join(root, "data", ".retention", "owners", "*.json"))
				for _, path := range owners {
					raw, _ := os.ReadFile(path)
					if strings.Contains(string(raw), `"role":"session-watch"`) {
						return true
					}
				}
				return false
			})
			t.Fatal("wrapper returned before joining watcher; its real session-watch command acquired a retention lease afterward")
		})
	}
}
