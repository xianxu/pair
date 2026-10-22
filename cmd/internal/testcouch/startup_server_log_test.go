package testcouch

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func serverLogFixture(t *testing.T) (*Fleet, string) {
	t.Helper()
	roots, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: roots, t: t}
	path := filepath.Join(roots.Mutable["tmp"], fmt.Sprintf("zellij-%d", os.Getuid()), "zellij-log", "zellij.log")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	return f, path
}

func TestStartupEvidenceRetainsPrivateServerLog(t *testing.T) {
	for _, size := range []int{8192, 16384} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			f, path := serverLogFixture(t)
			body := bytes.Repeat([]byte("x"), size)
			copy(body[len(body)-len("server-tail\x1b"):], "server-tail\x1b")
			if size > 8192 {
				copy(body, "omitted-server-prefix")
			}
			if err := os.WriteFile(path, body, 0600); err != nil {
				t.Fatal(err)
			}
			// The collector must derive the child's TMPDIR from its private roots.
			t.Setenv("TMPDIR", t.TempDir())
			snapshot := f.captureStartupLog()
			for _, want := range []string{
				"zellij_server_log", "bytes=8192", fmt.Sprintf("size=%d", size),
				fmt.Sprintf("offset=%d", size-8192), fmt.Sprintf("truncated=%t", size > 8192), `server-tail\x1b`,
			} {
				if !strings.Contains(snapshot, want) {
					t.Errorf("missing %q in retained private server log", want)
				}
			}
			if strings.Contains(snapshot, "omitted-server-prefix") || strings.ContainsRune(snapshot, '\x1b') {
				t.Fatal("retained omitted prefix or unescaped terminal control")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(snapshot, `server-tail\x1b`) {
				t.Fatal("snapshot borrowed deleted server log")
			}
		})
	}
}

