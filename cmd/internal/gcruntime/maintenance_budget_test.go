package gcruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestMaintenancePreMigrationVisitsEveryOwnerWithinBatchLimit(t *testing.T) {
	s, original := scheduleFixture(t)
	owners := []artifactpath.StorageOwner{original}
	for n := 0; n < 6; n++ {
		owner, err := artifactpath.NewStorageOwner(s.Collector.Coordinator.Root, "", fmt.Sprintf("owner%02d", n))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(owner.Directory(), "draft-"+owner.Tag+".md"), []byte("retain"), 0600); err != nil {
			t.Fatal(err)
		}
		owners = append(owners, owner)
	}
	probes, persists := 0, 0
	s.Collector.Legacy = func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
		probes++
		return storagegc.ProcessDead, nil
	}
	s.Collector.Coordinator.BeforePersist = func() error { persists++; return nil }
	cursor := ""
	complete := false
	for page := 0; page < 4; page++ {
		before := cursor
		probes, persists = 0, 0
		var err error
		cursor, complete, err = s.Batch(context.Background(), cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		if probes > 4 || persists > 2 {
			t.Fatalf("page exceeded owner budget: probes=%d persists=%d", probes, persists)
		}
		initialized := 0
		for _, owner := range owners {
			if _, err := s.Collector.Coordinator.ReadOwner(owner); err == nil {
				initialized++
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(owner.Directory(), "draft-"+owner.Tag+".md")); err != nil {
				t.Fatal("pre-migration removed payload", err)
			}
		}
		want := min((page+1)*2, len(owners))
		if initialized != want {
			t.Fatalf("page%d initialized%d owners, want%d (limit2)", page, initialized, want)
		}
		if complete && initialized < len(owners) {
			t.Fatalf("completed after initializing only%d/%d owners; cursor discarded", initialized, len(owners))
		}
		if !complete && cursor == before {
			t.Fatal("owner cursor did not advance")
		}
	}
	if !complete {
		t.Fatal("pre-migration pass did not finish")
	}
}

