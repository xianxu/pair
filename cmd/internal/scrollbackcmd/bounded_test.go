package scrollbackcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

func TestOwnedCaptureLeaseProtectsReplayAndReleases(t *testing.T) {
	for _, failReplay := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%t", failReplay), func(t *testing.T) {
			ctx := context.Background()
			c, err := storagegc.NewCoordinator(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
			c.Now = func() time.Time { return now }
			scope, tag := "0123456789abcdef", "couch-0123456789abcdef"
			owner, err := artifactpath.NewStorageOwner(c.Root, scope, tag)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(owner.Directory(), 0700); err != nil {
				t.Fatal(err)
			}
			if err := c.Initialize(ctx, owner); err != nil {
				t.Fatal(err)
			}
			if err := c.CompleteMigration(ctx, nil); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.ResolveScoped(owner.Directory(), tag)
			if err != nil {
				t.Fatal(err)
			}
			selected, _ := paths.ParkedScrollbackArtifacts("20260913T010101")
			other, _ := paths.ParkedScrollbackArtifacts("20260913T020202")
			for _, capture := range []artifactpath.ParkedScrollbackArtifactSet{selected, other} {
				events := `{"type":"resize","cols":20,"rows":2}`
				if failReplay && capture.Raw == selected.Raw {
					events = `{"type":"resize","cols":1001,"rows":2}`
				}
				for path, body := range map[string]string{capture.Raw: "selected history", capture.Events: events} {
					if err := os.WriteFile(path, []byte(body), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := c.PublishCaptureMetadata(ctx, owner, artifactpath.ParkedCapture{Raw: capture.Raw, Events: capture.Events, Metadata: capture.Metadata}, now.Add(-storagegc.CaptureRetentionPeriod-time.Hour)); err != nil {
					t.Fatal(err)
				}
			}
			gc := &storagegc.Collector{Coordinator: c, Agents: []string{"claude"}, Legacy: func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
				return storagegc.ProcessDead, nil
			}}
			collect := func(want int) {
				t.Helper()
				report, err := gc.Apply(ctx, 100)
				if err != nil || report.Collected != want {
					t.Fatalf("collection: %+v, %v; want %d collected", report, err, want)
				}
			}
			assertProtected := func() {
				t.Helper()
				state, err := c.ReadOwner(owner)
				if err != nil || len(state.Processes) != 1 || state.Processes[0].Target != selected.Raw || state.Processes[0].Role != "scrollback-reader" {
					t.Fatalf("replay lacks exact selected capture lease: %+v, %v", state, err)
				}
				for _, path := range []string{selected.Raw, selected.Events, selected.Metadata} {
					if _, err := os.Stat(path); err != nil {
						t.Fatalf("selected capture removed during replay: %s: %v", path, err)
					}
				}
			}
			original := replayOwnedCapture
			t.Cleanup(func() { replayOwnedCapture = original })
			called := false
			replayOwnedCapture = func(raw, events string, maxLines int) ([]string, error) {
				called = true
				if raw != selected.Raw || events != selected.Events || maxLines != 0 {
					t.Fatalf("wrong replay selection: %s %s %d", raw, events, maxLines)
				}
				assertProtected()
				collect(1) // The other old capture is collectible while this one is leased.
				assertProtected()
				if _, err := os.Stat(other.Raw); !os.IsNotExist(err) {
					t.Fatalf("unselected capture retained: %v", err)
				}
				lines, err := original(raw, events, maxLines)
				collect(0)
				assertProtected()
				return lines, err
			}
			lines, err := RenderOwnedCaptureLines(c.Root, scope, tag, "claude", "20260913T010101", 0)
			if !called || (err != nil) != failReplay {
				t.Fatalf("replay called=%t lines=%q error=%v", called, lines, err)
			}
			if failReplay {
				if !strings.Contains(err.Error(), "resize 1001x2") {
					t.Fatalf("unexpected replay failure: %v", err)
				}
			} else if strings.Join(lines, "\n") != "selected history" {
				t.Fatalf("wrong replay: %q", lines)
			}
			state, err := c.ReadOwner(owner)
			if err != nil || len(state.Processes) != 0 {
				t.Fatalf("replay leaked its lease: %+v, %v", state, err)
			}
			collect(1)
			for _, path := range []string{selected.Raw, selected.Events, selected.Metadata} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("capture retained after replay returned: %s: %v", path, err)
				}
			}
		})
	}
}

