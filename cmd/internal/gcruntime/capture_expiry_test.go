package gcruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// Classification itself is exercised against OSInspection in diagnosticlog's
// process_inventory_test.go. This stateful public Inspection seam exercises the
// resulting proof through actual runtime maintenance and capture collection.
type captureExpiryInspection struct {
	runtimes []diagnosticlog.Runtime
	err      error
	calls    int
}

type captureExpiryProcessProbe struct{ process storagegc.ProcessIdentity }

func (p captureExpiryProcessProbe) Inspect(process storagegc.ProcessIdentity) storagegc.Liveness {
	if process == p.process {
		return storagegc.ProcessAlive
	}
	return storagegc.ProcessDead
}

func (p *captureExpiryInspection) OpenFiles(ctx context.Context, _ string) ([]diagnosticlog.Registration, error) {
	return nil, ctx.Err()
}

func (p *captureExpiryInspection) Runtimes(ctx context.Context) ([]diagnosticlog.Runtime, error) {
	p.calls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.runtimes, p.err
}

func TestRuntimeMaintenanceCollectsExpiredCaptureWithRemovedProtocolRoot(t *testing.T) {
	for _, fence := range []string{"none", "legacy-root", "unknown-proof", "reader", "handoff", "young"} {
		t.Run(fence, func(t *testing.T) {
			ctx := context.Background()
			removedRoot := filepath.Join(t.TempDir(), "removed")
			if err := os.Mkdir(removedRoot, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(removedRoot); err != nil {
				t.Fatal(err)
			}
			inspection := &captureExpiryInspection{runtimes: []diagnosticlog.Runtime{{Process: diagnosticlog.Registration{PID: 123456789, Birth: "zellij-generation", Root: removedRoot}, Protocol: true}}}
			s, err := NewWithInspection(t.TempDir(), inspection)
			if err != nil {
				t.Fatal(err)
			}
			c := s.Collector.Coordinator
			now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
			c.Now = func() time.Time { return now }
			owner, err := artifactpath.NewStorageOwner(c.Root, "", "capture")
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Initialize(ctx, owner); err != nil {
				t.Fatal(err)
			}
			if err := c.CompleteMigration(ctx, nil); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
			if err != nil {
				t.Fatal(err)
			}
			at := now.Add(-storagegc.CaptureRetentionPeriod - time.Hour)
			if fence == "young" {
				at = now.Add(-storagegc.CaptureRetentionPeriod + time.Nanosecond)
			}
			capture, err := paths.ParkedScrollbackArtifacts(at.Format("20060102T150405"))
			if err != nil {
				t.Fatal(err)
			}
			for path, body := range map[string]string{capture.Raw: "previous objective\r\n", capture.Events: "{\"type\":\"resize\",\"offset\":0,\"cols\":40,\"rows\":2}\n"} {
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.PublishCaptureMetadata(ctx, owner, artifactpath.ParkedCapture{Raw: capture.Raw, Events: capture.Events, Metadata: capture.Metadata}, at); err != nil {
				t.Fatal(err)
			}
			members := []string{capture.Raw, capture.Events, capture.Metadata}
			original := map[string][]byte{}
			for _, path := range members {
				original[path], err = os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
			}
			process := storagegc.ProcessIdentity{PID: 42, Birth: "capture-reader"}
			c.Probe = captureExpiryProcessProbe{process}
			var protection string
			switch fence {
			case "legacy-root":
				inspection.runtimes[0].Protocol = false
				// OSInspection refuses a legacy runtime's missing root. Preserve
				// that error instead of treating an empty inventory as proof.
				inspection.err = diagnosticlog.ErrUnknownWriters
			case "unknown-proof":
				inspection.err = errors.New("process environment unavailable")
			case "reader":
				protection, err = c.AcquireRoleProcessTarget(ctx, owner, process, "retained-peek-1", "", capture.Raw)
			case "handoff":
				protection, err = c.BeginUse(ctx, owner, process, capture.Raw)
			}
			if err != nil {
				t.Fatal(err)
			}
			preview, err := s.Preview(ctx)
			if err != nil {
				t.Fatal(err)
			}
			wantState := storagegc.Eligible
			switch fence {
			case "legacy-root", "unknown-proof", "handoff":
				wantState = storagegc.Blocked
			case "reader":
				wantState = storagegc.Live
			case "young":
				wantState = storagegc.Grace
			}
			found := false
			for _, item := range preview.Storage.Items {
				if item.Bucket == artifactpath.CaptureRetention {
					found = true
					if item.Decision.State != wantState || len(item.Members) != len(members) {
						t.Fatalf("capture generation decision = %+v; want %s with all three members", item, wantState)
					}
				}
			}
			if !found {
				t.Fatal("capture generation absent from retention inventory")
			}
			// The sessions phase also owns independently aged parked captures.
			// Running the public bounded batch exercises the automatic path.
			if _, _, err := s.Batch(ctx, "", 10); err != nil {
				t.Fatal(err)
			}
			if inspection.calls == 0 {
				t.Fatal("maintenance bypassed runtime proof")
			}
			assertMembers := func(retained bool) {
				t.Helper()
				for _, path := range members {
					body, err := os.ReadFile(path)
					if retained {
						if err != nil || string(body) != string(original[path]) {
							t.Fatalf("protected recovery member changed %s: %q, %v", path, body, err)
						}
					} else if !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("expired recovery member retained %s: %v", path, err)
					}
				}
			}
			assertMembers(fence != "none")
			if fence == "reader" || fence == "handoff" {
				if fence == "reader" {
					err = c.ReleaseProcess(ctx, owner, protection)
				} else {
					err = c.CancelUnchangedUse(ctx, owner, protection)
				}
				if err != nil {
					t.Fatal(err)
				}
				applied, err := s.Apply(ctx, 10)
				if err != nil || applied.Storage.Collected != 1 {
					t.Fatalf("released capture not collected: %+v, %v", applied.Storage, err)
				}
				assertMembers(false)
			}
		})
	}
}
