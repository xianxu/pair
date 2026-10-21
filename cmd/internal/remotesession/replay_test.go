package remotesession

import (
	"bytes"
	"strconv"
	"testing"
)

func TestRemoteReplayNeverRedispatchesOldOrConflictingRequests(t *testing.T) {
	for _, n := range []int{8, 16, 512} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			var r Replay
			dispatches := 0
			for i := 1; i <= n; i++ {
				seq := uint64(i)
				got, err := r.Admit(seq, []byte("immutable request"))
				if err != nil || got.kind != replayNew {
					t.Fatal(got, err)
				}
				dispatches++
				duplicate, err := r.Admit(seq, []byte("immutable request"))
				if err != nil || duplicate.kind != replayPending {
					t.Fatal("in-flight duplicate", duplicate, err)
				}
				if err := r.Complete(seq, []byte("immutable request"), []byte("receipt"), false); err != nil {
					t.Fatal(err)
				}
				cached, err := r.Admit(seq, []byte("immutable request"))
				if err != nil || cached.kind != replayCached || !bytes.Equal(cached.result, []byte("receipt")) {
					t.Fatal("cache", cached, err)
				}
				if _, err := r.Admit(seq, []byte("different request")); err == nil {
					t.Fatal("same ID different payload")
				}
			}
			if dispatches != n || len(r.entries) > 256 {
				t.Fatal("unbounded or repeated dispatch")
			}
			if n > 256 {
				if _, err := r.Admit(1, []byte("immutable request")); err == nil {
					t.Fatal("evicted sequence re-admitted")
				}
			}
		})
	}
}
func TestRemoteReplayPinsUncertaintyAndRefusesAtCapacity(t *testing.T) {
	var r Replay
	for i := uint64(1); i <= 256; i++ {
		if _, err := r.Admit(i, []byte("request")); err != nil {
			t.Fatal(err)
		}
		if err := r.Complete(i, []byte("request"), []byte("uncertain"), true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Admit(257, []byte("request")); err == nil {
		t.Fatal("uncertainty evicted")
	}
	if err := r.Complete(1, []byte("request"), []byte("confirmed receipt"), false); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Admit(257, []byte("request")); err != nil {
		t.Fatal("reconciliation did not free capacity", err)
	}
	if _, err := r.Admit(1, []byte("request")); err == nil {
		t.Fatal("reconciled evicted request re-executed")
	}
}
func TestRemoteReplayBoundsAndOwnsBytes(t *testing.T) {
	var r Replay
	if _, err := r.Admit(0, nil); err == nil {
		t.Fatal("zero sequence")
	}
	if _, err := r.Admit(1, make([]byte, 65537)); err == nil {
		t.Fatal("unbounded request")
	}
	_, err := r.Admit(1, []byte("request"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Complete(1, []byte("request"), make([]byte, 1025), false); err == nil {
		t.Fatal("unbounded result")
	}
	result := []byte("receipt")
	if err := r.Complete(1, []byte("request"), result, false); err != nil {
		t.Fatal(err)
	}
	result[0] = 'X'
	cached, err := r.Admit(1, []byte("request"))
	if err != nil {
		t.Fatal(err)
	}
	if string(cached.result) != "receipt" {
		t.Fatal("caller mutated retained receipt")
	}
	cached.result[0] = 'Y'
	cached, _ = r.Admit(1, []byte("request"))
	if string(cached.result) != "receipt" {
		t.Fatal("result alias escaped")
	}
}
