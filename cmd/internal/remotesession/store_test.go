package remotesession

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

func TestStoreLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latch")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if p, g := s.Snapshot(); p != latchOff || g != 0 {
		t.Fatalf("initial %s %d", p, g)
	}
	if err := s.consoleReset(); err == nil {
		t.Fatal("reset allowed while off")
	}
	g, err := s.Arm()
	if err != nil || g != 1 {
		t.Fatalf("arm %d %v", g, err)
	}
	if _, err := s.Arm(); err == nil {
		t.Fatal("double arm")
	}
	recovered, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := recovered.Snapshot(); p != latchKilled {
		t.Fatal("unclean startup not killed")
	}
	if err := s.stop(g + 1); err == nil {
		t.Fatal("stale stop")
	}
	if err := s.stop(g); err != nil {
		t.Fatal(err)
	}
	g, err = s.Arm()
	if err != nil || g != 2 {
		t.Fatalf("rearm %d %v", g, err)
	}
	if err := s.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := s.stop(g); err == nil {
		t.Fatal("stop overwrote kill")
	}
	if _, err := s.Arm(); err == nil {
		t.Fatal("arm cleared kill")
	}
	if err := s.consoleReset(); err != nil {
		t.Fatal(err)
	}
	recovered, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if p, g := recovered.Snapshot(); p != latchOff || g != 2 {
		t.Fatalf("reset %s %d", p, g)
	}
}

func TestStorePublicationFault(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before rename", true: "after rename"}[after], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "latch")
			s, err := OpenStore(path)
			if err != nil {
				t.Fatal(err)
			}
			s.publish = func(path string, raw []byte) error {
				if after {
					if err := durablefile.WriteAtomic(path, raw, ".test-*"); err != nil {
						t.Fatal(err)
					}
				}
				return errors.New("injected publication failure")
			}
			if _, err := s.Arm(); err == nil {
				t.Fatal("failed publication granted arm")
			}
			if p, _ := s.Snapshot(); p != latchKilled {
				t.Fatal("failure did not block owner")
			}
			s.publish = writeLatch
			if _, err := s.Arm(); err == nil {
				t.Fatal("retry granted after uncertainty")
			}
			if err := s.consoleReset(); err == nil {
				t.Fatal("uncertainty reset without reopen")
			}
			reopened, err := OpenStore(path)
			if err != nil {
				t.Fatal(err)
			}
			p, _ := reopened.Snapshot()
			if after && p != latchKilled {
				t.Fatal("published arm recovered as enabled")
			}
		})
	}
}

func TestStoreConcurrentKillAndStop(t *testing.T) {
	for i := 0; i < 32; i++ {
		s, err := OpenStore(filepath.Join(t.TempDir(), "latch"))
		if err != nil {
			t.Fatal(err)
		}
		g, err := s.Arm()
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _ = s.stop(g) }()
		go func() {
			defer wg.Done()
			if err := s.Kill(); err != nil {
				t.Error(err)
			}
		}()
		wg.Wait()
		if p, _ := s.Snapshot(); p != latchKilled {
			t.Fatal("stop defeated kill")
		}
	}
}

func TestStoreUnsafeInputUnchanged(t *testing.T) {
	for _, raw := range []string{"", `{}`, `{"version":2,"phase":"off","generation":0}`, string(make([]byte, 4097))} {
		t.Run(raw[:min(len(raw), 20)], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "latch")
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenStore(path); err == nil {
				t.Fatal("accepted invalid latch")
			}
			got, _ := os.ReadFile(path)
			if string(got) != raw {
				t.Fatal("rewrote evidence")
			}
		})
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	path := filepath.Join(root, "link")
	_ = os.WriteFile(target, []byte(`{"version":1,"phase":"off","generation":0}`), 0600)
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStore(path); err == nil {
		t.Fatal("accepted symlink")
	}
	if _, err := OpenStore(root); err == nil {
		t.Fatal("accepted directory")
	}
}

func TestStoreEveryTransitionPublicationFault(t *testing.T) {
	for _, action := range []string{"arm", "kill", "stop", "reset"} {
		for _, after := range []bool{false, true} {
			t.Run(action+map[bool]string{false: "/before", true: "/after"}[after], func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "latch")
				s, err := OpenStore(path)
				if err != nil {
					t.Fatal(err)
				}
				var g uint64
				if action != "arm" {
					g, err = s.Arm()
					if err != nil {
						t.Fatal(err)
					}
				}
				if action == "reset" {
					if err := s.Kill(); err != nil {
						t.Fatal(err)
					}
				}
				before, _ := os.ReadFile(path)
				s.publish = func(path string, raw []byte) error {
					// Real publication models rename success followed by directory sync
					// error. No publication models every pre-rename write/sync error.
					if after {
						if err := writeLatch(path, raw); err != nil {
							t.Fatal(err)
						}
					}
					return errors.New("injected failure")
				}
				switch action {
				case "arm":
					_, err = s.Arm()
				case "kill":
					err = s.Kill()
				case "stop":
					err = s.stop(g)
				case "reset":
					err = s.consoleReset()
				}
				if err == nil {
					t.Fatal("fault succeeded")
				}
				if p, _ := s.Snapshot(); p != latchKilled {
					t.Fatal("failed publication retained authority")
				}
				if err := s.stop(g); err == nil {
					t.Fatal("poisoned stop succeeded")
				}
				if err := s.Kill(); err == nil {
					t.Fatal("poisoned kill claimed durable success")
				}
				if err := s.consoleReset(); err == nil {
					t.Fatal("poisoned reset succeeded")
				}
				if _, err := s.Arm(); err == nil {
					t.Fatal("poisoned arm succeeded")
				}
				if !after {
					now, _ := os.ReadFile(path)
					if string(now) != string(before) {
						t.Fatal("prepublication failure changed bytes")
					}
				}
				recovered, err := OpenStore(path)
				if err != nil {
					t.Fatal(err)
				}
				want := latchKilled
				if after && (action == "stop" || action == "reset") || !after && action == "arm" {
					want = latchOff
				}
				if p, _ := recovered.Snapshot(); p != want {
					t.Fatalf("recovered %s want %s", p, want)
				}
			})
		}
	}
}

func TestStoreGenerationExhaustion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latch")
	raw := []byte(`{"version":1,"phase":"off","generation":18446744073709551615}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Arm(); err == nil {
		t.Fatal("generation wrapped")
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(raw) {
		t.Fatal("exhaustion changed disk")
	}
}

func TestStorePublishesPrivateArmedBeforeGrant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "latch")
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	g, err := s.Arm()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	durable, err := decodeLatch(raw)
	if err != nil {
		t.Fatal(err)
	}
	if durable.Phase != latchArmed || durable.Generation != g {
		t.Fatal("grant lacks armed disk record")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("latch permissions %o", info.Mode().Perm())
	}
}