func TestStartupEvidenceServerLogRefusesUnsafeSources(t *testing.T) {
	for _, kind := range []string{"missing", "final-symlink", "escaping-parent", "fifo", "directory"} {
		t.Run(kind, func(t *testing.T) {
			f, path := serverLogFixture(t)
			outside := t.TempDir()
			secret := filepath.Join(outside, "zellij.log")
			if err := os.WriteFile(secret, []byte("foreign-server-secret"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			switch kind {
			case "final-symlink":
				err = os.Symlink(secret, path)
			case "escaping-parent":
				if err = os.Remove(filepath.Dir(path)); err == nil {
					err = os.Symlink(outside, filepath.Dir(path))
				}
			case "fifo":
				err = syscall.Mkfifo(path, 0600)
			case "directory":
				err = os.Mkdir(path, 0700)
			}
			if err != nil {
				t.Fatal(err)
			}
			snapshot := f.captureStartupLog()
			_, server, found := strings.Cut(snapshot, "zellij_server_log")
			if !found || !strings.Contains(server, "bytes=0") || !strings.Contains(server, "read_error=") || strings.Contains(server, "read_error=<nil>") || strings.Contains(snapshot, "foreign-server-secret") {
				t.Fatalf("missing or unsafe source was not refused: %s", snapshot)
			}
		})
	}
}

func TestStartupEvidenceServerLogCaptureBudgetSurvivesRepeatedCleanup(t *testing.T) {
	for _, repeats := range []int{16, 32} {
		t.Run(strconv.Itoa(repeats), func(t *testing.T) {
			f, path := serverLogFixture(t)
			// A missing file consumes this phase's snapshot too; diagnostics must
			// never become an unbounded retry loop during failed cleanup.
			captured := strings.Count(f.captureStartupLog(), "zellij_server_log")
			if err := os.WriteFile(path, []byte("late-server-log"), 0600); err != nil {
				t.Fatal(err)
			}
			for range repeats {
				snapshot := f.captureStartupLog()
				captured += strings.Count(snapshot, "zellij_server_log")
				if strings.Contains(snapshot, "late-server-log") {
					t.Error("repeated cleanup acquired another server-log snapshot")
				}
			}
			if captured != 1 {
				t.Fatalf("post-join snapshots=%d after %d repeats, want exactly 1", captured, repeats)
			}
		})
	}
}

func TestStartupEvidenceServerLogHasTwoIndependentPhases(t *testing.T) {
	f, path := serverLogFixture(t)
	if err := os.WriteFile(path, []byte("before-marker"), 0600); err != nil {
		t.Fatal(err)
	}
	before := f.captureStartupServerLogOnce(&f.serverLogBefore)
	if err := os.WriteFile(path, []byte("after-marker"), 0600); err != nil {
		t.Fatal(err)
	}
	after := f.captureStartupLog()
	if !strings.Contains(before, "before-marker") || !strings.Contains(after, "after-marker") {
		t.Fatal("phase snapshots did not retain independent bytes")
	}
	if f.captureStartupServerLogOnce(&f.serverLogBefore) != "" || strings.Contains(f.captureStartupLog(), "zellij_server_log") {
		t.Fatal("capture budget exceeded two snapshots")
	}
}

// Embedded TB keeps real test reporting while selecting the cleanup failure
// branch without failing the parent test itself.
type startupLogTB struct {
	testing.TB
	failed  bool
	logs    []string
	observe func(string)
}

func (b *startupLogTB) Failed() bool { return b.failed }
func (b *startupLogTB) Logf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	b.logs = append(b.logs, line)
	if b.observe != nil {
		b.observe(line)
	}
}

// Close evaluates Done when entering its admitted-call join. Complete the held
// producer there so post-join capture ordering is tested without a timed race.
type startupJoinContext struct {
	context.Context
	once sync.Once
	join func()
}

func (c *startupJoinContext) Done() <-chan struct{} {
	c.once.Do(c.join)
	return c.Context.Done()
}

func TestStartupEvidenceServerLogCloseLifecycle(t *testing.T) {
	for _, failed := range []bool{false, true} {
		for _, optIn := range []bool{false, true} {
			t.Run(fmt.Sprintf("failed=%t/opt-in=%t", failed, optIn), func(t *testing.T) {
				f, path := serverLogFixture(t)
				write := func(marker string) {
					t.Helper()
					if err := os.WriteFile(path, []byte(marker), 0600); err != nil {
						t.Fatal(err)
					}
				}
				write("before-cancel-marker")
				want := failed || optIn
				log := &startupLogTB{TB: t, failed: failed}
				f.t = log
				f.Options.RetainStartupServerLog = optIn
				f.RunID = "unit-server-log-close-" + filepath.Base(f.Roots.Root)
				f.idle = make(chan struct{})
				f.model = ReduceRun(OwnedRun{}, RunEvent{Kind: "ready"})
				canceled := false
				f.cancel = func() {
					before := strings.Join(log.logs, "\n")
					if got := strings.Contains(before, "before-cancel-marker"); got != want {
						t.Errorf("pre-cancellation capture=%t, want %t", got, want)
					}
					canceled = true
					write("not-yet-joined-marker")
				}
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				defer cancel()
				joined := &startupJoinContext{Context: ctx, join: func() {
					if !canceled {
						t.Error("join preceded cancellation")
					}
					write("after-join-marker")
					close(f.idle)
				}}
				log.observe = func(line string) {
					if !strings.Contains(line, "zellij_server_log") {
						return
					}
					if _, err := os.Stat(f.Roots.Root); err != nil {
						t.Errorf("capture after root removal: %v", err)
					}
					if strings.Contains(line, "after join:") {
						select {
						case <-f.idle:
						default:
							t.Error("final capture preceded producer join")
						}
						if !strings.Contains(line, "after-join-marker") {
							t.Error("final producer bytes were not retained")
						}
					}
				}
				if err := f.Close(joined); err != nil {
					t.Fatal(err)
				}
				if err := f.AssertCleaned(); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(f.Roots.Root); !os.IsNotExist(err) {
					t.Fatalf("root not removed: %v", err)
				}
				beforeRepeat := strings.Join(log.logs, "\n")
				for range 16 {
					if err := f.Close(joined); err != nil {
						t.Fatal(err)
					}
				}
				if got := strings.Join(log.logs, "\n"); got != beforeRepeat {
					t.Fatal("repeated Close captured additional evidence")
				}
				count := strings.Count(beforeRepeat, "zellij_server_log")
				expected := 0
				if want {
					expected = 2
				}
				if count != expected {
					t.Errorf("snapshots=%d want %d", count, expected)
				}
			})
		}
	}
}
