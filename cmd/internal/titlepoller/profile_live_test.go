package titlepoller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
)

// TestTitleProfileLive measures the real title projection against copied Pair
// identity/catalog files and real, read-only native transcripts. It deliberately
// bypasses Run/RunCLI: no live lease, PID file, title, or process is changed.
//
// Opt in with PAIR_TITLE_PROFILE_LIVE=1, an absolute
// PAIR_TITLE_PROFILE_OUT and PAIR_TITLE_PROFILE_SOURCE, and explicit
// PAIR_TITLE_PROFILE_SCOPE/TAG/AGENT/ORDINAL. PAIR_TITLE_PROFILE_ITERATIONS defaults to 10.
// PAIR_TITLE_PROFILE_PROJECTION=1 selects the explicitly cached comparison path.
// Profiles and stats are private local evidence; do not publish their contents.
func TestTitleProfileLive(t *testing.T) {
	if os.Getenv("PAIR_TITLE_PROFILE_LIVE") != "1" {
		t.Skip("set PAIR_TITLE_PROFILE_LIVE=1 for controlled production-path profiling")
	}
	required := func(key string) string {
		t.Helper()
		value := os.Getenv(key)
		if value == "" {
			t.Fatalf("%s is required", key)
		}
		return value
	}
	out, source := required("PAIR_TITLE_PROFILE_OUT"), required("PAIR_TITLE_PROFILE_SOURCE")
	if !filepath.IsAbs(out) || !filepath.IsAbs(source) {
		t.Fatal("profile output and source must be absolute directories")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	scope, tag, agent := required("PAIR_TITLE_PROFILE_SCOPE"), required("PAIR_TITLE_PROFILE_TAG"), required("PAIR_TITLE_PROFILE_AGENT")
	ordinal, err := strconv.ParseUint(required("PAIR_TITLE_PROFILE_ORDINAL"), 10, 64)
	if err != nil || ordinal == 0 {
		t.Fatal("profile requires exact nonzero invocation ordinal")
	}
	iterations := 10
	if raw := os.Getenv("PAIR_TITLE_PROFILE_ITERATIONS"); raw != "" {
		var err error
		iterations, err = strconv.Atoi(raw)
		if err != nil || iterations < 1 || iterations > 100 {
			t.Fatal("profile iterations must be between 1 and 100")
		}
	}
	sourcePaths, err := artifactpath.ResolveScoped(source, tag)
	if err != nil {
		t.Fatal(err)
	}
	replica := t.TempDir()
	type sourceStamp struct {
		Name    string
		Size    int64
		ModTime time.Time
		SHA256  string
	}
	var stamps []sourceStamp
	for _, path := range []string{sourcePaths.Ledger(), sourcePaths.SessionInventoryCatalog()} {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("source must be a regular file: %s: %v", path, err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(raw)
		stamps = append(stamps, sourceStamp{filepath.Base(path), info.Size(), info.ModTime(), hex.EncodeToString(digest[:])})
		if err := os.WriteFile(filepath.Join(replica, filepath.Base(path)), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Scope remains the original exact owner; only Pair-owned storage redirects.
	t.Setenv("PAIR_DATA_DIR", replica)
	t.Setenv("PAIR_TAG", tag)
	t.Setenv("PAIR_SCOPE_KEY", scope)
	t.Setenv("PAIR_LAUNCH_ORDINAL", strconv.FormatUint(ordinal, 10))
	rt := NewOSRuntime()
	activity, count := rt.SessionActivity, rt.ContextCount
	projectionErr := rt.Err
	projectionMode := os.Getenv("PAIR_TITLE_PROFILE_PROJECTION") == "1"
	if projectionMode {
		p := newTitleProjection(contextcmd.EnvFromOS(), ordinal)
		projectionErr = p.Err
		activity = func(tag, agent string) (time.Time, bool) { return p.activity(tag, agent) }
		count = func(tag, agent string) string { return p.count(tag, agent) }
	}
	privateFile := func(name string) *os.File {
		t.Helper()
		f, err := os.OpenFile(filepath.Join(out, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	profile := func(name, kind string) {
		t.Helper()
		f := privateFile(name)
		err := pprof.Lookup(kind).WriteTo(f, 0)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("profile %s: %v; close: %v", name, err, closeErr)
		}
	}
	type sample struct {
		Phase                                                            string
		Iteration                                                        int
		Elapsed                                                          time.Duration
		ActivityFound                                                    bool
		ContextCount                                                     string
		HeapAlloc, HeapInuse, HeapSys, HeapReleased, TotalAlloc, Mallocs uint64
		NumGC                                                            uint32
	}
	var samples []sample
	record := func(phase string, iteration int, elapsed time.Duration, found bool, count string) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		samples = append(samples, sample{phase, iteration, elapsed, found, count, m.HeapAlloc, m.HeapInuse, m.HeapSys, m.HeapReleased, m.TotalAlloc, m.Mallocs, m.NumGC})
	}
	runtime.GC()
	record("baseline", 0, 0, false, "")
	profile("heap-before.pprof", "heap")
	cpu := privateFile("cpu.pprof")
	if err := pprof.StartCPUProfile(cpu); err != nil {
		_ = cpu.Close()
		t.Fatal(err)
	}
	for i := 1; i <= iterations; i++ {
		started := time.Now()
		_, found := activity(tag, agent)
		occupancy := count(tag, agent)
		if err := projectionErr(); err != nil || !found {
			pprof.StopCPUProfile()
			_ = cpu.Close()
			t.Fatalf("profile iteration %d did not produce admitted activity: found=%v projection-error=%v", i, found, err)
		}
		if (agent == "claude" || agent == "codex" || agent == "grok") && occupancy == "" {
			pprof.StopCPUProfile()
			_ = cpu.Close()
			t.Fatalf("profile iteration %d has no supported usage positive control", i)
		}
		record("iteration", i, time.Since(started), found, occupancy)
	}
	pprof.StopCPUProfile()
	if err := cpu.Close(); err != nil {
		t.Fatal(err)
	}
	record("natural", iterations, 0, false, "")
	profile("heap-natural.pprof", "heap")
	profile("allocs.pprof", "allocs")
	runtime.GC()
	record("post-gc", iterations, 0, false, "")
	profile("heap-postgc.pprof", "heap")
	runtime.KeepAlive(rt)
	runtime.KeepAlive(activity)
	runtime.KeepAlive(count)
	evidence := privateFile("stats.json")
	err = json.NewEncoder(evidence).Encode(struct {
		GoVersion               string
		PID                     int
		ProjectionComparison    bool
		Scope, Tag, Agent, Home string
		Sources                 []sourceStamp
		Samples                 []sample
	}{runtime.Version(), os.Getpid(), projectionMode, scope, tag, agent, os.Getenv("HOME"), stamps, samples})
	closeErr := evidence.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("stats: %v; close: %v", err, closeErr)
	}
	for _, s := range samples {
		t.Logf("%s iteration=%d elapsed=%s found=%v count=%s HeapAlloc=%d HeapInuse=%d HeapSys=%d HeapReleased=%d TotalAlloc=%d NumGC=%d", s.Phase, s.Iteration, s.Elapsed, s.ActivityFound, s.ContextCount, s.HeapAlloc, s.HeapInuse, s.HeapSys, s.HeapReleased, s.TotalAlloc, s.NumGC)
	}
	t.Logf("controlled replica profiles: %s", out)
}
