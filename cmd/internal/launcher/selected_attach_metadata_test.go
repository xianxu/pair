package launcher

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"golang.org/x/sys/unix"
)

// Keep stateful listing/handoff facts, but exercise the production filesystem
// methods rather than a fake ReadFile that cannot model a FIFO open/read.
type selectedMetadataRuntime struct {
	*couchSessionRuntime
	os         *OSRuntime
	indexReads int
}

func (r *selectedMetadataRuntime) InferAgent(tag string) string { return r.os.InferAgent(tag) }
func (r *selectedMetadataRuntime) Touch(path string) error      { return r.os.Touch(path) }
func (r *selectedMetadataRuntime) KillTitlePoller(tag string)   { r.os.KillTitlePoller(tag) }
func (r *selectedMetadataRuntime) ReadSessionNameIndex() (SessionNameIndex, error) {
	r.indexReads++
	return r.os.ReadSessionNameIndex()
}
func (r *selectedMetadataRuntime) TakeQuitIntent(session string) (QuitIntent, bool, error) {
	return r.os.TakeQuitIntent(session)
}
func (r *selectedMetadataRuntime) WriteQuitIntent(session string, intent QuitIntent) error {
	return r.os.WriteQuitIntent(session, intent)
}
func (r *selectedMetadataRuntime) ReadRestartMarker(session string) (RestartMarker, bool, error) {
	return r.os.ReadRestartMarker(session)
}

