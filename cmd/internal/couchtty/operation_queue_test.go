package couchtty

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestOperationQueueCoalescesAndRefusesOverloadWithoutEffects(t *testing.T) {
	queue := newOperationQueue(1)
	effects := 0
	request := operationRequest{key: "park/scope/tag", name: "park", run: func() error {
		effects++
		return nil
	}}
	accepted, err := queue.Enqueue(request)
	if !accepted || err != nil {
		t.Fatalf("first enqueue = %v, %v", accepted, err)
	}
	accepted, err = queue.Enqueue(request)
	if accepted || err != nil {
		t.Fatalf("duplicate enqueue = %v, %v", accepted, err)
	}
	accepted, err = queue.Enqueue(operationRequest{key: "park/scope/other", name: "park", run: func() error {
		effects++
		return nil
	}})
	if accepted || !errors.Is(err, errOperationQueueOverloaded) || effects != 0 {
		t.Fatalf("overload = accepted %v err %v effects %d", accepted, err, effects)
	}

	stop := make(chan struct{})
	defer close(stop)
	go queue.Run(stop)
	select {
	case result := <-queue.results:
		if result.err != nil || result.name != "park" || effects != 1 {
			t.Fatalf("result = %+v effects %d", result, effects)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for queued operation")
	}
}

func TestChildExitNeverProvesPark(t *testing.T) {
	for _, phase := range []string{"requested", "awaiting", "completion-retained", "finalized"} {
		t.Run(phase, func(t *testing.T) {
			ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/repo")
			if err != nil {
				t.Fatal(err)
			}
			store := couchcore.NewThreadStore(ns)
			policy := couchcore.PolicyResult{
				PolicyVersion: 1, PolicyDigest: strings.Repeat("a", 64), RepoIdentity: "repo", AdmissionKey: "repo",
				Capacity: couchcore.PolicyCapacity{Kind: couchcore.CapacityUnbounded}, OnCapacity: couchcore.CapacityActionUnknown,
			}
			profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
			record, err := store.CreateThread(couchcore.ThreadRecord{
				SchemaVersion: couchcore.ThreadSchemaVersion,
				Address:       couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"},
				StartingPath:  "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), LastActiveAt: time.Unix(1, 0).UTC(),
				Incarnations:        []couchcore.ThreadIncarnation{{PID: 42, Identity: "helper", State: couchcore.IncarnationLive, Policy: &policy, LaunchProfile: &profile}},
				LatestLaunchProfile: &profile, Revision: 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			identity := couchcore.ParkIdentity{Nonce: "park-exit-proof", Address: record.Address, PID: 42, ProcessIdentity: "helper"}
			record, err = store.BeginPark(record.Address, record.Revision, identity)
			if err != nil {
				t.Fatal(err)
			}
			if phase != "requested" {
				record, err = store.AdvancePark(record.Address, record.Revision, couchcore.ParkEvent{Kind: couchcore.ParkRequestCommitted, Identity: identity, Attempt: 1})
				if err != nil {
					t.Fatal(err)
				}
			}
			if phase == "finalized" {
				record, err = store.FinalizePark(record.Address, record.Revision, identity, 1, time.Unix(2, 0).UTC())
				if err != nil {
					t.Fatal(err)
				}
			}

			console := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			child := ptychild.NewFakeChild(nil)
			console.attachThreadActor("handle", "actor", record.Address, "/repo", "repo", child)
			console.onExit(childExit{id: "handle", code: 0})
			child.Exit(0)
			console.Stop()
			after, err := store.GetThread(record.Address)
			if err != nil || !reflect.DeepEqual(after, record) {
				t.Fatalf("child exit changed durable phase %s: %+v, %v", phase, after, err)
			}
		})
	}
}
