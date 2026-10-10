package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcmd"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type peerCLIRuntime struct {
	couchcmd.Runtime // Any non-messaging operation must fail instead of touching ambient state.
	env              map[string]string
}

func (r peerCLIRuntime) Getenv(key string) string { return r.env[key] }

// This crosses the real broker, socket protocol, wrapper reservation, composer
// parser and input writer. Only the child application's painted screen is fake.
func TestPeerRuntimeBrokerToWrapperSubmitsOnce(t *testing.T) {
	namespace := t.TempDir()
	binding := func(slot, tag string) couchmessage.Binding {
		return couchmessage.Binding{Slot: slot, Repository: "/fixture/.git", Scope: "fixture-scope", Tag: tag, Session: "fixture-" + tag, Nonce: "launch-" + tag, Agent: "claude", Version: "test-fixture", PID: os.Getpid(), Start: "fixture-start"}
	}
	from, to := binding("brain:0", "sender"), binding("pair:1", "receiver")
	f, _ := peerIntegrationFixture(t)
	receiver := newPeerDelivery(to, time.Now)
	receiver.session = &recordingPeerSink{}
	sender := newPeerDelivery(from, time.Now)
	f.proxy.peer = receiver
	lifetime, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, d := range []*peerDelivery{sender, receiver} {
		socket, err := couchmessage.EndpointSocket(namespace, d.binding)
		if err != nil {
			t.Fatal(err)
		}
		server, err := couchmessage.StartServer(lifetime, socket, d.handleEndpoint)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = server.Close() })
	}
	broker := couchmessage.NewBroker(lifetime, time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
	t.Cleanup(func() { _ = broker.Close() })
	for _, b := range []couchmessage.Binding{from, to} {
		if err := broker.Register(b, couchmessage.RemoteEndpoint{Namespace: namespace, Binding: b}); err != nil {
			t.Fatal(err)
		}
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(lifetime, socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return nil, err
		}
		return json.Marshal(couchmessage.Handle(ctx, broker, request))
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	rt := peerCLIRuntime{env: map[string]string{"COUCH_STORE_DIR": namespace, "COUCH_THREAD_SCOPE": from.Scope, "COUCH_THREAD_TAG": from.Tag, "PAIR_SESSION_NAME": from.Session, "PAIR_LAUNCH_NONCE": from.Nonce}}
	var stdout, stderr bytes.Buffer
	if code := couchcmd.RunWithRuntime([]string{"--send-to", to.Slot, "--message", "Review pair#353; preserve human acceptance."}, nil, &stdout, &stderr, rt); code != 0 {
		t.Fatalf("CLI admission %d: %s", code, stderr.String())
	}
	fields := strings.Fields(stdout.String())
	if len(fields) != 5 || fields[4] != "queued" {
		t.Fatalf("CLI did not acknowledge admission: %q", stdout.String())
	}
	accepted, err := broker.Status(from, fields[0])
	if err != nil {
		t.Fatalf("admission %+v %v", accepted, err)
	}
	if _, err := broker.Send(lifetime, from, "runtime-second", couchmessage.Route{Target: to.Slot}, "Another instruction"); !errors.Is(err, couchmessage.ErrRecipientBusy) {
		t.Fatalf("pending message not preserved: %v", err)
	}
	if !waitFor(time.Second, func() bool { return receiver.receipt().Message.ID == accepted.Message.ID }) {
		t.Fatal("socket commit did not reach wrapper")
	}
	var input bytes.Buffer
	f.proxy.dispatchPeer(&input)
	paste := input.String()
	if !strings.HasPrefix(paste, "\x1b[200~") || !strings.HasSuffix(paste, "\x1b[201~") || !strings.Contains(paste, accepted.Message.Body) || !strings.Contains(paste, from.Slot) || !strings.Contains(paste, accepted.Message.ID) {
		t.Fatalf("missing canonical bracketed envelope: %q", paste)
	}
	f.proxy.dispatchPeer(&input)
	if input.String() != paste {
		t.Fatal("submitted before the post-paste delay")
	}
	// The fake child shows the paste; the wrapper submits after the fixed
	// delay without comparing it (pair#427), then confirms on the clear.
	peerRender(f, receiver, peerFilledComposer(strings.Split(peerEnvelope(accepted.Message), "\n")...))
	time.Sleep(PeerSubmitDelay)
	f.proxy.dispatchPeer(&input)
	f.proxy.dispatchPeer(&input)
	if input.String() != paste+"\r" {
		t.Fatalf("want exactly one submit after the delay: %q", input.String())
	}
	peerRender(f, receiver, peerEmptyComposer())
	f.proxy.dispatchPeer(&input)
	var receipt couchmessage.Receipt
	if !waitFor(2*time.Second, func() bool {
		var e error
		receipt, e = broker.Status(from, accepted.Message.ID)
		return e == nil && receipt.Status.Terminal()
	}) {
		t.Fatal("terminal wrapper receipt did not return through endpoint to broker")
	}
	if receipt.Status != couchmessage.Submitted || receipt.Message.ID != accepted.Message.ID || receipt.Message.From != from || receipt.Message.To != to || receipt.Message.Body != accepted.Message.Body {
		t.Fatalf("wrong canonical receipt %+v", receipt)
	}
	receiverReceipt, err := broker.Status(to, accepted.Message.ID)
	if err != nil || receiverReceipt.Status != couchmessage.Submitted {
		t.Fatalf("recipient cannot verify receipt %+v %v", receiverReceipt, err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := couchcmd.RunWithRuntime([]string{"--message-status", accepted.Message.ID, "--json"}, nil, &stdout, &stderr, rt); code != 0 {
		t.Fatalf("CLI receipt %d: %s", code, stderr.String())
	}
	var response couchmessage.Response
	if err := strictjson.Decode(stdout.Bytes(), &response); err != nil || response.Receipt == nil || response.Receipt.Status != couchmessage.Submitted {
		t.Fatalf("CLI canonical receipt mismatch: %s (%v)", stdout.String(), err)
	}
	message := response.Receipt.Message
	if !message.Deadline.Equal(accepted.Message.Deadline) {
		t.Fatal("CLI changed deadline")
	}
	// JSON preserves the instant, not Go's process-local monotonic clock data.
	message.Deadline = accepted.Message.Deadline
	if message != accepted.Message {
		t.Fatal("CLI changed canonical envelope")
	}
	if peerSubmits(receiver) != 0 {
		t.Fatal("peer submission falsely reset operator-only breaker")
	}
}