func TestMaintenanceBusyRootReturnsWithoutWaitingForForeground(t *testing.T) {
	s, _ := scheduleFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- s.Collector.Coordinator.WithLock(context.Background(), func(*storagegc.Locked) error { close(entered); <-release; return nil })
	}()
	<-entered
	defer func() {
		close(release)
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	result := make(chan error, 1)
	go func() { _, _, err := s.Batch(context.Background(), "", 1); result <- err }()
	select {
	case err := <-result:
		if !errors.Is(err, storagegc.ErrCoordinatorBusy) {
			t.Fatalf("busy root result %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("maintenance waited on foreground root lock")
	}
}

func TestMaintenanceHundredThousandFilenamesBoundsOwnerWork(t *testing.T) {
	s, owner := scheduleFixture(t)
	for n := 0; n < 6; n++ {
		if err := os.WriteFile(filepath.Join(owner.Directory(), fmt.Sprintf("draft-owner%02d.md", n)), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	queue := filepath.Join(owner.Directory(), "queue-tag")
	if err := os.Mkdir(queue, 0700); err != nil {
		t.Fatal(err)
	}
	// Hard links reduce fixture allocation while retaining 100,000 real payload
	// filenames for production directory enumeration and classification. Spread
	// links across seeds to stay below filesystems' per-inode hard-link limits.
	source := filepath.Join(owner.Directory(), "draft-tag.md")
	for n := 0; n < 99993; n++ {
		seed := filepath.Join(owner.Directory(), fmt.Sprintf("draft-owner%02d.md", n%6))
		if err := os.Link(seed, filepath.Join(queue, fmt.Sprintf("%06d.md", n))); err != nil {
			t.Fatal(err)
		}
	}
	// Make the first selected owners otherwise eligible: incomplete inventory,
	// rather than missing migration acknowledgement or fresh grace, must retain them.
	c := s.Collector.Coordinator
	for n := 0; n < 2; n++ {
		candidate, err := artifactpath.NewStorageOwner(c.Root, "", fmt.Sprintf("owner%02d", n))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Initialize(context.Background(), candidate); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.CompleteMigration(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	old := c.Now()
	c.Now = func() time.Time { return old.Add(61 * 24 * time.Hour) }
	probes, persists := 0, 0
	s.Collector.Legacy = func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
		probes++
		return storagegc.ProcessDead, nil
	}
	s.Collector.Coordinator.BeforePersist = func() error { persists++; return nil }
	report, err := s.Collector.ApplyPage(context.Background(), "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.OwnersProcessed != 2 || report.BatchComplete || report.NextOwner == "" {
		t.Fatalf("missing bounded cursor %+v", report)
	}
	if probes > 4 || persists > 2 {
		t.Fatalf("unbounded owner effects: probes%d persists%d", probes, persists)
	}
	if report.Complete || report.Collected != 0 {
		t.Fatalf("100k cap authorized incomplete inventory: complete=%v collected=%d", report.Complete, report.Collected)
	}
	for n := 0; n < 2; n++ {
		if _, err := os.Stat(filepath.Join(owner.Directory(), fmt.Sprintf("draft-owner%02d.md", n))); err != nil {
			t.Fatal("incomplete inventory removed eligible payload", err)
		}
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("incomplete inventory removed payload", err)
	}
}

func TestMaintenanceExpiredContextDoesNoWork(t *testing.T) {
	s, owner := scheduleFixture(t)
	writes := 0
	s.Collector.Coordinator.BeforePersist = func() error { writes++; return nil }
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, _, err := s.Batch(ctx, "", 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired context %v", err)
	}
	if writes != 0 {
		t.Fatalf("expired worker persisted%d records", writes)
	}
	if _, err := s.Collector.Coordinator.ReadOwner(owner); !os.IsNotExist(err) {
		t.Fatal("expired worker initialized owner", err)
	}
}

func TestMaintenanceSchedulerDeadlineRetainsDurableCursor(t *testing.T) {
	s, _ := scheduleFixture(t)
	calls := 0
	scheduler := storagegc.Scheduler{Coordinator: s.Collector.Coordinator, WorkBudget: 10 * time.Millisecond, Limit: 2}
	scheduler.Batch = func(ctx context.Context, cursor string, limit int) (string, bool, error) {
		calls++
		if limit != 2 {
			t.Fatal("lost configured batch budget")
		}
		if calls == 1 {
			return "saved", false, nil
		}
		if cursor != "saved" {
			t.Fatalf("deadline replaced durable cursor with %q", cursor)
		}
		if calls == 2 {
			<-ctx.Done()
			return "uncommitted", false, ctx.Err()
		}
		return "", true, nil
	}
	for n := 0; n < 3; n++ {
		if ran, err := scheduler.RunOnce(context.Background()); err != nil || !ran {
			t.Fatalf("run%d ran%v %v", n, ran, err)
		}
	}
	if calls != 3 {
		t.Fatalf("worker failed to resume: calls%d", calls)
	}
}

func TestMaintenanceDiagnosticPagesDoNotEvaluateSessionOwners(t *testing.T) {
	s, _ := scheduleFixture(t)
	ctx := context.Background()
	c := s.Collector.Coordinator
	for n := 0; n < 7; n++ {
		if err := os.WriteFile(filepath.Join(c.Root, fmt.Sprintf("draft-extra%d.md", n)), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.CompleteMigration(ctx, nil); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s.DiagnosticOptions.Now = func() time.Time { return now }
	s.DiagnosticOptions.Proof = func(context.Context, string, []diagnosticlog.Registration) error { return nil }
	for _, tag := range []string{"tag", "extra0"} {
		p := filepath.Join(c.Root, "wrap-events-"+tag+".jsonl")
		if err := os.WriteFile(p, []byte("expired diagnostic"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, now.Add(-8*24*time.Hour), now.Add(-8*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	probes, persists := 0, 0
	c.BeforePersist = func() error { persists++; return nil }
	s.Collector.Legacy = func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
		probes++
		return storagegc.ProcessDead, nil
	}
	s.Collector.LegacyRoot = func(context.Context, []storagegc.ProcessIdentity) (storagegc.Liveness, error) {
		probes++
		return storagegc.ProcessDead, nil
	}
	cursor, complete := `{"phase":"diagnostics"}`, false
	for n := 0; !complete && n < 4; n++ {
		previous := cursor
		var err error
		cursor, complete, err = s.Batch(ctx, cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		if probes != 0 || persists != 0 {
			t.Fatalf("diagnostic page evaluated sessions: probes=%d persists=%d", probes, persists)
		}
		if !complete && cursor == previous {
			t.Fatal("diagnostic page failed to advance")
		}
	}
	if !complete {
		t.Fatal("diagnostic pass did not complete")
	}
	for _, tag := range []string{"tag", "extra0"} {
		if _, err := os.Stat(filepath.Join(c.Root, "wrap-events-"+tag+".jsonl")); !os.IsNotExist(err) {
			t.Fatalf("diagnostic remains: %s %v", tag, err)
		}
	}
}