// A writer keeps a legacy blocking reader asleep until the deadline, then its
// close releases that reader so a RED run joins the goroutine instead of leaking.
func holdMetadataFIFO(t *testing.T, path string) *os.File {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	f := os.NewFile(uintptr(fd), path)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestSelectedAttachProductionMetadataDoesNotBlockHandoff(t *testing.T) {
	for _, entry := range []string{"RunLaunch", "runOnce"} {
		for _, kind := range []string{"ledger", "agent", "index", "draft", "failed-poller", "quit", "restart", "changed-at-effect", "healthy"} {
			if entry == "runOnce" && (kind == "index" || kind == "quit" || kind == "restart") {
				continue
			}
			t.Run(entry+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				t.Setenv("HOME", root)
				opts := managedSessionOptions("attach")
				opts.GlobalDataDir = root
				opts.Env.DataDir = root
				rt := &selectedMetadataRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}, os: NewOSRuntime(root, "/pair")}
				rt.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
				paths, err := artifactpath.ResolveScoped(root, opts.Args.ForcedTag)
				if err != nil {
					t.Fatal(err)
				}
				var fifo *os.File
				switch kind {
				case "ledger":
					fifo = holdMetadataFIFO(t, paths.Ledger())
				case "agent":
					fifo = holdMetadataFIFO(t, paths.Agent())
				case "index":
					scope, err := artifactpath.ResolveSelectedScope(root)
					if err != nil {
						t.Fatal(err)
					}
					fifo = holdMetadataFIFO(t, scope.SessionBindings())
				case "draft":
					if err := unix.Mkfifo(paths.Draft(), 0600); err != nil {
						t.Fatal(err)
					}
				case "failed-poller":
					fifo = holdMetadataFIFO(t, paths.TitlePID())
					rt.attachCode = 7
				case "quit":
					path, _ := quitMarkerPath(opts.Args.CouchSession.Name)
					fifo = holdMetadataFIFO(t, path)
				case "restart":
					path, _ := restartMarkerPath(opts.Args.CouchSession.Name)
					fifo = holdMetadataFIFO(t, path)
				case "changed-at-effect":
					rt.changeDuringCmux = true
				case "healthy":
					line, err := BuildLedgerLine(LedgerEntry{Agent: "codex", SessionID: "retained", LastActive: time.Unix(40, 0)})
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(paths.Ledger(), []byte(line+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				done := make(chan int, 1)
				go func() {
					var code int
					if entry == "RunLaunch" {
						code, _ = RunLaunch(opts, rt, io.Discard)
					} else {
						step, _ := runOnce(opts, normalizeEnv(opts.Env), rt, io.Discard)
						code = step.code
						_ = finishLaunchRetention(step, io.Discard)
					}
					done <- code
				}()
				var code int
				select {
				case code = <-done:
				case <-time.After(500 * time.Millisecond):
					if kind == "draft" {
						fd, err := unix.Open(paths.Draft(), unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
						if err != nil {
							t.Fatal(err)
						}
						fifo = os.NewFile(uintptr(fd), paths.Draft())
						defer fifo.Close()
					}
					if fifo != nil && kind != "draft" {
						_ = fifo.Close()
					}
					select {
					case <-done:
					case <-time.After(time.Second):
						t.Fatal("metadata read did not stop after FIFO writer closed")
					}
					t.Fatal("selected attachment blocked on optional " + kind + " metadata")
				}
				if kind == "changed-at-effect" {
					if code == 0 || len(rt.attached) != 0 || rt.launchCount != 0 {
						t.Fatalf("changed listing reached handoff: code=%d attached=%v creates=%d", code, rt.attached, rt.launchCount)
					}
					return
				}
				if len(rt.attached) != 1 || rt.attached[0] != opts.Args.CouchSession.Name || rt.launchCount != 0 {
					t.Fatalf("handoff=%v creates=%d code=%d", rt.attached, rt.launchCount, code)
				}
				if rt.proofMigrations != 0 || rt.indexReads != 0 || len(rt.swept) != 0 {
					t.Fatalf("selected attachment ran unrelated startup lifecycle: migration=%d index=%d sweeps=%d", rt.proofMigrations, rt.indexReads, len(rt.swept))
				}
				if kind == "healthy" {
					if rt.env["PAIR_AGENT"] != "codex" || !strings.Contains(strings.Join(rt.pollers, " "), "|codex") {
						t.Fatalf("healthy title agent lost: env=%v pollers=%v", rt.env, rt.pollers)
					}
					if _, err := os.Stat(paths.Draft()); err != nil {
						t.Fatal("healthy draft missing:", err)
					}
				}
			})
		}
	}
}

func TestOSRuntimeLifecycleMetadataRejectsOversizeAndKeepsLegacyQuit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	rt := NewOSRuntime(root, "/pair")
	quit, _ := quitMarkerPath("pair-work")
	if err := os.MkdirAll(filepath.Dir(quit), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(quit, nil, 0600); err != nil {
		t.Fatal(err)
	}
	intent, present, err := rt.TakeQuitIntent("pair-work")
	if err != nil || !present || intent.Kind != QuitIntentDirect {
		t.Fatalf("legacy intent=%+v present=%v err=%v", intent, present, err)
	}
	if err := os.WriteFile(quit, bytes.Repeat([]byte(" "), 64*1024+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := rt.TakeQuitIntent("pair-work"); err == nil {
		t.Fatal("accepted oversized quit intent")
	}
	restart, _ := restartMarkerPath("pair-work")
	if err := os.WriteFile(restart, bytes.Repeat([]byte(" "), maxRestartMarkerBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := rt.ReadRestartMarker("pair-work"); err == nil {
		t.Fatal("accepted oversized restart marker")
	}
}

func TestOSRuntimeInferAgentBoundsLocalPresentationMetadata(t *testing.T) {
	for _, kind := range []string{"ledger oversize", "agent oversize", "ledger symlink", "typed launch with native FIFO"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("HOME", root)
			rt := NewOSRuntime(root, "/pair")
			paths, err := artifactpath.ResolveScoped(root, "work")
			if err != nil {
				t.Fatal(err)
			}
			want := "codex"
			if err := os.WriteFile(paths.Agent(), []byte("codex\n"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "ledger oversize":
				line, err := BuildLedgerLine(LedgerEntry{Agent: "claude"})
				if err != nil {
					t.Fatal(err)
				}
				raw := append(bytes.Repeat([]byte(" "), 4*1024*1024), []byte(line+"\n")...)
				if err := os.WriteFile(paths.Ledger(), raw, 0600); err != nil {
					t.Fatal(err)
				}
			case "agent oversize":
				want = ""
				if err := os.WriteFile(paths.Agent(), bytes.Repeat([]byte("x"), 4097), 0600); err != nil {
					t.Fatal(err)
				}
			case "ledger symlink":
				target := filepath.Join(root, "unrelated-ledger")
				line, err := BuildLedgerLine(LedgerEntry{Agent: "claude"})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, []byte(line+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, paths.Ledger()); err != nil {
					t.Fatal(err)
				}
			case "typed launch with native FIFO":
				want = "claude"
				const id = "11111111-1111-4111-8111-111111111111"
				launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: want, RequestedNativeID: id, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(paths.Ledger(), append(launch, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
				native := filepath.Join(root, ".claude", "projects", "-repo", id+".jsonl")
				holdMetadataFIFO(t, native)
			}
			if got := rt.InferAgent("work"); got != want {
				t.Fatalf("InferAgent=%q want=%q", got, want)
			}
		})
	}
}

func TestOSRuntimeTouchPreservesRegularDraftAndRefusesSymlink(t *testing.T) {
	root := t.TempDir()
	rt := NewOSRuntime(root, "/pair")
	draft := filepath.Join(root, "draft")
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("retained draft"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rt.Touch(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, draft); err != nil {
		t.Fatal(err)
	}
	if err := rt.Touch(draft); err == nil {
		t.Fatal("Touch followed substituted draft symlink")
	}
	if raw, err := os.ReadFile(target); err != nil || string(raw) != "retained draft" {
		t.Fatalf("draft changed: %q %v", raw, err)
	}
}
