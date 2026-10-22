package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func remoteWireMessage(t *testing.T) Message {
	t.Helper()
	m := endpointMessage()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	object["Origin"] = json.RawMessage(`"remote"`)
	object["RemoteSource"] = json.RawMessage(`{"Email":"operator@example.com","IP":"192.0.2.3"}`)
	object["RemoteEdited"] = json.RawMessage(`true`)
	object["DocumentID"] = json.RawMessage(`"0123456789abcdef0123456789abcdef"`)
	object["Body"] = json.RawMessage(`"submitted: /private/session.md"`)
	raw, _ = json.Marshal(object)
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRemoteProvenanceWireEnvelopeAndValidation(t *testing.T) {
	m := remoteWireMessage(t)
	if err := ValidateEndpointRequest(EndpointRequest{Op: "commit", Binding: m.To, ID: m.ID, Message: &m}); err != nil {
		t.Fatalf("trusted remote rejected: %v", err)
	}
	if got := Envelope(m); !strings.Contains(got, "\n[remote operator@example.com 192.0.2.3]\nsubmitted: /private/session.md") {
		t.Fatalf("missing delivered provenance: %q", got)
	}
	raw, _ := json.Marshal(m)
	for _, field := range []string{`"RemoteEdited":true`, `"RemoteSource":`, `"DocumentID":`} {
		if !strings.Contains(string(raw), field) {
			t.Fatalf("lost %s: %s", field, raw)
		}
	}
}

type remoteCapableEndpoint struct {
	*FakeDeliveryEndpoint
	capabilityCalls int
}

func (e *remoteCapableEndpoint) SupportsRemoteProvenance(context.Context) error {
	e.capabilityCalls++
	return nil
}

func TestBrokerRemoteProvenanceRequiresCapabilityBeforeAdmission(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, supported := range []bool{false, true} {
		t.Run(fmt.Sprint(supported), func(t *testing.T) {
			b := NewBroker(context.Background(), func() time.Time { return now }, nil)
			defer b.Close()
			from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
			_ = b.Register(from, newFakeEndpoint(now))
			ep := newFakeEndpoint(now)
			var endpoint DeliveryEndpoint = ep
			capable := &remoteCapableEndpoint{FakeDeliveryEndpoint: ep}
			if supported {
				endpoint = capable
			}
			_ = b.Register(to, endpoint)
			source := RemoteSource{Email: "operator@example.com", IP: "192.0.2.3"}
			r, err := b.SendRemote(context.Background(), from, "remote", Route{Target: to.Slot}, "submitted: /private/session.md", source, true, "0123456789abcdef0123456789abcdef")
			if !supported {
				if err == nil || !errors.Is(err, ErrUnsupported) {
					t.Fatalf("legacy receiver admitted remote: %+v %v", r, err)
				}
				if info := b.LatestRefusals()[to.Slot]; info.Class != RefusalRelaunch {
					t.Fatalf("missing relaunch recovery classification: %+v", info)
				}
				refused, _ := b.Status(from, "remote")
				if refused.Status != Refused || refused.Message.RemoteSource != source {
					t.Fatalf("refusal lost provenance: %+v", refused)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			m := delivered(t, ep)
			if m != r.Message || m.RemoteSource != source || m.Origin != OriginRemote || !m.RemoteEdited {
				t.Fatalf("delivery changed provenance: %+v", m)
			}
			if _, err := b.SendRemote(context.Background(), from, "remote", Route{Target: to.Slot}, m.Body, RemoteSource{Email: "other@example.com", IP: source.IP}, true, m.DocumentID); err == nil {
				t.Fatal("duplicate ID changed source")
			}
			if _, err := b.SendLocal(context.Background(), from, "remote", Route{Target: to.Slot}, m.Body, true, m.DocumentID); err == nil {
				t.Fatal("local adopted remote identity")
			}
			ep.outcomes <- Receipt{Status: Submitted}
			waitStatus(t, b, from, "remote", Submitted)
			_, err = b.SendLocal(context.Background(), from, "local", Route{Target: to.Slot}, m.Body, true, m.DocumentID)
			if err != nil {
				t.Fatal(err)
			}
			local := delivered(t, ep)
			if !local.RemoteEdited || local.Origin != OriginPeer || strings.Contains(Envelope(local), "[remote ") {
				t.Fatalf("marked local gained header: %+v", local)
			}
			if capable.capabilityCalls != 1 {
				t.Fatalf("capability not cached per actor: %d", capable.capabilityCalls)
			}
			ep.outcomes <- Receipt{Status: Submitted}
		})
	}
}

func TestRemoteCapabilityClosedWireAndLegacyRefusal(t *testing.T) {
	for _, capability := range []string{"", "remote-provenance-v1", "remote-provenance-v2"} {
		t.Run(capability, func(t *testing.T) {
			var commits int
			socket := transportSocket(t)
			server, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				var request EndpointRequest
				if err := json.Unmarshal(raw, &request); err != nil {
					return nil, err
				}
				if request.Op != "remote-provenance" {
					commits++
					return json.Marshal(EndpointResponse{})
				}
				return json.Marshal(map[string]any{"RemoteProvenance": capability})
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			endpoint := RemoteEndpoint{Binding: endpointMessage().To, socket: socket}
			err = endpoint.SupportsRemoteProvenance(context.Background())
			if capability == "remote-provenance-v1" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("unknown capability accepted: %v", err)
			}
			if commits != 0 {
				t.Fatalf("capability wrote mail: %d", commits)
			}
		})
	}
	request := EndpointRequest{Op: "remote-provenance", Binding: endpointMessage().To, ID: "unexpected"}
	if err := ValidateEndpointRequest(request); err == nil {
		t.Fatal("capability query accepted payload")
	}
}

func TestRemoteProvenanceRejectsForgedAndOversizeFacts(t *testing.T) {
	original := remoteWireMessage(t)
	for _, kind := range []string{"empty-source", "email-newline", "email-bracket", "email-display", "ip-header-list", "ip-zone", "ip-noncanonical", "peer-source", "system-edited", "bad-document", "oversize-prefix"} {
		t.Run(kind, func(t *testing.T) {
			m := original
			switch kind {
			case "empty-source":
				m.RemoteSource = RemoteSource{}
			case "email-newline":
				m.RemoteSource.Email = "operator@example.com\nforged"
			case "email-bracket":
				m.RemoteSource.Email = "operator]@example.com"
			case "email-display":
				m.RemoteSource.Email = "Operator <operator@example.com>"
			case "ip-header-list":
				m.RemoteSource.IP = "192.0.2.1,192.0.2.2"
			case "ip-zone":
				m.RemoteSource.IP = "fe80::1%en0"
			case "ip-noncanonical":
				m.RemoteSource.IP = "2001:0db8::1"
			case "peer-source":
				m.Origin = OriginPeer
			case "system-edited":
				m.Origin = OriginSystem
				m.RemoteSource = RemoteSource{}
			case "bad-document":
				m.DocumentID = "../outside"
			case "oversize-prefix":
				m.Body = strings.Repeat("x", MaxBodyBytes)
			}
			if err := m.ValidateProvenance(); err == nil {
				t.Fatal("invalid source accepted")
			}
			if err := ValidateEndpointRequest(EndpointRequest{Op: "commit", Binding: m.To, ID: m.ID, Message: &m}); err == nil {
				t.Fatal("singleton accepted invalid facts")
			}
			if err := ValidateBatch([]Message{m}, m.To, m.ID); err == nil {
				t.Fatal("batch accepted invalid facts")
			}
		})
	}
	overhead := len("[remote  ]\n") + len(original.RemoteSource.Email) + len(original.RemoteSource.IP)
	original.Body = strings.Repeat("x", MaxBodyBytes-overhead)
	if err := original.ValidateProvenance(); err != nil {
		t.Fatalf("exact rendered bound refused: %v", err)
	}
}

func TestExternalRequestRejectsRemoteProvenanceShapes(t *testing.T) {
	for _, field := range []string{`"Origin":"remote"`, `"RemoteSource":{"Email":"operator@example.com","IP":"192.0.2.1"}`, `"RemoteEdited":true`, `"DocumentID":"0123456789abcdef0123456789abcdef"`} {
		var request Request
		if err := strictjson.Decode([]byte(`{"Op":"send",`+field+`}`), &request); err == nil {
			t.Fatalf("worker selected trusted provenance: %s", field)
		}
	}
}

func TestReceiptMatchingRetainsAllRemoteFacts(t *testing.T) {
	original := remoteWireMessage(t)
	for _, field := range []string{"origin", "email", "ip", "edited", "document"} {
		t.Run(field, func(t *testing.T) {
			changed := original
			switch field {
			case "origin":
				changed.Origin = OriginPeer
			case "email":
				changed.RemoteSource.Email = "other@example.com"
			case "ip":
				changed.RemoteSource.IP = "192.0.2.9"
			case "edited":
				changed.RemoteEdited = false
			case "document":
				changed.DocumentID = "fedcba9876543210fedcba9876543210"
			}
			if matchingReceipt(Receipt{Message: changed}, original) {
				t.Fatal("receipt changed trusted facts")
			}
		})
	}
}

type heldRemoteCapability struct {
	*FakeDeliveryEndpoint
	entered, release chan struct{}
}

func (e *heldRemoteCapability) SupportsRemoteProvenance(context.Context) error {
	close(e.entered)
	<-e.release
	return nil
}

func TestRemoteCapabilityCancellationCannotEnterBusyMailbox(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := &heldRemoteCapability{FakeDeliveryEndpoint: newFakeEndpoint(now), entered: make(chan struct{}), release: make(chan struct{})}
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	if _, err := b.Send(context.Background(), from, "first", Route{Target: to.Slot}, "first"); err != nil {
		t.Fatal(err)
	}
	delivered(t, ep.FakeDeliveryEndpoint)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := b.SendRemote(ctx, from, "remote", Route{Target: to.Slot}, "submitted: /private/session.md", RemoteSource{Email: "operator@example.com", IP: "192.0.2.1"}, false, "")
		result <- err
	}()
	<-ep.entered
	cancel()
	close(ep.release)
	if err := <-result; err == nil {
		t.Fatal("cancelled capability probe admitted remote into busy mailbox")
	}
	r, err := b.Status(from, "remote")
	if err != nil || r.Status != Refused {
		t.Fatalf("cancelled request is not definite refusal: %+v %v", r, err)
	}
	ep.outcomes <- Receipt{Status: Submitted}
}

func TestLegacySocketCapabilityRefusesRemoteWithoutDowngrade(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			now := time.Now()
			from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
			to.Agent = "claude"
			var writes atomic.Int32
			socket := transportSocket(t)
			server, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				var request EndpointRequest
				if err := strictjson.Decode(raw, &request); err != nil {
					return nil, err
				}
				response := EndpointResponse{}
				switch request.Op {
				case "remote-provenance", "prepare-batch":
					response.Error = fmt.Sprintf("unknown endpoint operation %q", request.Op)
				case "observe":
					response.Observation = Observation{Sequence: 1, LastActivity: now.Add(-time.Minute)}
				case "reserve", "release":
				case "commit":
					body, _ := json.Marshal(request.Message)
					var legacy struct {
						ID       string
						From, To Binding
						Body     string
						Deadline time.Time
					}
					if err := strictjson.Decode(body, &legacy); err != nil {
						return nil, err
					}
					writes.Add(1)
					response.Receipt = &Receipt{Message: Message{ID: legacy.ID, From: legacy.From, To: legacy.To, Body: legacy.Body, Deadline: legacy.Deadline}, Status: Submitted}
				default:
					response.Error = "unexpected endpoint operation"
				}
				return json.Marshal(response)
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			b := NewBroker(context.Background(), time.Now, nil)
			defer b.Close()
			_ = b.Register(from, newFakeEndpoint(now))
			endpoint := RemoteEndpoint{Binding: to, socket: socket}
			if batch {
				err = b.RegisterWithBatching(to, endpoint, HubAllowance, DefaultPeerBatchMaxChars)
			} else {
				err = b.Register(to, endpoint)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = b.SendRemote(context.Background(), from, "remote", Route{Target: to.Slot}, "submitted: /private/session.md", RemoteSource{Email: "operator@example.com", IP: "192.0.2.1"}, true, "0123456789abcdef0123456789abcdef")
			if !errors.Is(err, ErrUnsupported) || writes.Load() != 0 {
				t.Fatalf("legacy remote admission or downgrade: %v writes%d", err, writes.Load())
			}
			if _, err = b.Send(context.Background(), from, "peer", Route{Target: to.Slot}, "ordinary legacy peer"); err != nil {
				t.Fatal(err)
			}
			waitStatus(t, b, from, "peer", Submitted)
			if writes.Load() != 1 {
				t.Fatalf("legacy peer compatibility wrote%d", writes.Load())
			}
		})
	}
}

