package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestLegacyFreezeOneCensusPartitionsOverlappingFamilies(t *testing.T) {
	table := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "a"}, {PID: 11, PPID: 10, Identity: "b"}, {PID: 12, PPID: 11, Identity: "c"}}}
	plan := freezeLegacyReap(context.Background(), fastOrphanReaper(table), []couchcore.OrphanNvim{orphanRow(10, "a")}, []couchcore.LeftRunningOrphan{{Process: couchcore.ProcessIdentity{PID: 11, Identity: "b"}}})
	if len(plan.targets) != 3 {
		t.Fatalf("frozen targets=%+v", plan.targets)
	}
	seen := map[couchcore.ResourceProcessKey]bool{}
	for _, v := range plan.targets {
		if seen[v.Key] {
			t.Fatal("duplicate target")
		}
		seen[v.Key] = true
	}
	table.rows = append(table.rows, launcher.ProcessRow{PID: 13, PPID: 10, Identity: "late"})
	if len(plan.targets) != 3 {
		t.Fatal("plan expanded after census")
	}
}

func TestLegacyFreezeRejectsUnknownCensusAndMissingRoot(t *testing.T) {
	for _, tc := range []struct {
		table  *fakeOrphanTable
		status string
	}{
		{&fakeOrphanTable{snapErr: errors.New("census unavailable")}, orphanFailed},
		{&fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: ""}}}, orphanRefused},
		{&fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "a"}, {PID: 10, Identity: "a"}}}, orphanRefused},
		{&fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 99, Identity: "other"}}}, orphanSkipped},
		{&fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "new"}}}, orphanSkipped},
	} {
		plan := freezeLegacyReap(context.Background(), fastOrphanReaper(tc.table), []couchcore.OrphanNvim{orphanRow(10, "a")}, nil)
		if len(plan.targets) != 0 || len(plan.nvim) != 1 || plan.nvim[0].Status != tc.status {
			t.Fatalf("status=%s plan=%+v", tc.status, plan)
		}
		if tc.table.snapErr != nil && !strings.Contains(plan.nvim[0].Detail, tc.table.snapErr.Error()) {
			t.Fatal("census uncertainty detail lost")
		}
	}
}

type legacyCensusTable struct {
	*fakeOrphanTable
	snapshots int
}

func (t *legacyCensusTable) Snapshot(ctx context.Context) ([]launcher.ProcessRow, error) {
	t.snapshots++
	return t.fakeOrphanTable.Snapshot(ctx)
}
func (t *legacyCensusTable) Signal(int, syscall.Signal) error { panic("CLI must never signal locally") }

func TestLegacyFreezeLinearForestVisits(t *testing.T) {
	for _, n := range []int{512, 1024} {
		table := &legacyCensusTable{fakeOrphanTable: &fakeOrphanTable{}}
		roots := make([]couchcore.OrphanNvim, 0, n)
		for i := 1; i <= n; i++ {
			table.rows = append(table.rows, launcher.ProcessRow{PID: i, PPID: i - 1, Identity: fmt.Sprint(i)})
			roots = append(roots, orphanRow(i, fmt.Sprint(i)))
		}
		plan := freezeLegacyReap(context.Background(), fastOrphanReaper(table), roots, nil)
		if table.snapshots != 1 || plan.visits != 2*n || len(plan.targets) != n {
			t.Fatalf("N=%d snapshots=%d visits=%d targets=%d", n, table.snapshots, plan.visits, len(plan.targets))
		}
	}
}

func legacyReply(target couchcore.LegacyOrphanTarget, status, detail string, out *couchmessage.Response) error {
	b, err := json.Marshal(map[string]any{"Code": "ok", "LegacySignal": map[string]any{"Target": target, "Status": status, "Detail": detail}})
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func TestLegacyEffectsUseSupervisorAndPreserveRefusedChild(t *testing.T) {
	host := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "a"}, {PID: 11, PPID: 10, Identity: "b"}}, ids: map[int]string{10: "a", 11: "b"}, diesOn: map[int]syscall.Signal{10: syscall.SIGTERM}}
	table := &legacyCensusTable{fakeOrphanTable: host}
	r := fastOrphanReaper(table)
	p := freezeLegacyReap(context.Background(), r, []couchcore.OrphanNvim{orphanRow(10, "a")}, nil)
	calls := 0
	call := func(ctx context.Context, socket string, request any, response any) error {
		req := request.(couchmessage.Request)
		out := response.(*couchmessage.Response)
		calls++
		if req.Op != "reap-legacy-signal" || !req.Confirmed || req.Scope != "scope" || req.Nonce != "nonce" {
			t.Fatalf("wrong effect %+v", req)
		}
		if req.LegacyTarget.Key.PID == 11 {
			return legacyReply(*req.LegacyTarget, "refused", "proof lost", out)
		}
		if err := host.Signal(req.LegacyTarget.Key.PID, syscall.SIGTERM); err != nil {
			return err
		}
		return legacyReply(*req.LegacyTarget, "sent", "", out)
	}
	nv, left := reapFrozenLegacy(context.Background(), "socket", couchmessage.Request{Scope: "scope", Nonce: "nonce"}, call, r, p)
	if calls != 2 || len(nv) != 1 || len(left) != 0 || nv[0].Status != orphanRefused || !strings.Contains(nv[0].Detail, "proof lost") {
		t.Fatalf("calls=%d nv=%+v left=%+v", calls, nv, left)
	}
	if host.ids[11] != "b" {
		t.Fatal("refused child was signalled")
	}
}

