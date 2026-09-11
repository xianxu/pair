// Measurement probe for pair#206: what does reattaching a detached zellij
// session cost, one at a time versus all at once, and what does it do to
// `zellij action` latency while it runs?
//
// #206 reattaches every detached thread at couch startup. The question the
// issue refuses to answer by analogy is the server-side cost of a `zellij
// attach` -- each one drives a restore and a full render -- and whether a burst
// of them makes the workbench unusable (workshop/targets/workbench-latency.md).
//
// Method: create N detached sessions with a pair-shaped layout (a framed agent
// pane over a fixed 12-row borderless draft) under the repo's own zellij config,
// at a realistic terminal size. Each agent pane prints a screenful and then a
// per-session marker. Then time real `zellij attach` clients until the marker
// renders, in three phases: one alone, N one after another, N at once. A sampler
// times `zellij action query-tab-names` against a session nobody attaches, for
// the whole run, so each phase has its own latency distribution, including a
// quiet baseline.
//
//	make test-reattach-cost            # N=8
//	PAIR_PROBE_N=11 make test-reattach-cost
//
// It also times couch's zellij work around each attach, as two couch-shaped
// passes built from production's own calls: the critical path as it ran before
// pair#228 (five full snapshots and a name probe), and after it (two targeted
// snapshots and three liveness ones). The count those calls make is proved by
// the stub tests; this is the timing that goes with it. It does not time the
// process spawns (pair-launch-helper, pair) or the registration poll interval.
//
// A timing is meaningless without its co-tenancy, so the report leads with it:
// the load average, and how many agent processes were running.
//
// Gotchas inherited from the other zellij probes: ZELLIJ* must be scrubbed
// (zellij refuses to nest), and every exit returns through run() so the
// sessions this probe made are always deleted (TestNoProbeExitsPastItsOwnCleanup).
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/probes/zellijprobe"
)

const (
	rows, cols    = 50, 200
	attachTimeout = 20 * time.Second

	// The couch reattach's critical path, site by site (pair#228's table):
	//   1-2 couchcore's detached proof, and its re-proof before the child effect
	//   3   couchcore's registration poll (PairSession)
	//   4-5 `pair resume`'s launcher: the orphan-nvim sweep, and runOnce
	//   6   the launcher's session-name acceptance probe (one list-clients)
	// The inventory refresh on completion (site 7) is async and off the path,
	// so neither pattern includes it.
	//
	// oldFullSnapshots is sites 1-5 as they ran before pair#228: each a full
	// snapshot, two list-sessions plus one list-clients per live pair session.
	oldFullSnapshots = 5
)

func main() { os.Exit(run()) }

type sample struct {
	at  time.Time
	dur time.Duration
	ok  bool
}

type window struct {
	name       string
	start, end time.Time
}