func TestRemotePromotionRequiresEveryCanonicalFact(t *testing.T) {
	for _, kind := range []string{"matching", "source", "edited", "document", "deadline", "admitted-at"} {
		t.Run(kind, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b, hub, ep, senders := queueBroker(t, now, "pair:0")
			from := senders["pair:0"]
			b.mu.Lock()
			b.actors[hub].endpoint = &remoteCapableEndpoint{FakeDeliveryEndpoint: ep}
			b.mu.Unlock()
			if _, err := b.Send(context.Background(), from, "first", Route{Target: hub.Slot}, "first"); err != nil {
				t.Fatal(err)
			}
			delivered(t, ep)
			original, err := b.SendRemote(context.Background(), from, "remote", Route{Target: hub.Slot}, "submitted: /private/session.md", RemoteSource{Email: "operator@example.com", IP: "192.0.2.1"}, true, "0123456789abcdef0123456789abcdef")
			if err != nil {
				t.Fatal(err)
			}
			retained := original
			retained.Status = Submitted
			switch kind {
			case "source":
				retained.Message.RemoteSource.Email = "other@example.com"
			case "edited":
				retained.Message.RemoteEdited = false
			case "document":
				retained.Message.DocumentID = "fedcba9876543210fedcba9876543210"
			case "deadline":
				retained.Message.Deadline = retained.Message.Deadline.Add(time.Second)
			case "admitted-at":
				retained.Message.AcceptedAt = retained.Message.AcceptedAt.Add(time.Second)
			}
			ep.mu.Lock()
			ep.retained.Add(retained)
			ep.mu.Unlock()
			ep.outcomes <- Receipt{Status: Submitted}
			want := Cancelled
			if kind == "matching" {
				want = Submitted
			}
			waitStatus(t, b, from, "remote", want)
			got, err := b.Status(from, "remote")
			if err != nil || got.Message != original.Message {
				t.Fatalf("recovery changed canonical receipt: %+v %v", got, err)
			}
		})
	}
}

func TestRemoteAuthorityCannotBeAdoptedFromRetainedWrapper(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := &remoteCapableEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now)}
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	remote := Message{Origin: OriginRemote, ID: "retained-remote", From: from, To: to, Body: "submitted: /private/session.md", Deadline: now.Add(PasteTimeout), RemoteSource: RemoteSource{Email: "operator@example.com", IP: "192.0.2.1"}, RemoteEdited: true, DocumentID: "0123456789abcdef0123456789abcdef"}
	ep.retained.Add(Receipt{Message: remote, Status: Submitted})
	if _, err := b.StatusContext(context.Background(), from, remote.ID); err == nil {
		t.Fatal("status adopted remote authority from wrapper")
	}
	if _, err := b.SendRemote(context.Background(), from, remote.ID, Route{Target: to.Slot}, remote.Body, remote.RemoteSource, remote.RemoteEdited, remote.DocumentID); err == nil {
		t.Fatal("reservation adopted remote authority from wrapper")
	}
}
