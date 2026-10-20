package couchscratch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Manager, string, time.Time) {
	t.Helper()
	root := t.TempDir()
	now := time.Now().Truncate(time.Second)
	m := New(Options{Probe: func() Probe { return Probe{Root: root, Mounted: true, FreeBytes: Floor} }, Now: func() time.Time { return now }})
	env := m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3})
	if len(env) != 3 {
		t.Fatalf("overlay %v", env)
	}
	return m, strings.TrimPrefix(env[0], "TMPDIR="), now
}
func TestPrepareFallbackAndOwnership(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	for _, p := range []Probe{{Root: root}, {Root: root, Mounted: true, FreeBytes: Floor - 1}} {
		m := New(Options{Probe: func() Probe { return p }})
		if got := m.PrepareSlot(Slot{Repo: "pair", Identity: "a", Number: 0}); got != nil {
			t.Fatal(got)
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatalf("fallback created directory: %v", err)
		}
	}
	m, path, _ := fixture(t)
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0700 {
		t.Fatal(fi.Mode())
	}
	if got := m.PrepareSlot(Slot{Repo: "pair", Identity: "different", Number: 3}); got != nil {
		t.Fatal("identity conflict reused directory")
	}
}
func TestSweepAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name       string
		old, fresh bool
		removed    int
	}{{"fresh", false, false, 0}, {"old", true, false, 1}, {"fresh descendant", true, true, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			m, root, now := fixture(t)
			tree := filepath.Join(root, "entry")
			os.Mkdir(tree, 0700)
			file := filepath.Join(tree, "file")
			os.WriteFile(file, []byte("x"), 0600)
			if tc.old {
				old := now.Add(-25 * time.Hour)
				os.Chtimes(file, old, old)
				os.Chtimes(tree, old, old)
			}
			if tc.fresh {
				os.Chtimes(file, now, now)
			}
			got, err := m.Sweep(context.Background())
			if err != nil || got.Removed != tc.removed {
				t.Fatalf("%+v %v", got, err)
			}
			if _, err := os.Stat(root); err != nil {
				t.Fatal("slot root removed", err)
			}
		})
	}
}
func TestSweepSpecialAndChangedTreesRetained(t *testing.T) {
	m, root, now := fixture(t)
	old := now.Add(-25 * time.Hour)
	tree := filepath.Join(root, "special")
	os.Mkdir(tree, 0700)
	os.Symlink("missing", filepath.Join(tree, "link"))
	os.Chtimes(tree, old, old)
	changed := filepath.Join(root, "changed")
	os.Mkdir(changed, 0700)
	file := filepath.Join(changed, "file")
	os.WriteFile(file, nil, 0600)
	os.Chtimes(file, old, old)
	os.Chtimes(changed, old, old)
	m.beforeRemove = func() { os.Chtimes(file, now, now) }
	got, err := m.Sweep(context.Background())
	if err != nil || got.Removed != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}
func TestRunCancellation(t *testing.T) {
	m, _, _ := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not join")
	}
}

func TestOwnerUnknownFieldsArePreserved(t *testing.T) {
	m, root, _ := fixture(t)
	declaration := filepath.Join(filepath.Dir(filepath.Dir(root)), "couch-scratch", "owners", "pair-slot3.json")
	data := []byte(`{"Repo":"pair","Identity":"git-common","Number":3,"future":{"bytes":[1,2]}}`)
	if err := os.WriteFile(declaration, data, 0600); err != nil {
		t.Fatal(err)
	}
	if got := m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3}); len(got) != 3 {
		t.Fatal(got)
	}
	got, err := os.ReadFile(declaration)
	if err != nil || string(got) != string(data) {
		t.Fatalf("owner rewritten: %s %v", got, err)
	}
	for _, bad := range []string{`{"Repo":"pair","Repo":"pair","Identity":"git-common","Number":3}`, `{"Repo":"pair","Identity":"git-common","Number":"3"}`} {
		os.WriteFile(declaration, []byte(bad), 0600)
		if got := m.PrepareSlot(Slot{Repo: "pair", Identity: "git-common", Number: 3}); got != nil {
			t.Fatal("accepted malformed known field")
		}
	}
}
func TestMetadataVisitsLinearAcrossSlots(t *testing.T) {
	for _, slots := range []int{8, 16} {
		t.Run(strconv.Itoa(slots), func(t *testing.T) {
			m, _, now := fixture(t)
			probe := m.options.Probe()
			old := now.Add(-25 * time.Hour)
			files := 0
			for i := 0; i < slots; i++ {
				env := m.PrepareSlot(Slot{Repo: "repo", Identity: "same", Number: i})
				root := strings.TrimPrefix(env[0], "TMPDIR=")
				tree := filepath.Join(root, "tree")
				os.Mkdir(tree, 0700)
				for j := 0; j < 8; j++ {
					file := filepath.Join(tree, strconv.Itoa(j))
					os.WriteFile(file, nil, 0600)
					os.Chtimes(file, old, old)
					files++
				}
				os.Chtimes(tree, old, old)
				files++
			}
			got, err := m.Sweep(context.Background())
			if err != nil || got.Removed != slots || got.Visited > 2*files+slots || got.Inspections > 4*files+4*slots+4 || got.BookkeepingReads != slots+1 || got.TempPayloadReads != 0 {
				t.Fatalf("%+v %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(probe.Root, "couch-scratch", "owners")); err != nil {
				t.Fatal("bookkeeping removed")
			}
		})
	}
}
func TestHourlyTickSweepsAndJoins(t *testing.T) {
	m, root, now := fixture(t)
	ticks := make(chan time.Time)
	m.options.Ticks = ticks
	calls := make(chan struct{}, 4)
	probe := m.options.Probe
	m.options.Probe = func() Probe { calls <- struct{}{}; return probe() }
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()
	// The probe runs with the sweep lock held: receiving its signal, then
	// acquiring the lock, joins the corresponding complete sweep.
	<-calls
	m.mu.Lock()
	file := filepath.Join(root, "old")
	os.WriteFile(file, nil, 0600)
	old := now.Add(-25 * time.Hour)
	os.Chtimes(file, old, old)
	m.mu.Unlock()
	select {
	case ticks <- now:
	case <-time.After(time.Second):
		t.Fatal("tick not consumed")
	}
	<-calls
	m.mu.Lock()
	m.mu.Unlock()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("did not join")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("tick retained old file", err)
	}
}

