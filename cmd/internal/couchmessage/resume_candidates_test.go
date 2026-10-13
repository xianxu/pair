package couchmessage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResumeCandidateReceiptIdentity(t *testing.T) {
	event := ReceiptEvent{Kind: ReceiptAdmit, ID: "request", Op: "resume", Target: "pair:1", CandidateKey: "key-one", At: time.Now()}
	got, err := ApplyReceiptEvent(OperationReceipt{}, event)
	if err != nil {
		t.Fatal(err)
	}
	event.CandidateKey = "key-two"
	if _, err = ApplyReceiptEvent(got, event); !errors.Is(err, ErrReceiptIDConflict) {
		t.Fatalf("changed key replay: %v", err)
	}
}
func TestCandidateKeyOnlyAppliesToResume(t *testing.T) {
	r := Request{Op: "resume", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", ID: "request", Target: "pair:1", CandidateKey: "key"}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"actors", "resume-candidates", "operation-status", "reboot"} {
		r.Op = op
		if ValidateRequest(r) == nil {
			t.Fatalf("key accepted for %s", op)
		}
	}
}

func TestRecoveryPreviewTransportCancellation(t *testing.T) {
	socket := transportSocket(t)
	entered := make(chan time.Duration, 1)
	stopped := make(chan struct{})
	server, err := StartServer(context.Background(), socket, func(ctx context.Context, _ []byte) ([]byte, error) {
		deadline, _ := ctx.Deadline()
		entered <- time.Until(deadline)
		<-ctx.Done()
		close(stopped)
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	request := Request{Op: "resume-candidates", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Target: "pair:1"}
	go func() { var response Response; done <- Call(ctx, socket, request, &response) }()
	select {
	case remaining := <-entered:
		if remaining <= TransportTimeout || remaining > RecoveryPreviewTimeout {
			t.Fatalf("preview bound=%v", remaining)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preview never entered")
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("caller cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client ignored cancellation")
	}
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("disconnected client left scan running")
	}
}
