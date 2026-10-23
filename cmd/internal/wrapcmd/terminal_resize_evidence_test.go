package wrapcmd

import (
	"testing"
	"time"
)

func TestTerminalResizeEvidenceGeneratedTransactions(t *testing.T) {
	m, err := newTerminalModel(80, 38)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if !m.SnapshotDiagnostic().LastResize.AttemptedAt.IsZero() {
		t.Fatal("fabricated resize before observation")
	}
	for step := range 128 {
		width, height := 80+step%2, 38-step%2
		before := m.SnapshotDiagnostic()
		begin := time.Now()
		r, err := m.PrepareResize(width, height)
		if err != nil {
			t.Fatal(err)
		}
		prepared := m.SnapshotDiagnostic()
		if prepared.LastResize.Outcome != "prepared" || !prepared.LastResize.CompletedAt.IsZero() || prepared.LastResize.AttemptedAt.Before(begin) || prepared.Snapshot.Width != width || prepared.Snapshot.Height != height {
			t.Fatalf("bad prepared observation: %+v", prepared.LastResize)
		}
		if step%3 == 0 {
			r.Abort()
		} else if err := r.Commit(); err != nil {
			t.Fatal(err)
		}
		resolved := m.SnapshotDiagnostic()
		if resolved.LastResize.CompletedAt.Before(resolved.LastResize.AttemptedAt) {
			t.Fatal("completion precedes attempt")
		}
		if step%3 == 0 {
			if resolved.LastResize.Outcome != "aborted" || resolved.Snapshot.Complete != prepared.Snapshot.Complete {
				t.Fatal("abort confirmed geometry")
			}
		} else if resolved.LastResize.Outcome != "committed" {
			t.Fatal("commit not recorded")
		}
		r.Abort()
		if err := r.Commit(); err != nil {
			t.Fatal(err)
		}
		if m.SnapshotDiagnostic().LastResize != resolved.LastResize {
			t.Fatal("duplicate resolution changed evidence")
		}
		if _, err := m.PrepareResize(0, 38); err == nil {
			t.Fatal("invalid size accepted")
		}
		if m.SnapshotDiagnostic().LastResize != resolved.LastResize {
			t.Fatal("rejected size replaced admitted observation")
		}
		if resolved.Generation < before.Generation {
			t.Fatal("generation went backwards")
		}
	}
	final := m.SnapshotDiagnostic().LastResize
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.PrepareResize(80, 38); err == nil {
		t.Fatal("closed model accepted resize")
	}
	if m.SnapshotDiagnostic().LastResize != final {
		t.Fatal("closed call replaced last observation")
	}
}

func TestTerminalDiagnosticSnapshotConcurrentResize(t *testing.T) {
	m, err := newTerminalModel(80, 38)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	done := make(chan error, 1)
	go func() {
		for step := range 128 {
			r, err := m.PrepareResize(80+step%2, 38-step%2)
			if err != nil {
				done <- err
				return
			}
			if err := m.Feed([]byte("\x1b[?25h")); err != nil {
				r.Abort()
				done <- err
				return
			}
			if step%3 == 0 {
				r.Abort()
			} else if err := r.Commit(); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	var previous uint64
	for range 1024 {
		e := m.SnapshotDiagnostic()
		if !e.LastResize.AttemptedAt.IsZero() && (e.Snapshot.Width != e.LastResize.Cols || e.Snapshot.Height != e.LastResize.Rows) {
			t.Error("mixed screen and resize observation")
		}
		if e.Generation < previous {
			t.Error("snapshot generation went backwards")
		}
		previous = e.Generation
		if len(e.Snapshot.Cells) != e.Snapshot.Width*e.Snapshot.Height {
			t.Error("mixed geometry and cells")
		}
		if !e.LastResize.CompletedAt.IsZero() && e.LastResize.CompletedAt.Before(e.LastResize.AttemptedAt) {
			t.Error("mixed resize times")
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestTerminalResizeRetryReportsSameGeometry(t *testing.T) {
	m, err := newTerminalModel(80, 38)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	r, err := m.PrepareResize(79, 37)
	if err != nil {
		t.Fatal(err)
	}
	r.Abort()
	if err := m.Resize(79, 37); err != nil {
		t.Fatal(err)
	}
	e := m.SnapshotDiagnostic()
	if e.LastResize.Changed || e.LastResize.Outcome != "committed" || e.Snapshot.CursorVisible || !e.Snapshot.Complete {
		t.Fatal("same geometry recovery changed diagnostic meaning or granted cursor")
	}
}