func run() int {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("PROBE-ERROR: cannot locate probe assets")
		return 1
	}
	repo := filepath.Join(filepath.Dir(self), "..", "..", "..")
	config := filepath.Join(repo, "zellij", "config.kdl")

	n := 8
	if raw := os.Getenv("PAIR_PROBE_N"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 2 {
			fmt.Println("PROBE-ERROR PAIR_PROBE_N: want an integer >= 2")
			return 1
		}
		n = v
	}

	env := zellijprobe.Scrub([]string{"ZELLIJ"}, "TERM=xterm-256color")
	layout, err := writeLayout()
	if err != nil {
		fmt.Println("PROBE-ERROR layout:", err)
		return 1
	}
	defer os.RemoveAll(filepath.Dir(layout))

	// Session 0 is the latency CONTROL: never attached, only queried. Sessions
	// 1..n are the ones reattached. Every name carries this pid, so cleanup can
	// only ever delete what this run created.
	names := make([]string, n+1)
	for i := range names {
		// `pair-` prefixed: every snapshot filters to pair session names
		// (isPairSessionName), so a probe session named otherwise would be
		// invisible to the very calls being measured. They show in `pair list`
		// while the probe runs.
		names[i] = fmt.Sprintf("pair-rc%d-%d", os.Getpid(), i)
	}
	defer func() {
		for _, name := range names {
			_ = exec.Command("zellij", "delete-session", name, "--force").Run()
		}
	}()

	fmt.Println(cotenancy())
	// Every zellij call here is bounded. An unbounded one hung this probe for
	// ten minutes on its first N=8 run: the eighth `attach --create-background`
	// never returned. A hang is a precondition failure, not a measurement, so it
	// becomes PROBE-INCONCLUSIVE rather than an indefinite wait.
	for _, name := range names {
		created := false
		for attempt := 0; attempt < 2 && !created; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			cmd := exec.CommandContext(ctx, "zellij", "--config", config, "--layout", layout, "attach", "--create-background", name)
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			cancel()
			if err == nil {
				created = true
				break
			}
			fmt.Printf("  create %s attempt %d failed: %v %s\n", name, attempt+1, err, strings.TrimSpace(string(out)))
			_ = exec.Command("zellij", "delete-session", name, "--force").Run()
		}
		if !created {
			fmt.Printf("PROBE-INCONCLUSIVE: could not create session %s; nothing was measured.\n", name)
			return 2
		}
		time.Sleep(300 * time.Millisecond)
	}
	if !allListed(names, 15*time.Second) {
		fmt.Println("PROBE-INCONCLUSIVE: the probe's sessions never all appeared; nothing was measured.")
		return 2
	}
	time.Sleep(2 * time.Second) // let every agent pane print its screen and marker

	var mu sync.Mutex
	var samples []sample
	stop := make(chan struct{})
	var sampler sync.WaitGroup
	sampler.Add(1)
	go func() {
		defer sampler.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			began := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			cmd := exec.CommandContext(ctx, "zellij", "--session", names[0], "action", "query-tab-names")
			cmd.Env = env
			err := cmd.Run()
			cancel()
			mu.Lock()
			samples = append(samples, sample{at: began, dur: time.Since(began), ok: err == nil})
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
	}()

	var windows []window
	phase := func(name string, body func() bool) bool {
		w := window{name: name, start: time.Now()}
		okay := body()
		w.end = time.Now()
		windows = append(windows, w)
		return okay
	}
	type result struct {
		name string
		dur  time.Duration
		ok   bool
	}
	var results = map[string][]result{}

	phase("quiet baseline", func() bool { time.Sleep(3 * time.Second); return true })
	phase("one attach, x3", func() bool {
		for i := 0; i < 3; i++ {
			d, ok := attach(config, names[1], env)
			results["one attach, x3"] = append(results["one attach, x3"], result{names[1], d, ok})
			time.Sleep(500 * time.Millisecond)
		}
		return true
	})
	phase(fmt.Sprintf("%d sequential", n), func() bool {
		for _, name := range names[1:] {
			d, ok := attach(config, name, env)
			results[fmt.Sprintf("%d sequential", n)] = append(results[fmt.Sprintf("%d sequential", n)], result{name, d, ok})
		}
		return true
	})
	// The couch-shaped passes: each reattach preceded by the zellij work couch's
	// real critical path does, through PRODUCTION's own calls against this host's
	// real session list -- first as it ran before pair#228, then after. One run,
	// so both see the same co-tenancy. The column includes that work.
	bounded := func(f func(ctx context.Context)) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		f(ctx)
	}
	oldShaped := fmt.Sprintf("%d old pattern", n)
	phase(oldShaped, func() bool {
		for _, name := range names[1:] {
			began := time.Now()
			for k := 0; k < oldFullSnapshots; k++ { // sites 1-5
				bounded(func(ctx context.Context) { _, _ = (launcher.ZellijSource{}).SnapshotContext(ctx) })
			}
			_ = launcher.OSRuntime{}.ProbeSessionName(name) // site 6: one list-clients
			_, ok := attach(config, name, env)
			results[oldShaped] = append(results[oldShaped], result{name, time.Since(began), ok})
		}
		return true
	})
	time.Sleep(time.Second)
	newShaped := fmt.Sprintf("%d new pattern", n)
	phase(newShaped, func() bool {
		for _, name := range names[1:] {
			began := time.Now()
			for k := 0; k < 2; k++ { // sites 1-2: the proof asks only this session
				bounded(func(ctx context.Context) {
					_, _ = (launcher.ZellijSource{}).SnapshotSessionsContext(ctx, []string{name})
				})
			}
			for k := 0; k < 3; k++ { // sites 3-5: liveness only
				bounded(func(ctx context.Context) { _, _ = (launcher.ZellijSource{}).LivenessContext(ctx) })
			}
			// site 6: skipped -- a live session under the name proves acceptance.
			_, ok := attach(config, name, env)
			results[newShaped] = append(results[newShaped], result{name, time.Since(began), ok})
		}
		return true
	})
	time.Sleep(time.Second)
	phase(fmt.Sprintf("%d concurrent", n), func() bool {
		var wg sync.WaitGroup
		var rmu sync.Mutex
		for _, name := range names[1:] {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				d, ok := attach(config, name, env)
				rmu.Lock()
				results[fmt.Sprintf("%d concurrent", n)] = append(results[fmt.Sprintf("%d concurrent", n)], result{name, d, ok})
				rmu.Unlock()
			}(name)
		}
		wg.Wait()
		return true
	})
	phase("quiet after", func() bool { time.Sleep(2 * time.Second); return true })
	close(stop)
	sampler.Wait()

	failed := 0
	fmt.Printf("\n%-18s %8s %8s %8s %8s   %s\n", "phase", "wall", "attach", "attach", "attach", "zellij action query-tab-names (control session)")
	fmt.Printf("%-18s %8s %8s %8s %8s   %s\n", "", "", "min", "median", "max", "")
	for _, w := range windows {
		var ds []time.Duration
		for _, r := range results[w.name] {
			if !r.ok {
				failed++
				fmt.Printf("  %s: attach to %s never rendered its marker within %v\n", w.name, r.name, attachTimeout)
				continue
			}
			ds = append(ds, r.dur)
		}
		mu.Lock()
		var lat []time.Duration
		errs := 0
		for _, s := range samples {
			if s.at.Before(w.start) || s.at.After(w.end) {
				continue
			}
			if !s.ok {
				errs++
			}
			lat = append(lat, s.dur)
		}
		mu.Unlock()
		attachCols := "       -        -        -"
		if len(ds) > 0 {
			attachCols = fmt.Sprintf("%8s %8s %8s", ms(minOf(ds)), ms(pct(ds, 50)), ms(maxOf(ds)))
		}
		latency := "no samples"
		if len(lat) > 0 {
			latency = fmt.Sprintf("n=%d p50=%s p95=%s max=%s errors=%d", len(lat), ms(pct(lat, 50)), ms(pct(lat, 95)), ms(maxOf(lat)), errs)
		}
		fmt.Printf("%-18s %8s %s   %s\n", w.name, ms(w.end.Sub(w.start)), attachCols, latency)
	}
	fmt.Printf("\nattach = spawn of `zellij attach` until the session's own marker renders (%dx%d, pair-shaped layout, repo config).\n", cols, rows)
	// S = live pair sessions the full snapshots asked; the probe's own n+1 are
	// counted separately because they are synthetic.
	hostLive := 0
	if sessions, err := (launcher.ZellijSource{}).LivenessContext(context.Background()); err == nil {
		for _, s := range sessions {
			if s.State != launcher.SessionExited && !strings.HasPrefix(s.Name, fmt.Sprintf("pair-rc%d-", os.Getpid())) {
				hostLive++
			}
		}
	}
	s := hostLive + n + 1
	fmt.Printf("old pattern = %d full snapshots + 1 name probe + the attach: %d x (2 + S) + 1 list-clients per thread; S = %d (host %d + probe %d), so %d zellij calls before each attach.\n",
		oldFullSnapshots, oldFullSnapshots, s, hostLive, n+1, oldFullSnapshots*(2+s)+1)
	fmt.Printf("new pattern = 2 targeted snapshots + 3 liveness + the attach: 2 x (2 + 1) + 3 x 2 = 12 zellij calls before each attach, whatever S is.\n")
	fmt.Println("caveat: synthetic sessions answer list-clients in ~53ms where a real detached pair session takes ~250ms, so the OLD column understates the real before-cost; the count is the promise.")
	if failed > 0 {
		fmt.Printf("PROBE-INCONCLUSIVE: %d attach(es) never rendered their marker; the phases that include them are not measurements.\n", failed)
		return 2
	}
	return 0
}

