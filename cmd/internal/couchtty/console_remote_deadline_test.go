package couchtty

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestRemoteDeadlineIncludesQueueWait(t *testing.T) {
	r := newRemoteRig(t)
	prepared := false
	finished := make(chan error, 1)
	err := r.c.EnqueueRemoteOperationDeadline("drive-slot", "keys", time.Now().Add(-time.Second), func(context.Context) (couchcore.OperationCall, error) {
		prepared = true
		return couchcore.OperationCall{Name: "keys"}, nil
	}, func() {}, func(_ any, err error) { finished <- err })
	if err != nil {
		t.Fatal(err)
	}
	r.drain(t)
	if prepared || len(r.calls) != 0 {
		t.Fatal("expired queued operation reached prepare or effects")
	}
	if err = <-finished; err == nil {
		t.Fatal("expired job reported success")
	}
}
func TestRemoteDeadlineReachesDispatcher(t *testing.T) {
	r := newRemoteRig(t)
	deadline := time.Now().Add(time.Second)
	r.resume = func(call couchcore.OperationCall) (any, error) {
		got, ok := call.Context.Deadline()
		if !ok || !got.Equal(deadline) {
			t.Errorf("deadline %v present %v", got, ok)
		}
		return nil, nil
	}
	err := r.c.EnqueueRemoteOperationDeadline("drive-slot", "keys", deadline, func(context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "keys"}, nil
	}, func() {}, func(any, error) {})
	if err != nil {
		t.Fatal(err)
	}
	r.drain(t)
}