func boundedFixture(t *testing.T, raw, events string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	r, e := filepath.Join(dir, "capture.raw"), filepath.Join(dir, "capture.events.jsonl")
	for p, b := range map[string]string{r: raw, e: events} {
		if err := os.WriteFile(p, []byte(b), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return r, e
}
func TestBoundedMatchesReplay(t *testing.T) {
	r, e := boundedFixture(t, "\x1b[31mfirst\x1b[0m\r\nsecond\r\nthird\r\nfourth", `{"type":"resize","cols":20,"rows":2}`)
	for _, n := range []int{0, -1, 1, 100} {
		want, err := RenderLines(r, e, n)
		if err != nil {
			t.Fatal(err)
		}
		got, err := RenderBoundedLines(r, e, n)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("max %d: %q %v, want %q", n, got, err, want)
		}
	}
}
func TestBoundedRejectsUnsafeSources(t *testing.T) {
	for _, kind := range []string{"symlink", "fifo", "directory", "raw-size", "events-size", "missing-events"} {
		t.Run(kind, func(t *testing.T) {
			r, e := boundedFixture(t, "safe", "")
			source := r
			switch kind {
			case "symlink":
				source = r + ".link"
				if err := os.Symlink(r, source); err != nil {
					t.Fatal(err)
				}
				r = source
			case "fifo":
				source = r + ".fifo"
				if err := unix.Mkfifo(source, 0600); err != nil {
					t.Fatal(err)
				}
				r = source
			case "directory":
				r = filepath.Dir(r)
				source = r
			case "raw-size":
				if err := os.Truncate(r, 64<<20+1); err != nil {
					t.Fatal(err)
				}
			case "events-size":
				source = e
				if err := os.Truncate(e, 8<<20+1); err != nil {
					t.Fatal(err)
				}
			case "missing-events":
				e += ".missing"
				source = e
			}
			got, err := RenderBoundedLines(r, e, 0)
			if err == nil || len(got) != 0 || !strings.Contains(err.Error(), source) {
				t.Fatalf("got %q, %v; want rejection naming %s", got, err, source)
			}
		})
	}
}
func TestBoundedRejectsResizeBeforeReplay(t *testing.T) {
	for _, size := range [][2]int{{1001, 1}, {1, 1001}, {0, 5}, {5, -1}} {
		r, e := boundedFixture(t, "x", fmt.Sprintf(`{"type":"resize","cols":20,"rows":2}`+"\n"+`{"type":"resize","offset":1,"cols":%d,"rows":%d}`, size[0], size[1]))
		if _, err := RenderBoundedLines(r, e, 0); err == nil || !strings.Contains(err.Error(), e) {
			t.Fatalf("size %v: %v", size, err)
		}
	}
}
func TestBoundedRowsRejectInsteadOfReturningTail(t *testing.T) {
	// A small injected envelope exercises overflow without a large fixture.
	for _, suffix := range []string{""} {
		r, e := boundedFixture(t, "first\r\nsecond\r\nthird\r\nfourth"+suffix, `{"type":"resize","cols":20,"rows":1}`)
		got, err := renderBoundedLines(r, e, 0, 2)
		if err == nil || len(got) != 0 || !strings.Contains(err.Error(), r) || !strings.Contains(err.Error(), e) {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}
func TestOwnedCaptureSelectsExactToken(t *testing.T) {
	root := t.TempDir()
	scope, tag := "0123456789abcdef", "couch-0123456789abcdef"
	p, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: scope, Tag: tag})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	live, _ := p.ScrollbackArtifacts("claude")
	old, _ := p.ParkedScrollbackArtifacts("20260913T010101")
	for path, body := range map[string]string{live.Raw: "current", live.Events: "", old.Raw: "previous", old.Events: ""} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for token, want := range map[string]string{"": "current", "20260913T010101": "previous"} {
		got, err := RenderOwnedCaptureLines(root, scope, tag, "claude", token, 0)
		if err != nil || strings.Join(got, "\n") != want {
			t.Fatalf("token %q got %q %v", token, got, err)
		}
	}
	if _, err := RenderOwnedCaptureLines(root, scope, tag, "claude", "20260913T020202", 0); err == nil {
		t.Fatal("missing previous fell back")
	}
}

func TestBoundedBlankHistoryTail(t *testing.T) {
	r, e := boundedFixture(t, strings.Repeat("\r\n", 10), `{"type":"resize","cols":20,"rows":2}`)
	want, err := RenderLines(r, e, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderBoundedLines(r, e, 1)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q %v, want %q", got, err, want)
	}
}

func TestBoundedRowsAtLimitAreComplete(t *testing.T) {
	r, e := boundedFixture(t, "first\r\nsecond", `{"type":"resize","cols":20,"rows":1}`)
	got, err := renderBoundedLines(r, e, 0, 2)
	if err != nil || !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("got %q %v", got, err)
	}
}