func TestLegacyUncertainRepliesNeverEscalateOrRetry(t *testing.T) {
	for _, mode := range []string{"lost", "missing", "mismatch", "failed", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			host := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "a"}}, ids: map[int]string{10: "a"}}
			table := &legacyCensusTable{fakeOrphanTable: host}
			r := fastOrphanReaper(table)
			p := freezeLegacyReap(context.Background(), r, []couchcore.OrphanNvim{orphanRow(10, "a")}, nil)
			calls := 0
			call := func(_ context.Context, _ string, request any, response any) error {
				req := request.(couchmessage.Request)
				out := response.(*couchmessage.Response)
				calls++
				switch mode {
				case "lost":
					return errors.New("reply lost")
				case "missing":
					out.Code = "ok"
					return nil
				case "mismatch":
					target := *req.LegacyTarget
					target.Key.Identity = "other"
					return legacyReply(target, "sent", "", out)
				default:
					return legacyReply(*req.LegacyTarget, mode, "uncertain", out)
				}
			}
			nv, _ := reapFrozenLegacy(context.Background(), "socket", couchmessage.Request{}, call, r, p)
			if calls != 1 || nv[0].Status != orphanFailed || host.ids[10] != "a" {
				t.Fatalf("calls=%d result=%+v", calls, nv)
			}
		})
	}
}

func TestLegacyEscalationUsesOnlyFrozenSupervisorKeys(t *testing.T) {
	host := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "a"}}, ids: map[int]string{10: "a"}, diesOn: map[int]syscall.Signal{10: syscall.SIGKILL}}
	table := &legacyCensusTable{fakeOrphanTable: host}
	r := fastOrphanReaper(table)
	p := freezeLegacyReap(context.Background(), r, []couchcore.OrphanNvim{orphanRow(10, "a")}, nil)
	host.rows = append(host.rows, launcher.ProcessRow{PID: 11, PPID: 10, Identity: "late"})
	host.ids[11] = "late"
	var signals []string
	call := func(ctx context.Context, _ string, request any, response any) error {
		req := request.(couchmessage.Request)
		out := response.(*couchmessage.Response)
		if req.LegacyTarget.Key.PID != 10 || req.LegacyTarget.Key.Identity != "a" || req.LegacyTarget.Root != req.LegacyTarget.Key {
			t.Fatalf("expanded target %+v", req.LegacyTarget)
		}
		signals = append(signals, req.OrphanSignal)
		sig := syscall.SIGTERM
		if req.OrphanSignal == "KILL" {
			sig = syscall.SIGKILL
		}
		if err := host.Signal(10, sig); err != nil {
			return err
		}
		return legacyReply(*req.LegacyTarget, "sent", "", out)
	}
	nv, _ := reapFrozenLegacy(context.Background(), "socket", couchmessage.Request{}, call, r, p)
	if strings.Join(signals, ",") != "TERM,KILL" || nv[0].Status != orphanReaped || host.ids[11] != "late" {
		t.Fatalf("signals=%v outcomes=%+v late=%q", signals, nv, host.ids[11])
	}
}

func TestLegacyFreezeMalformedRootNeverClaimsAbsence(t *testing.T) {
	table := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 10, Identity: "a"}}}
	for _, root := range []couchcore.OrphanNvim{orphanRow(10, ""), orphanRow(0, "a")} {
		p := freezeLegacyReap(context.Background(), fastOrphanReaper(table), []couchcore.OrphanNvim{root}, nil)
		if len(p.targets) != 0 || p.nvim[0].Status != orphanRefused {
			t.Fatalf("malformed root claimed safe absence: %+v", p)
		}
	}
}
