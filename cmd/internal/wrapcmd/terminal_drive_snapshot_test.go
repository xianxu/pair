package wrapcmd

import (
	"strings"
	"testing"
)

func TestAgentDriveScreenUsesTerminalFeedGeneration(t *testing.T) {
	f, _, _ := driveFixture(t)
	before := f.proxy.agentDriveScreen()
	// Feed directly while handleChunk has not advanced its deferred proxy epoch.
	if err := f.proxy.terminal.Feed([]byte("\x1b[?2026h\x1b[2J\x1b[Hnew paint\x1b[?2026l")); err != nil {
		t.Fatal(err)
	}
	after := f.proxy.agentDriveScreen()
	if after.Epoch <= before.Epoch {
		t.Fatalf("terminal repaint kept stale drive epoch: before=%d after=%d", before.Epoch, after.Epoch)
	}
}
func TestTerminalDriveSnapshotCoherentAcrossConcurrentPaints(t *testing.T) {
	model, err := newTerminalModel(20, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	done := make(chan error, 1)
	ready := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			if err := model.Feed([]byte("\x1b[?2026h\x1b[2J\x1b[HA\x1b[?2026l")); err != nil {
				done <- err
				return
			}
			if i == 0 {
				close(ready)
			}
			if err := model.Feed([]byte("\x1b[?2026h\x1b[2J\x1b[HB")); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	<-ready
	for i := 0; i < 3000; i++ {
		evidence := model.DriveSnapshot(3)
		if evidence.Generation == 0 {
			continue
		}
		want := "B"
		complete := false
		if evidence.Generation%2 == 1 {
			want = "A"
			complete = true
		}
		cell := evidence.Snapshot.CellAt(0, 0)
		if cell == nil || cell.Content != want || evidence.Complete != complete || len(evidence.Tail.Lines) == 0 || !strings.Contains(evidence.Tail.Lines[0], want) {
			t.Fatalf("mixed generation %d: complete=%v cell=%#v tail=%q", evidence.Generation, evidence.Complete, cell, evidence.Tail.Lines)
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestTerminalDriveSnapshotClosesAndOwnsCells(t *testing.T) {
	model, err := newTerminalModel(20, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err = model.Feed([]byte("\x1b[?2026h\x1b[2J\x1b[HA\x1b[?2026l")); err != nil {
		t.Fatal(err)
	}
	evidence := model.DriveSnapshot(3)
	evidence.Snapshot.Cells[0].Content = "changed"
	if model.DriveSnapshot(3).Snapshot.CellAt(0, 0).Content != "A" {
		t.Fatal("drive snapshot aliases terminal cells")
	}
	if err = model.Close(); err != nil {
		t.Fatal(err)
	}
	closed := model.DriveSnapshot(3)
	if closed.Complete || closed.Generation <= evidence.Generation {
		t.Fatalf("closed evidence %+v", closed)
	}
}

func TestTerminalDriveSnapshotResizeInvalidatesEvidence(t *testing.T) {
	model, err := newTerminalModel(20, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	if err = model.Feed([]byte("\x1b[?2026h\x1b[2J\x1b[HA\x1b[?2026l")); err != nil {
		t.Fatal(err)
	}
	before := model.DriveSnapshot(3)
	resize, err := model.PrepareResize(30, 3)
	if err != nil {
		t.Fatal(err)
	}
	during := model.DriveSnapshot(3)
	if during.Complete || during.Generation <= before.Generation || during.Snapshot.Width != 30 {
		t.Fatalf("resize retained valid old evidence: before=%+v during=%+v", before, during)
	}
	if err = resize.Commit(); err != nil {
		t.Fatal(err)
	}
	after := model.DriveSnapshot(3)
	if after.Generation <= during.Generation {
		t.Fatal("resize commit did not invalidate held evidence")
	}
}
