package wrapcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// This is the presentation half of the detached clipboard CLI oracle: an
// accepted broker datagram is insufficient unless the real wrapper output
// carries its complete, decodable user notification while the child stays live.
func TestReplyRefusalReachesWrapperPresentationEnvelope(t *testing.T) {
	isolateNotificationSockets(t)
	binding := filepath.Join(t.TempDir(), "pair-wrap-pid")
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	reader, childOutput, err := os.Pipe()
	if err != nil {
		broker.Close()
		t.Fatal(err)
	}
	const refusal = "Copied; auto-quote refused: uncertain Zellij reply"
	presented := make(chan []byte, 1)
	var wire bytes.Buffer
	decoded := notificationWriter(func(data []byte) (int, error) {
		select {
		case presented <- append([]byte(nil), data...):
		default:
		}
		return len(data), nil
	})
	slugCalls := 0
	p := &proxy{ptmx: reader, notificationBroker: broker, spawnSlug: func() { slugCalls++ }, stdout: writerFunc(func(data []byte) (int, error) {
		wire.Write(data)
		return decoded.Write(data)
	})}
	done := make(chan struct{})
	prepareMasterPumpTransport(t, p, done)
	go func() { p.masterPump(); close(done) }()
	t.Cleanup(func() {
		broker.Close()
		childOutput.Close()
		p.childIO.Close()
		reader.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("refusal presentation fixture did not join")
		}
	})
	// Use the application's actual notification adapter, not Broker.Messages.
	if err := (clipcmd.OSRuntime{}).NotifyRefusal(refusal); err != nil {
		t.Fatal(err)
	}
	select {
	case envelope := <-presented:
		if !bytes.Equal(envelope, notifyosc.Encode(refusal)) {
			t.Fatalf("presentation=%q", envelope)
		}
	case <-testwait.After(t):
		t.Fatal("accepted refusal never reached wrapper output")
	}
	select {
	case <-done:
		t.Fatal("refusal ended the live child output stream")
	default:
	}
	const tail = "child output remains available"
	if _, err := childOutput.Write([]byte(tail)); err != nil {
		t.Fatal(err)
	}
	if err := childOutput.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("wrapper output did not finish")
	}
	if bytes.Count(wire.Bytes(), notifyosc.Encode(refusal)) != 1 || !bytes.Contains(wire.Bytes(), []byte(tail)) {
		t.Fatalf("output=%q", wire.Bytes())
	}
	if slugCalls != 0 || p.notificationLifecycle.Active || p.notificationLifecycle.Completed {
		t.Fatal("refusal altered agent turn lifecycle")
	}
}
