package couchtty

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestOperationQueueCoalescesAndRefusesOverloadWithoutEffects(t *testing.T) {
	queue := newOperationQueue(1)
	effects := 0
	want := couchcore.ThreadAddress{RepoScope: "scope", Tag: "tag"}
	request := operationRequest{key: "park/scope/tag", name: "park", run: func() (any, error) {
		effects++
		return want, nil
	}}
	accepted, err := queue.Enqueue(request)
	if !accepted || err != nil {
		t.Fatalf("first enqueue = %v, %v", accepted, err)
	}
	accepted, err = queue.Enqueue(request)
	if accepted || err != nil {
		t.Fatalf("duplicate enqueue = %v, %v", accepted, err)
	}
	accepted, err = queue.Enqueue(operationRequest{key: "park/scope/other", name: "park", run: func() (any, error) {
		effects++
		return nil, nil
	}})
	if accepted || !errors.Is(err, errOperationQueueOverloaded) || effects != 0 {
		t.Fatalf("overload = accepted %v err %v effects %d", accepted, err, effects)
	}

	stop := make(chan struct{})
	defer close(stop)
	go queue.Run(stop)
	select {
	case result := <-queue.results:
		if result.err != nil || result.name != "park" || result.key != request.key || !reflect.DeepEqual(result.value, want) || effects != 1 {
			t.Fatalf("result = %+v effects %d", result, effects)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for queued operation")
	}

	retry := operationRequest{key: "park/scope/other", name: "park", run: func() (any, error) {
		effects++
		return "retried", nil
	}}
	accepted, err = queue.Enqueue(retry)
	if !accepted || err != nil {
		t.Fatalf("overload did not restore rejected request = %v, %v", accepted, err)
	}
	result := <-queue.results
	if result.key != retry.key || result.value != "retried" || effects != 2 {
		t.Fatalf("retried result = %+v effects %d", result, effects)
	}
	select {
	case extra := <-queue.results:
		t.Fatalf("request emitted duplicate completion %+v", extra)
	case <-time.After(10 * time.Millisecond):
	}
}

func TestOperationQueueCarriesTypedResultForEveryMenuOperation(t *testing.T) {
	for _, operation := range []string{"switch", "resume", "park", "name", "describe", "start", "leave"} {
		t.Run(operation, func(t *testing.T) {
			queue := newOperationQueue(1)
			stop := make(chan struct{})
			defer close(stop)
			go queue.Run(stop)
			want := struct {
				Operation string
				Sequence  int
			}{Operation: operation, Sequence: 7}
			accepted, err := queue.Enqueue(operationRequest{
				key: operation + "/scope/tag", name: operation,
				run: func() (any, error) { return want, nil },
			})
			if !accepted || err != nil {
				t.Fatalf("enqueue = %v, %v", accepted, err)
			}
			select {
			case result := <-queue.results:
				if result.name != operation || result.key != operation+"/scope/tag" || !reflect.DeepEqual(result.value, want) || result.err != nil {
					t.Fatalf("completion = %+v", result)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for typed result")
			}
		})
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
			profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
			record, err := store.CreateThread(couchcore.ThreadRecord{
				SchemaVersion: couchcore.ThreadSchemaVersion,
				Address:       couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"},
				StartingPath:  "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), LastActiveAt: time.Unix(1, 0).UTC(),
				Incarnations:        []couchcore.ThreadIncarnation{{PID: 42, Identity: "helper", State: couchcore.IncarnationLive, RepoIdentity: "/repo/.git", LaunchProfile: &profile}},
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
