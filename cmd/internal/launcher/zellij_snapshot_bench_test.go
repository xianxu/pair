package launcher

import (
	"testing"
	"time"
)

// BenchmarkZellijSnapshotLive measures what a Couch inventory refresh -- and,
// since pair#170 M3, a `couch` STARTUP with any detach candidate -- actually
// pays to observe zellij.
//
// The cost is two `list-sessions` runs plus one `action list-clients` per
// non-exited session on the host, so it scales with the operator's whole
// session set rather than with the threads couch cares about. Opt-in because it
// spawns real subprocesses and its number depends on live host state.
func BenchmarkZellijSnapshotLive(b *testing.B) {
	if testing.Short() {
		b.Skip("spawns real zellij subprocesses")
	}
	sessions, err := (ZellijSource{}).Snapshot()
	if err != nil {
		b.Skipf("zellij unavailable: %v", err)
	}
	b.ReportMetric(float64(len(sessions)), "sessions")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, err := (ZellijSource{}).Snapshot(); err != nil {
			b.Fatal(err)
		}
		_ = time.Since(start)
	}
}
