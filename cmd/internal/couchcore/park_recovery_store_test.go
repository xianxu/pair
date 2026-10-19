package couchcore

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParkRecoveryReadAndWriteRefuseBusyStore(t *testing.T) {
	for _, op := range []string{"read", "retire", "operation"} {
		t.Run(op, func(t *testing.T) {
			env, current := interruptedParkEnv(t, false)
			store := env.Couch.Threads
			lock, err := acquireThreadStoreLockMode(store.root, false)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				var err error
				if op == "read" {
					_, err = store.recoveryReadThread(ctx, current.Address)
				} else if op == "operation" {
					_, err = env.Couch.recoverPark(ctx, current.Address)
				} else {
					_, err = store.RetireInterruptedPark(ctx, current.Address, current.Revision, current.Park.Identity, env.Now)
				}
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("busy store accepted recovery")
				}
				if op == "operation" && !strings.Contains(err.Error(), "Recover park") {
					t.Fatalf("busy recovery omitted next action: %v", err)
				}
			case <-time.After(time.Second):
				_ = lock.Close()
				<-done
				t.Fatal("recovery blocked on a legacy unbounded store lock")
			}
		})
	}
}
