package couchtty

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

const menuPerfRows = 100

func menu100Fixture() []couchcore.ActionableThreadSummary {
	rows := make([]couchcore.ActionableThreadSummary, menuPerfRows)
	for i := range rows {
		rows[i] = couchcore.ActionableThreadSummary{
			Address:     menuAddress(fmt.Sprintf("couch-%03d", i)),
			WorkingPath: fmt.Sprintf("/repo/thread-%03d", i),
			Name:        fmt.Sprintf("thread-%03d", i),
			State:       couchcore.ThreadLive,
		}
	}
	return rows
}

func TestMenu100Bounds(t *testing.T) {
	rows := menu100Fixture()
	state := NewMenuState(rows, rows[0].Address)
	now := time.Unix(1_800_000_000, 0)

	if got := testing.AllocsPerRun(100, func() {
		_ = RenderMenu(state, 120, 40, now, true)
	}); got > 800 {
		t.Fatalf("100-row render allocations = %.0f, want <= 800", got)
	}
	if got := testing.AllocsPerRun(100, func() {
		next, _ := reduceKey(state, PanelKey{Kind: KeyRune, Rune: 't'})
		_ = next
	}); got > 40 {
		t.Fatalf("100-row filter allocations = %.0f, want <= 40", got)
	}
	if len(state.Inventory) != menuPerfRows || len(state.Frames) != 1 {
		t.Fatalf("fixture/state bounds changed: rows=%d frames=%d", len(state.Inventory), len(state.Frames))
	}
	if got := RenderMenu(state, 39, 10, now, false); got != "resize terminal to at least 40x10" {
		t.Fatalf("minimum-width refusal = %q", got)
	}
}

func TestMenuFeedbackBounds(t *testing.T) {
	rows := menu100Fixture()
	state := NewMenuState(rows, rows[0].Address)
	state, effects := dispatchThreadOperation(state, "switch", rows[0].Address)
	if len(effects) != 1 || state.Notice.Level != MenuNoticeProgress || state.Notice.Owner.OperationAttempt == 0 {
		t.Fatalf("bounded feedback dispatch = state %+v effects %+v", state, effects)
	}
	before := state.Notice
	for i := 0; i < 100; i++ {
		state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	}
	if state.Notice != before || len(state.Frames) != 1 || len(state.Inventory) != menuPerfRows {
		t.Fatalf("input accumulated work during progress: notice=%+v frames=%d rows=%d", state.Notice, len(state.Frames), len(state.Inventory))
	}
}

func BenchmarkMenu100(b *testing.B) {
	rows := menu100Fixture()
	now := time.Unix(1_800_000_000, 0)
	base := NewMenuState(rows, rows[0].Address)

	b.Run("open", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state := NewMenuState(rows, rows[0].Address)
			_ = RenderMenu(state, 120, 40, now, true)
		}
	})
	b.Run("filter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state, _ := reduceKey(base, PanelKey{Kind: KeyRune, Rune: '9'})
			_ = RenderMenu(state, 120, 40, now, true)
		}
	})
	b.Run("navigation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state, _ := reduceKey(base, PanelKey{Kind: KeyDown})
			_ = RenderMenu(state, 120, 40, now, true)
		}
	})
	b.Run("render", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = RenderMenu(base, 120, 40, now, true)
		}
	})
	b.Run("refresh", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state, _ := ReduceMenu(base, MenuEvent{Kind: MenuEventInventory, Inventory: rows})
			_ = RenderMenu(state, 120, 40, now, true)
		}
	})
	b.Run("feedback", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			state, _ := dispatchThreadOperation(base, "switch", rows[0].Address)
			_ = RenderMenu(state, 120, 40, now, true)
		}
	})
}

type menuPerfStats struct {
	p50 time.Duration
	p95 time.Duration
	max time.Duration
}