// attach spawns a real client and times it until this session's marker renders,
// then detaches the way couch does: close the pty, SIGTERM the client.
func attach(config, name string, env []string) (time.Duration, bool) {
	cmd := exec.Command("zellij", "--config", config, "attach", name)
	cmd.Env = env
	began := time.Now()
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return 0, false
	}
	defer func() {
		_ = f.Close()
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_, _ = cmd.Process.Wait()
	}()
	marker := []byte("MARKER-" + name[strings.LastIndex(name, "-")+1:])
	seen := make([]byte, 0, 1<<16)
	found := make(chan time.Duration, 1)
	go func() {
		buf := make([]byte, 16384)
		for {
			k, rerr := f.Read(buf)
			if k > 0 {
				seen = append(seen, buf[:k]...)
				if containsBytes(seen, marker) {
					found <- time.Since(began)
					return
				}
			}
			if rerr != nil {
				return
			}
		}
	}()
	select {
	case d := <-found:
		return d, true
	case <-time.After(attachTimeout):
		return attachTimeout, false
	}
}

// writeLayout is pair's workbench shape: a framed agent pane that has printed a
// full screen (the restore zellij must render) over a fixed 12-row borderless
// draft. The marker is the session's own index, so an attach can only be
// satisfied by ITS session's render.
//
// The screen is generated by a script beside the layout rather than inline:
// KDL strings do not take shell escapes (`\033` is not a KDL escape), so an
// inline SGR sequence would make zellij reject the layout.
func writeLayout() (string, error) {
	dir, err := os.MkdirTemp("", "reattachcost-")
	if err != nil {
		return "", err
	}
	script := filepath.Join(dir, "agent.sh")
	agent := `#!/bin/sh
i=0
while [ $i -lt 120 ]; do
  printf '\033[3%dm%04d\033[0m the agent screen: a realistic line of prose that fills most of a wide terminal, repeated to make a real restore ....\n' $((i % 7 + 1)) "$i"
  i=$((i+1))
done
printf 'MARKER-%s\n' "${ZELLIJ_SESSION_NAME##*-}"
exec sleep 86400
`
	if err := os.WriteFile(script, []byte(agent), 0o755); err != nil {
		return "", err
	}
	layout := filepath.Join(dir, "layout.kdl")
	body := `layout {
    pane split_direction="horizontal" {
        pane name="agent" command="sh" {
            args "` + script + `"
        }
        pane size=12 name="draft" borderless=true command="sh" {
            args "-c" "echo draft pane; exec sleep 86400"
        }
    }
}
`
	return layout, os.WriteFile(layout, []byte(body), 0o644)
}

