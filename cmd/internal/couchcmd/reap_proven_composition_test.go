package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Signals change the same native world that the real sampler worker observes.
// The foreign target remains present, including its legacy-registry overlap.
type composedReapTable struct {
	*fakeOrphanTable
	reader *samplerReader
}

func (p *composedReapTable) Signal(pid int, sig syscall.Signal) error {
	if err := p.fakeOrphanTable.Signal(pid, sig); err != nil {
		return err
	}
	p.reader.mu.Lock()
	var rows []procutil.Process
	for _, row := range p.reader.rows {
		if _, alive := p.ids[row.PID]; alive {
			rows = append(rows, row)
		}
	}
	p.reader.rows = rows
	p.reader.mu.Unlock()
	return nil
}

func composedReapWorld(t *testing.T) (*serviceRig, testRT, *composedReapTable) {
	t.Helper()
	rig, bindings := hubMessageRig(t)
	own := *effectTarget()
	foreign := own
	foreign.Key = couchcore.ResourceProcessKey{PID: 31, Identity: "foreign-child"}
	foreign.Root = couchcore.ResourceProcessKey{PID: 30, Identity: "foreign-root"}
	foreign.Slot = "repo:2"
	foreign.Thread.Tag = "1-repo-2"
	n, address, rest, head := 2, "repo:2", "main-slot2", strings.Repeat("a", 40)
	layout := couchcore.NewSlotLayout("/repo", "", 2)
	foreign.Workspace = couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: "/repo/.git", PrimaryRoot: "/repo", FleetRoot: "/", EnvironmentRoot: layout.Env(), WorktreeRoot: layout.Host(), Kind: "slot", Slot: &n, Address: &address, RestingBranch: &rest, Head: &head, EnvironmentHost: &couchcore.EnvironmentHost{Repo: "repo", Slot: 2, RepoIdentity: "/repo/.git", PrimaryRoot: "/repo", WorktreeRoot: layout.Host()}}
	if err := foreign.Validate(); err != nil {
		t.Fatal(err)
	}
	old := []procutil.Process{{PID: 10, PPID: 1, Identity: "old-root", Name: "zellij"}, {PID: 11, PPID: 10, Identity: "old-child", Name: "codex"}, {PID: 30, PPID: 1, Identity: "foreign-root", Name: "zellij"}, {PID: 31, PPID: 30, Identity: "foreign-child", Name: "codex"}}
	slots := []couchcore.ResourceSlot{}
	for _, target := range []couchcore.ProvenOrphanTarget{own, foreign} {
		root := couchcore.ResourceRoot{Key: target.Root, Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace}
		slots = append(slots, couchcore.ResourceSlot{Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{root}})
	}
	prior := couchcore.SelectResourceProcesses(old, slots, nil, true, true).Processes
	slots[0].Roots[0].Key = couchcore.ResourceProcessKey{PID: 20, Identity: "current-own"}
	slots[1].Roots[0].Key = couchcore.ResourceProcessKey{PID: 40, Identity: "current-foreign"}
	current := []procutil.Process{{PID: 11, PPID: 1, Identity: "old-child", Name: "codex"}, {PID: 31, PPID: 1, Identity: "foreign-child", Name: "codex"}, {PID: 20, PPID: 1, Identity: "current-own", Name: "zellij"}, {PID: 40, PPID: 1, Identity: "current-foreign", Name: "zellij"}}
	reader := &samplerReader{rows: current}
	sampler := testResourceSampler(reader)
	sampler.state.Provenance = prior
	sampler.slots = func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: slots}
	}
	sampler.sweeps = make(chan resourceRequest)
	sampler.policy.Cadence = time.Hour
	ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatal(err)
	}
	sampler.leftRunning = couchcore.NewLeftRunningRegistry(ns)
	var overlap []couchcore.LeftRunningOrphan
	for _, target := range []couchcore.ProvenOrphanTarget{own, foreign} {
		overlap = append(overlap, couchcore.LeftRunningOrphan{Origin: "slot:" + target.Workspace.WorktreeRoot, Process: couchcore.ProcessIdentity{PID: target.Key.PID, Identity: target.Key.Identity}, Command: "codex", LeftAt: time.Now()})
	}
	if err = sampler.leftRunning.Add(t.Context(), overlap); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); sampler.run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	waitFor(t, func() bool { return len(sampler.view().Orphans) == 2 })
	table := &composedReapTable{fakeOrphanTable: &fakeOrphanTable{ids: map[int]string{}, diesOn: map[int]syscall.Signal{11: syscall.SIGTERM}}, reader: reader}
	for _, row := range current {
		table.rows = append(table.rows, launcher.ProcessRow{PID: row.PID, PPID: row.PPID, Identity: row.Identity, Name: row.Name})
		table.ids[row.PID] = row.Identity
	}
	rig.s.resources.Store(sampler)
	rig.s.provenOrphanProcesses = table
	caller := bindings[0]
	rt := testRT{dir: t.TempDir(), env: map[string]string{"COUCH_THREAD_SCOPE": caller.Scope, "COUCH_THREAD_TAG": caller.Tag, "PAIR_SESSION_NAME": caller.Session, "PAIR_LAUNCH_NONCE": caller.Nonce}}
	return rig, rt, table
}

