package couchtty

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
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
	now := time.Unix(1_800_000_000, 0)
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

			host := hostty.NewFakeHost(ptychild.Size{Rows: 41, Cols: 120})
			con := New(host, nil)
			defer con.Stop()
			child := ptychild.NewFakeChild(nil)
			con.attachThreadActor("perf-root", "perf-root", rows[0].Address, couchcore.Worktree(rows[0].WorkingPath), rows[0].Name, child)
			resetConsole := func() {
				con.mu.Lock()
				con.menu = NewMenuState(rows, rows[0].Address)
				con.menuReady = true
				con.focus = FocusPanel()
				con.size = ptychild.Size{Rows: 41, Cols: 120}
				con.refreshSchedule = RefreshSchedule{}
				con.mu.Unlock()
				host.Reset()
			}
			paths := map[string]func(){
				"open": func() {
					resetConsole()
					con.showMenu()
				},
				"filter": func() {
					resetConsole()
					con.onMenuInput([]byte("9"))
				},
				"navigation": func() {
					resetConsole()
					con.onMenuInput([]byte("\x1b[B"))
				},
				"render": func() {
					_ = RenderMenu(NewMenuState(rows, rows[0].Address), 120, 40, now, true)
				},
				"refresh": func() {
					resetConsole()
					con.mu.Lock()
					con.refreshSchedule = RefreshSchedule{Sequence: 1, Running: 1}
					con.mu.Unlock()
					con.finishMenuRefresh(menuRefreshResult{generation: 1, inventory: rows})
				},
				"feedback": func() {
					resetConsole()
					con.onMenuInput([]byte("\r"))
				},
			}
			limits := map[string]time.Duration{
				"open": 50 * time.Millisecond, "filter": 16 * time.Millisecond,
				"navigation": 16 * time.Millisecond, "render": 16 * time.Millisecond,
				"refresh": 16 * time.Millisecond, "feedback": 100 * time.Millisecond,
			}
			for name, path := range paths {
				stats := measureMenuPath(path, 20, 200)
				boundary := "console-input-to-repaint-return"
				if name == "render" {
					boundary = "render-call-to-ansi-return"
				} else if name == "refresh" {
					boundary = "refresh-result-to-repaint-return"
				}
				t.Logf("target=m2-max os=%s arch=%s trial=%s path=%s dimensions=120x40 warmups=20 samples=200 p50=%s p95=%s max=%s boundary=%s", runtime.GOOS, runtime.GOARCH, trial.name, name, stats.p50, stats.p95, stats.max, boundary)
				if stats.p95 > limits[name] {
					t.Errorf("%s p95 %s exceeds %s", name, stats.p95, limits[name])
				}
			}
		})
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
