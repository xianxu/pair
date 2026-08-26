package couchcore

import (
	"testing"
	"time"
)

func TestThreadStoreLockSerializesIndependentHandles(t *testing.T) {
	_, ns := newTestThreadStore(t)
	root := NewThreadStore(ns).root
	first, err := acquireThreadStoreLock(root)
	if err != nil {
		t.Fatalf("acquire first: %v", err)
	}
	acquired := make(chan *threadStoreLock, 1)
	errs := make(chan error, 1)
	go func() {
		second, err := acquireThreadStoreLock(root)
		if err != nil {
			errs <- err
			return
		}
		acquired <- second
	}()
	select {
	case second := <-acquired:
		_ = second.Close()
		t.Fatal("second independent handle entered the critical section")
	case err := <-errs:
		t.Fatalf("second lock errored instead of waiting: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := first.Close(); err != nil {
		t.Fatalf("release first: %v", err)
	}
	select {
	case second := <-acquired:
		defer second.Close()
	case err := <-errs:
		t.Fatalf("second lock after release: %v", err)
	case <-time.After(time.Second):
		t.Fatal("second lock did not acquire after release")
	}
}