func allListed(names []string, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		out, _ := exec.Command("zellij", "list-sessions", "--short").Output()
		listed := map[string]bool{}
		for _, line := range strings.Split(string(out), "\n") {
			listed[strings.TrimSpace(line)] = true
		}
		all := true
		for _, name := range names {
			all = all && listed[name]
		}
		if all {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// cotenancy is the envelope every timing here depends on (workbench-latency):
// the load, and how many agents were running.
func cotenancy() string {
	load, _ := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	agents := 0
	for _, name := range []string{"claude", "codex"} {
		out, _ := exec.Command("pgrep", "-x", name).Output()
		agents += len(strings.Fields(string(out)))
	}
	sessions, _ := exec.Command("zellij", "list-sessions", "--short").Output()
	return fmt.Sprintf("co-tenancy: load %s | agent processes (claude+codex): %d | zellij sessions listed: %d | cores: %d",
		strings.TrimSpace(string(load)), agents, len(strings.Fields(string(sessions))), runtime.NumCPU())
}

func containsBytes(haystack, needle []byte) bool {
	return strings.Contains(string(haystack), string(needle))
}

func ms(d time.Duration) string { return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond)) }

func pct(ds []time.Duration, p int) time.Duration {
	sorted := append([]time.Duration(nil), ds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := (len(sorted)*p + 99) / 100
	if idx < 1 {
		idx = 1
	}
	return sorted[idx-1]
}

func minOf(ds []time.Duration) time.Duration { return pct(ds, 0) }
func maxOf(ds []time.Duration) time.Duration { return pct(ds, 100) }