func TestMaintainDoesNotRepeatStartupSweep(t *testing.T) {
	m, root, now := fixture(t)
	path := filepath.Join(root, "old")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-25 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time)
	close(ticks)
	m.options.Ticks = ticks
	m.Maintain(context.Background())
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("maintenance performed unsolicited startup sweep: %v", err)
	}
}

func TestMaintainSweepsOnPrivateTickAndJoins(t *testing.T) {
	m, root, now := fixture(t)
	file := filepath.Join(root, "old")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-25 * time.Hour)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time)
	m.options.Ticks = ticks
	calls := make(chan struct{}, 1)
	probe := m.options.Probe
	m.options.Probe = func() Probe { calls <- struct{}{}; return probe() }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { m.Maintain(ctx); close(done) }()
	select {
	case ticks <- now:
	case <-time.After(time.Second):
		t.Fatal("maintenance did not accept tick")
	}
	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("tick did not trigger sweep")
	}
	m.mu.Lock()
	m.mu.Unlock()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("maintenance did not join")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("tick left old payload: %v", err)
	}
}

func TestSweepReportsOneBoundedAggregate(t *testing.T) {
	m, root, now := fixture(t)
	var notices []string
	m.options.Notify = func(s string) { notices = append(notices, s) }
	for _, name := range []string{"a", "b"} {
		path := filepath.Join(root, name)
		os.WriteFile(path, nil, 0600)
		old := now.Add(-25 * time.Hour)
		os.Chtimes(path, old, old)
	}
	got, err := m.Sweep(context.Background())
	if err != nil || got.Removed != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "2") || len(notices[0]) > 512 {
		t.Fatalf("aggregate: %q", notices)
	}
	m.Sweep(context.Background())
	if len(notices) != 1 {
		t.Fatalf("untouched entries produced noise: %q", notices)
	}
	probe := m.options.Probe()
	probe.Root = filepath.Join(probe.Root, "missing")
	m.options.Probe = func() Probe { return probe }
	if _, err = m.Sweep(context.Background()); err == nil {
		t.Fatal("expected filesystem error")
	}
	if len(notices) != 2 || len(notices[1]) > 512 {
		t.Fatalf("error aggregate: %q", notices)
	}
}

func TestPrepareFallsBackWhileSweepIsPaused(t *testing.T) {
	m, root, now := fixture(t)
	file := filepath.Join(root, "old")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-25 * time.Hour)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	m.beforeRemove = func() { close(entered); <-release }
	swept := make(chan struct{})
	go func() { m.Sweep(context.Background()); close(swept) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("sweep did not reach pause")
	}
	prepared := make(chan []string, 1)
	resolved := false
	go func() {
		prepared <- m.PrepareResolvedSlot(func() (Slot, error) { resolved = true; return Slot{Repo: "other", Identity: "other", Number: 8}, nil })
	}()
	select {
	case env := <-prepared:
		close(release)
		<-swept
		if env != nil {
			t.Fatalf("busy manager adopted scratch: %v", env)
		}
		if resolved {
			t.Fatal("busy manager resolved optional identity")
		}
	case <-time.After(time.Second):
		close(release)
		<-swept
		<-prepared
		t.Fatal("slot preparation waited behind paused sweep")
	}
}

func TestPrepareResolvedSlotSkipsUnavailableAndBoundsErrors(t *testing.T) {
	for _, p := range []Probe{{Root: "/absent"}, {Root: "/absent", Mounted: true, FreeBytes: Floor - 1}} {
		m := New(Options{Probe: func() Probe { return p }})
		called := false
		if env := m.PrepareResolvedSlot(func() (Slot, error) { called = true; return Slot{}, nil }); env != nil || called {
			t.Fatalf("unavailable scratch resolved identity: %v %v", env, called)
		}
	}
	probes := 0
	notices := []string{}
	root := t.TempDir()
	m := New(Options{Probe: func() Probe { probes++; return Probe{Root: root, Mounted: true, FreeBytes: Floor} }, Notify: func(s string) { notices = append(notices, s) }})
	if env := m.PrepareResolvedSlot(func() (Slot, error) { return Slot{}, fmt.Errorf("%s", strings.Repeat("resolver error", 1000)) }); env != nil {
		t.Fatal(env)
	}
	if probes != 1 || len(notices) != 1 || len(notices[0]) > 512 {
		t.Fatalf("probe/notice mismatch: %d %q", probes, notices)
	}
}
