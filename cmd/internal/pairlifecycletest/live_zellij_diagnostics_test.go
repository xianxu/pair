package pairlifecycletest

import (
	"bytes"
	"sync"
	"testing"
)

func TestZellijStartupCaptureBoundedDuringDrain(t *testing.T) {
	capture := &zellijStartupCapture{}
	prefix := bytes.Repeat([]byte("a"), 16384)
	if n, err := capture.Write(prefix); err != nil || n != len(prefix) {
		t.Fatalf("prefix: %d %v", n, err)
	}
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for range 100 {
			capture.String()
		}
	}()
	payload := bytes.Repeat([]byte("b"), 65536)
	for range 100 {
		if n, err := capture.Write(payload); err != nil || n != len(payload) {
			t.Errorf("drain: %d %v", n, err)
		}
	}
	workers.Wait()
	if got := capture.String(); got != string(prefix) {
		t.Fatalf("capture retained %d bytes or changed prefix", len(got))
	}
}