func TestMenuTargetPerformance(t *testing.T) {
	if os.Getenv("PAIR_MENU_PERF_TARGET") != "m2-max" {
		t.Skip("set PAIR_MENU_PERF_TARGET=m2-max on the target Apple M2 Max")
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Fatalf("target protocol requires darwin/arm64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	rows := menu100Fixture()
	trials := []struct {
		name string
		load bool
	}{{name: "baseline"}, {name: "load-1", load: true}, {name: "load-2", load: true}}
	for _, trial := range trials {
		t.Run(trial.name, func(t *testing.T) {
			stopLoad := func() {}
			if trial.load {
				stopLoad = startMenuCPULoad(4)
			}
			defer stopLoad()

			f := newFixture(t, 41, 120)
			epoch := uint64(0)
			resetConsole := func(focus Focus, markerRow int) string {
				deadline := time.Now().Add(time.Second)
				for {
					f.con.mu.Lock()
					idle := f.con.menu.InFlight.Operation == ""
					if idle {
						epoch++
						inventory := append([]couchcore.ActionableThreadSummary(nil), rows...)
						marker := fmt.Sprintf("epoch-%d-row-%03d", epoch, markerRow)
						inventory[markerRow].Name = marker
						f.con.menu = NewMenuState(inventory, rows[0].Address)
						f.con.menuReady = true
						f.con.focus = focus
						f.con.size = ptychild.Size{Rows: 41, Cols: 120}
					}
					f.con.mu.Unlock()
					if idle {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("prior performance operation did not finish")
					}
					runtime.Gosched()
				}
				f.host.Reset()
				return fmt.Sprintf("epoch-%d-row-%03d", epoch, markerRow)
			}
			generation := uint64(0)
			expected := ""
			refreshRows := []couchcore.ActionableThreadSummary(nil)
			paths := map[string]menuLifecyclePath{
				"open": {
					prepare: func() { expected = resetConsole(FocusActor("c1"), 0) },
					trigger: func() { _, _ = f.stdin.Write([]byte{0}) },
					paint:   func(frame string) bool { return strings.Contains(frame, expected) },
				},
				"filter": {
					prepare: func() { expected = resetConsole(FocusPanel(), 9) },
					trigger: func() { _, _ = f.stdin.Write([]byte("e")) },
					paint:   func(frame string) bool { return strings.Contains(frame, expected) },
				},
				"navigation": {
					prepare: func() { expected = resetConsole(FocusPanel(), 1) },
					trigger: func() { _, _ = f.stdin.Write([]byte("\x1b[B")) },
					paint:   func(frame string) bool { return strings.Contains(frame, "▸ "+expected) },
				},
				"render": {
					prepare: func() { expected = resetConsole(FocusPanel(), 0) },
					trigger: func() { f.host.SetSize(ptychild.Size{Rows: 41, Cols: 120}) },
					paint:   func(frame string) bool { return strings.Contains(frame, expected) },
				},
				"refresh": {
					prepare: func() {
						_ = resetConsole(FocusPanel(), 0)
						generation++
						expected = fmt.Sprintf("refresh-generation-%d", generation)
						refreshRows = append([]couchcore.ActionableThreadSummary(nil), rows...)
						refreshRows[0].Name = expected
						f.con.mu.Lock()
						f.con.refreshSchedule = RefreshSchedule{Sequence: generation, Running: generation}
						f.con.mu.Unlock()
					},
					trigger: func() { f.con.refreshResults <- menuRefreshResult{generation: generation, inventory: refreshRows} },
					paint:   func(frame string) bool { return strings.Contains(frame, expected) },
				},
				"feedback": {
					prepare: func() { expected = resetConsole(FocusPanel(), 0) },
					trigger: func() { _, _ = f.stdin.Write([]byte("\r")) },
					paint: func(frame string) bool {
						return strings.Contains(frame, expected) && strings.Contains(frame, "switch…")
					},
				},
			}
			limits := map[string]time.Duration{
				"open": 50 * time.Millisecond, "filter": 16 * time.Millisecond,
				"navigation": 16 * time.Millisecond, "render": 16 * time.Millisecond,
				"refresh": 16 * time.Millisecond, "feedback": 100 * time.Millisecond,
			}
			for name, path := range paths {
				stats := measureMenuLifecyclePath(t, f.host.Writes(), path, 20, 200)
				boundary := "semantic-input-to-correlated-emitted-frame"
				t.Logf("target=m2-max os=%s arch=%s trial=%s path=%s dimensions=120x40 warmups=20 samples=200 p50=%s p95=%s max=%s boundary=%s", runtime.GOOS, runtime.GOARCH, trial.name, name, stats.p50, stats.p95, stats.max, boundary)
				if stats.p95 > limits[name] {
					t.Errorf("%s p95 %s exceeds %s", name, stats.p95, limits[name])
				}
			}
		})
	}
}

type menuLifecyclePath struct {
	prepare func()
	trigger func()
	paint   func(string) bool
}

func measureMenuLifecyclePath(t *testing.T, writes <-chan []byte, path menuLifecyclePath, warmups, samples int) menuPerfStats {
	t.Helper()
	measure := func() time.Duration {
		path.prepare()
		started := time.Now()
		path.trigger()
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case raw := <-writes:
				if path.paint(string(raw)) {
					return time.Since(started)
				}
			case <-deadline.C:
				t.Fatal("timed out waiting for correlated menu frame")
				return 0
			}
		}
	}
	for i := 0; i < warmups; i++ {
		_ = measure()
	}
	durations := make([]time.Duration, samples)
	for i := range durations {
		durations[i] = measure()
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return menuPerfStats{
		p50: durations[(50*len(durations)+99)/100-1],
		p95: durations[(95*len(durations)+99)/100-1],
		max: durations[len(durations)-1],
	}
}

func measureMenuPath(path func(), warmups, samples int) menuPerfStats {
	for i := 0; i < warmups; i++ {
		path()
	}
	durations := make([]time.Duration, samples)
	for i := range durations {
		started := time.Now()
		path()
		durations[i] = time.Since(started)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return menuPerfStats{
		p50: durations[(50*len(durations)+99)/100-1],
		p95: durations[(95*len(durations)+99)/100-1],
		max: durations[len(durations)-1],
	}
}

func startMenuCPULoad(workers int) func() {
	stop := make(chan struct{})
	var joined sync.WaitGroup
	buffer := make([]byte, 1<<20)
	for i := range buffer {
		buffer[i] = byte(i)
	}
	joined.Add(workers)
	for i := 0; i < workers; i++ {
		go func(seed byte) {
			defer joined.Done()
			local := append([]byte(nil), buffer...)
			local[0] = seed
			for {
				select {
				case <-stop:
					return
				default:
					_ = sha256.Sum256(local)
				}
			}
		}(byte(i))
	}
	return func() {
		close(stop)
		joined.Wait()
	}
}
