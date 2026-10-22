package remotesession

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRemoteDocumentRequestIsAnAllowlist(t *testing.T) {
	valid := `{"sequence":1,"action":"edit","document":{"id":"` + originDocA + `","generation":"editor-1"},"base_revision":"revision-1","edit":{"start":0,"end":0,"text":"hello"}}`
	if _, err := decodeDocumentRequest([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		strings.Replace(valid, `"edit","document"`, `"shell","document"`, 1),
		strings.Replace(valid, `"sequence":1`, `"Sequence":1`, 1),
		strings.Replace(valid, `"start":0`, `"Start":0`, 1),
		strings.Replace(valid, `"generation":"editor-1"`, `"generation":"../../socket"`, 1),
		strings.Replace(valid, `"sequence":1`, `"sequence":0`, 1),
		strings.Replace(valid, `"hello"`, `"\u0000"`, 1),
		strings.Replace(valid, `"start":0`, `"start":-1`, 1),
		strings.Replace(valid, `"start":0`, `"start":null`, 1),
		strings.Replace(valid, `"start":0,`, ``, 1),
		strings.Replace(valid, `"base_revision":"revision-1"`, `"base_revision":null`, 1),
		strings.Replace(valid, `"sequence":1`, `"origin":"remote","sequence":1`, 1),
		valid + ` {}`,
	} {
		if _, err := decodeDocumentRequest([]byte(raw)); err == nil {
			t.Fatalf("unsafe action admitted: %s", raw)
		}
	}
}
func FuzzRemoteDocumentRequest(f *testing.F) {
	f.Add([]byte(`{"sequence":1,"action":"read","document":{"id":"` + originDocA + `","generation":"editor-1"}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		request, err := decodeDocumentRequest(raw)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		again, err := decodeDocumentRequest(encoded)
		if err != nil || again.Action != request.Action || again.Document != request.Document {
			t.Fatal("typed request cannot roundtrip", err)
		}
	})
}
func TestRemotePermitCannotBeForgedOrUsedAfterKill(t *testing.T) {
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	calls := 0
	if (Permit{}).EffectDoor(ref, func() bool { calls++; return true }) {
		t.Fatal("zero permit accepted")
	}
	gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 7}}
	done := make(chan struct{})
	now := time.Now()
	p := newPermit(gate, sessionRef{epoch: 7, done: done}, ref, context.Background(), now.Add(time.Hour), time.Now)
	other := ref
	other.Generation = "editor-2"
	if p.EffectDoor(other, func() bool { calls++; return true }) {
		t.Fatal("permit crossed editor generation")
	}
	gate.mu.Lock()
	gate.state.phase = phaseKilled
	gate.mu.Unlock()
	if p.EffectDoor(ref, func() bool { calls++; return true }) || calls != 0 {
		t.Fatal("effect after kill")
	}
}
func TestRemotePermitLinearizesHandoffAndIsSingleUse(t *testing.T) {
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 8}}
	done := make(chan struct{})
	p := newPermit(gate, sessionRef{epoch: 8, done: done}, ref, context.Background(), time.Now().Add(time.Hour), time.Now)
	calls := 0
	if !p.EffectDoor(ref, func() bool { calls++; return true }) {
		t.Fatal("current permit refused")
	}
	if p.EffectDoor(ref, func() bool { calls++; return true }) || calls != 1 {
		t.Fatal("effect handed off twice")
	}
	gate.mu.Lock()
	gate.state.phase = phaseKilled
	gate.mu.Unlock()
	// A completed handoff remains an effect, not a fictitious rollback.
	if calls != 1 {
		t.Fatal("kill rolled back already handed off effect")
	}
}
func TestRemotePermitSessionAndAccessBoundaries(t *testing.T) {
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	for _, boundary := range []string{"session", "expiry", "context", "epoch"} {
		t.Run(boundary, func(t *testing.T) {
			now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
			gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 9}}
			done := make(chan struct{})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			expiry := now.Add(time.Hour)
			p := newPermit(gate, sessionRef{epoch: 9, done: done}, ref, ctx, expiry, func() time.Time { return now })
			switch boundary {
			case "session":
				close(done)
			case "expiry":
				now = expiry
			case "context":
				cancel()
			case "epoch":
				gate.state.epoch++
			}
			if p.EffectDoor(ref, func() bool { t.Fatal("revoked callback invoked"); return true }) {
				t.Fatal("revoked permit accepted")
			}
		})
	}
}
func TestRemotePermitConcurrentUseHasOneHandoff(t *testing.T) {
	ref := DocumentRef{ID: originDocA, Generation: "editor-1"}
	gate := &admissionGate{state: remoteState{phase: phaseReady, epoch: 10}}
	p := newPermit(gate, sessionRef{epoch: 10, done: make(chan struct{})}, ref, context.Background(), time.Now().Add(time.Hour), time.Now)
	count := 0
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); p.EffectDoor(ref, func() bool { count++; return true }) }()
	}
	wg.Wait()
	if count != 1 {
		t.Fatalf("%d effect handoffs", count)
	}
}

func TestRemoteNavigationMustResolveBeforeEffectDoor(t *testing.T) {
	request := DocumentRequest{Action: ActionNavigate, Document: DocumentRef{ID: originDocA, Generation: "source"}}
	if request.EffectDocument().valid() {
		t.Fatal("unresolved navigation reaches effect door")
	}
	request.destination = DocumentRef{ID: originDocB, Generation: "destination"}
	if request.EffectDocument() != request.destination {
		t.Fatal("resolved target not used at effect door")
	}
}
