package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestReapProvenCLIExactOutcomesAndNoLegacyFallback(t *testing.T) {
	for _, status := range []string{"sent", "refused", "lost", "unsupported", "wrong-target"} {
		t.Run(status, func(t *testing.T) {
			target := *effectTarget()
			view := couchcore.ResourceView{Orphans: []couchcore.ResourceProcessView{{ResourceProvenance: couchcore.ResourceProvenance{ResourceRoot: couchcore.ResourceRoot{Key: target.Key, Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace}, Root: target.Root, Orphan: true, OrphanKnown: true}}}, LeftRunning: []couchcore.LeftRunningOrphan{{Process: couchcore.ProcessIdentity{PID: target.Key.PID, Identity: target.Key.Identity}}}}
			table := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: target.Key.PID, Identity: target.Key.Identity}}, ids: map[int]string{target.Key.PID: target.Key.Identity}, diesOn: map[int]syscall.Signal{target.Key.PID: syscall.SIGTERM}}
			var effects int
			call := func(ctx context.Context, _ string, raw any, out any) error {
				r := raw.(couchmessage.Request)
				reply := out.(*couchmessage.Response)
				if r.Op == orphanReapOp {
					reply.Code = "ok"
					reply.Resources = &view
					return nil
				}
				if r.Op != "reap-orphan-signal" {
					t.Fatalf("unexpected request %+v", r)
				}
				effects++
				if effects > 1 {
					t.Fatal("uncertain/refused effect retried")
				}
				if status == "lost" {
					return errors.New("reply lost")
				}
				if status == "unsupported" {
					reply.Code = "unsupported"
					return nil
				}
				result := &couchmessage.ProvenOrphanSignalResult{Target: *r.OrphanTarget, Status: status, Detail: "inspect resources"}
				if status == "sent" {
					if err := table.Signal(r.OrphanTarget.Key.PID, syscall.SIGTERM); err != nil {
						t.Fatal(err)
					}
				}
				if status == "wrong-target" {
					result.Target.Key.Identity = "different"
					result.Status = "sent"
				}
				reply.Code = "ok"
				reply.OrphanSignal = result
				return nil
			}
			var out, errw bytes.Buffer
			code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, testRT{dir: t.TempDir()}, &out, &errw, call, fastOrphanReaper(table))
			var report orphanReapReport
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatalf("%v %s %s", err, out.String(), errw.String())
			}
			if effects != 1 || len(report.Proven) != 1 || len(report.LeftRunning) != 0 {
				t.Fatalf("effects %d report %+v", effects, report)
			}
			want := "failed"
			if status == "sent" {
				want = "reaped"
			}
			if status == "refused" {
				want = "refused"
			}
			if report.Proven[0].Status != want || (code == 0) != (status == "sent") {
				t.Fatalf("exit %d report %+v", code, report)
			}
			if status != "sent" && len(table.signals) != 0 {
				t.Fatalf("legacy fallback signalled %+v", table.signals)
			}
		})
	}
}

func TestReapProvenPreviewNamesExactTarget(t *testing.T) {
	target := *effectTarget()
	view := couchcore.ResourceView{Orphans: []couchcore.ResourceProcessView{{ResourceProvenance: couchcore.ResourceProvenance{ResourceRoot: couchcore.ResourceRoot{Key: target.Key, Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace}, Root: target.Root, Orphan: true, OrphanKnown: true}}}}
	var calls []couchmessage.Request
	var out, errw bytes.Buffer
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &calls), fastOrphanReaper(&fakeOrphanTable{}))
	if code != 1 || len(calls) != 1 || !strings.Contains(out.String(), "proven-orphan repo:0 pid=11") || !strings.Contains(out.String(), "old-child") {
		t.Fatalf("exit %d calls %+v text %s%s", code, calls, out.String(), errw.String())
	}
}