func TestReapProvenCLIMixedOwnerServiceComposition(t *testing.T) {
	for _, confirmed := range []bool{false, true} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(fmt.Sprintf("confirmed=%v/json=%v", confirmed, jsonOutput), func(t *testing.T) {
				rig, rt, table := composedReapWorld(t)
				call := func(ctx context.Context, _ string, raw, out any) error {
					*out.(*couchmessage.Response) = rig.s.handle(ctx, raw.(couchmessage.Request))
					return nil
				}
				var stdout, stderr bytes.Buffer
				code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: confirmed, jsonOutput: jsonOutput}, rt, &stdout, &stderr, call, fastOrphanReaper(table))
				if code != 1 {
					t.Fatalf("mixed ownership must refuse foreign row: exit%d out%s err%s", code, stdout.String(), stderr.String())
				}
				ownStatus := "refused"
				if confirmed {
					ownStatus = "reaped"
				}
				if jsonOutput {
					var report orphanReapReport
					if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
						t.Fatalf("%v out%s err%s", err, stdout.String(), stderr.String())
					}
					if report.Confirmed != confirmed || len(report.Proven) != 2 || len(report.Outcomes) != 0 || len(report.LeftRunning) != 0 {
						t.Fatalf("multiplicity/legacy fallback: %+v", report)
					}
					seen := map[int]bool{}
					for _, row := range report.Proven {
						want := ownStatus
						if row.Target.Key.PID == 31 {
							want = "refused"
						}
						if seen[row.Target.Key.PID] || (row.Target.Key.PID != 11 && row.Target.Key.PID != 31) || row.Status != want {
							t.Fatalf("unexpected row %+v", row)
						}
						seen[row.Target.Key.PID] = true
					}
				} else {
					for _, want := range []string{ownStatus + " proven-orphan repo:0 pid=11 identity=old-child", "refused proven-orphan repo:2 pid=31 identity=foreign-child"} {
						if strings.Count(stdout.String(), want) != 1 {
							t.Fatalf("missing/duplicated %q in %s", want, stdout.String())
						}
					}
					if strings.Count(stdout.String(), "proven-orphan ") != 2 || strings.Contains(stdout.String(), "left-running ") {
						t.Fatalf("multiplicity/legacy fallback: %s", stdout.String())
					}
				}
				if confirmed {
					if len(table.signals) != 1 || len(table.signals[11]) != 1 || table.signals[11][0] != syscall.SIGTERM {
						t.Fatalf("signals %+v", table.signals)
					}
				} else if len(table.signals) != 0 {
					t.Fatalf("preview signalled %+v", table.signals)
				}
				for pid, identity := range map[int]string{20: "current-own", 31: "foreign-child", 40: "current-foreign"} {
					if table.Identity(pid) != identity {
						t.Fatalf("protected generation %d changed", pid)
					}
				}
			})
		}
	}
}
