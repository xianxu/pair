package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type ownershipReader struct {
	*samplerReader
	envErrors map[int]error
}

// Double P and S independently of detached candidates. The proof shell must
// still take exactly one table and one inventory observation per effect and
// perform no metrics/argv/cwd probes. Existing classifier ordering stays shared.
func TestCollectOwnershipCountsRowsAndSlotsIndependently(t *testing.T) {
	for _, rows := range []int{8, 16} {
		for _, slots := range []int{8, 16} {
			t.Run(fmt.Sprintf("P%d-S%d", rows, slots), func(t *testing.T) {
				f := &samplerReader{}
				for i := 0; i < rows; i++ {
					f.rows = append(f.rows, procutil.Process{PID: 100 + i, PPID: 1, Identity: fmt.Sprint(i), Name: "zellij"})
				}
				s := testResourceSampler(f)
				var reads atomic.Int64
				s.slots = func() couchcore.ResourceSlotInventory {
					reads.Add(1)
					inventory := couchcore.ResourceSlotInventory{Complete: true}
					for i := 0; i < slots; i++ {
						inventory.Slots = append(inventory.Slots, couchcore.ResourceSlot{Slot: fmt.Sprint(i), OwnershipKnown: true})
					}
					return inventory
				}
				for i := 0; i < 16; i++ {
					sample, _, _ := s.collectOwnership(context.Background())
					if len(sample.Table) != rows || len(sample.Slots) != slots {
						t.Fatal("ownership inputs changed")
					}
				}
				if f.tables != 16 || reads.Load() != 16 || len(f.usages)+len(f.commands)+len(f.cwds)+len(f.environments) != 0 {
					t.Fatalf("tables %d slots %d extra probes", f.tables, reads.Load())
				}
			})
		}
	}
}

func (r ownershipReader) Environment(ctx context.Context, p procutil.Process, keys ...string) (map[string]string, int, error) {
	e, n, err := r.samplerReader.Environment(ctx, p, keys...)
	if injected := r.envErrors[p.PID]; injected != nil {
		return nil, n, injected
	}
	return e, n, err
}

func TestCollectOwnershipPreservesDetachedProtection(t *testing.T) {
	thread := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "1-pair-2"}
	root := couchcore.ResourceRoot{Key: couchcore.ResourceProcessKey{PID: 10, Identity: "old"}}
	slots := []couchcore.ResourceSlot{{Slot: "pair:2", Thread: thread, OwnershipKnown: true, Roots: []couchcore.ResourceRoot{root}}}
	before := []procutil.Process{{PID: 10, PPID: 1, Identity: "old", Name: "codex"}, {PID: 30, PPID: 10, Identity: "suite", Name: "sdlc"}, {PID: 31, PPID: 30, Identity: "test", Name: "couchcore.test"}}
	prior := couchcore.SelectResourceProcesses(before, slots, nil, true, true).Processes
	after := append([]procutil.Process(nil), before...)
	after[1].PPID = 1
	for _, tc := range []struct {
		name    string
		env     map[string]string
		err     error
		unknown int
		orphan  bool
	}{
		{"owned", map[string]string{"COUCH_THREAD_SCOPE": thread.RepoScope, "COUCH_THREAD_TAG": string(thread.Tag)}, nil, 0, false},
		{"known absence", map[string]string{"PATH": "/usr/bin"}, nil, 0, true},
		{"withheld", nil, nil, 1, true},
		{"unreadable", nil, errors.New("denied"), 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &samplerReader{rows: after, env: map[int]map[string]string{30: tc.env, 31: {"PATH": "/usr/bin"}}}
			s := testResourceSampler(f)
			s.reader = ownershipReader{f, map[int]error{30: tc.err}}
			s.state.Provenance = prior
			s.slots = func() couchcore.ResourceSlotInventory {
				return couchcore.ResourceSlotInventory{Slots: slots, Complete: true}
			}
			sample, selection, unknown := s.collectOwnership(context.Background())
			if !sample.TableKnown || !sample.InventoryComplete || len(unknown) != tc.unknown {
				t.Fatalf("sample %+v unknown %+v", sample, unknown)
			}
			for _, p := range selection.Processes {
				if p.Key.PID == 31 && p.Orphan != tc.orphan {
					t.Fatalf("selection %+v", selection)
				}
			}
			if len(f.usages)+len(f.commands)+len(f.cwds) != 0 {
				t.Fatal("ownership proof collected metrics")
			}
		})
	}
}

func TestCollectOwnershipCountsColdAndWarmGenerations(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := &samplerReader{env: map[int]map[string]string{}}
			for i := 0; i < n; i++ {
				pid := 100 + i
				f.rows = append(f.rows, procutil.Process{PID: pid, PPID: 1, Identity: fmt.Sprint(pid), Name: "sdlc"})
				f.env[pid] = map[string]string{"PATH": "/usr/bin"}
			}
			s := testResourceSampler(f)
			for i := 0; i < 2; i++ {
				s.collectOwnership(context.Background())
			}
			if f.tables != 2 || len(f.environments) != n || len(f.usages)+len(f.commands)+len(f.cwds) != 0 {
				t.Fatalf("tables %d env %d metrics %v", f.tables, len(f.environments), f.usages)
			}
		})
	}
}
