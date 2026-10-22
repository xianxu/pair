package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// A live wrapper process owns its endpoint and reconnecting registry session.
// This fixture models the external agent boundary with retained deliveries;
// it has no Console, attach operation or pane lifecycle producer.
func TestMessageUnfocusedWrapperProcess(t *testing.T) {
	raw := os.Getenv("PAIR_TEST_MESSAGE_WRAPPER")
	if raw == "" {
		t.Skip("subprocess fixture")
	}
	var config struct {
		Binding                                        couchmessage.Binding
		Namespace, Registry, Ready, Release, Delivered string
	}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatal(err)
	}
	process, err := (couchcore.OSProcOps{}).Current()
	if err != nil {
		t.Fatal(err)
	}
	b := config.Binding
	b.PID, b.Start = process.PID, process.Identity
	socket, err := couchmessage.EndpointSocket(config.Namespace, b)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var reservation string
	receipts := map[string]couchmessage.Receipt{}
	server, err := couchmessage.StartServer(ctx, socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.EndpointRequest
		if err := json.Unmarshal(raw, &request); err != nil {
			return nil, err
		}
		if err := couchmessage.ValidateEndpointRequest(request); err != nil {
			return nil, err
		}
		if request.Binding != b {
			return nil, errors.New("foreign wrapper binding")
		}
		mu.Lock()
		defer mu.Unlock()
		response := couchmessage.EndpointResponse{}
		switch request.Op {
		case "observe":
			response.Observation = couchmessage.Observation{Sequence: 1, LastActivity: time.Now().Add(-time.Hour)}
		case "reserve":
			if receipt, ok := receipts[request.ID]; ok {
				response.Error = couchmessage.ErrAlreadyCommitted.Error()
				response.Receipt = &receipt
			} else if reservation != "" || request.Sequence != 1 {
				response.Error = couchmessage.ErrRecipientBusy.Error()
			} else {
				reservation = request.ID
			}
		case "release":
			if reservation == request.ID {
				reservation = ""
			}
		case "commit":
			if reservation != request.ID {
				response.Error = "not reserved"
				break
			}
			receipt := couchmessage.Receipt{Message: *request.Message, Status: couchmessage.Submitted}
			receipts[request.ID], response.Receipt = receipt, &receipt
			file, err := os.OpenFile(config.Delivered, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return nil, err
			}
			err = json.NewEncoder(file).Encode(receipt.Message)
			file.Close()
			if err != nil {
				return nil, err
			}
		case "status":
			if receipt, ok := receipts[request.ID]; ok {
				response.Receipt = &receipt
			} else {
				response.Error = couchmessage.ErrUnknownDelivery.Error()
			}
		}
		return json.Marshal(response)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	encoded, _ := json.Marshal(b)
	if err := (osfs.FS{}).WriteAtomic(config.Ready, string(encoded)); err != nil {
		t.Fatal(err)
	}
	client := couchmessage.NewSessionClient()
	client.Update(couchmessage.Observation{Sequence: 1, LastActivity: time.Now().Add(-time.Hour)})
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.Run(ctx, config.Registry, b, couchmessage.DefaultReconnectBackoff, 10*time.Millisecond)
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(config.Release); err == nil {
				cancel()
				<-done
				return
			}
		}
	}
}

func messageFixtureFile(t *testing.T, path string) []byte {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	deadline := testwait.After(t)
	for {
		raw, err := os.ReadFile(path)
		if err == nil {
			return raw
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("fixture file unavailable: %s: %v", path, err)
		}
	}
}

func TestMessageUnfocusedWrappersReconnectAndExchangeWithoutAttachment(t *testing.T) {
	rt := newRT(t, "/repo1", "/repo2")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
	dir, err := os.MkdirTemp("", "pcmd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	r.brokerSock, r.regSock = filepath.Join(dir, "broker.sock"), filepath.Join(dir, "registry.sock")
	records := []couchcore.ThreadRecord{seedDetachedThread(t, rt, "/repo1"), seedDetachedThread(t, rt, "/repo2")}
	var bindings []couchmessage.Binding
	var deliveries []string
	for i, record := range records {
		original := r.world.add(i + 1)
		b := original
		b.Scope, b.Tag, b.Session, b.Agent = record.Address.RepoScope, string(record.Address.Tag), fmt.Sprintf("listed-session-%d", i+1), "claude"
		files := t.TempDir()
		ready, release, delivered := filepath.Join(files, "ready"), filepath.Join(files, "release"), filepath.Join(files, "delivered")
		config := struct {
			Binding                                        couchmessage.Binding
			Namespace, Registry, Ready, Release, Delivered string
		}{b, dir, r.regSock, ready, release, delivered}
		raw, _ := json.Marshal(config)
		cmd := exec.Command(os.Args[0], "-test.run=^TestMessageUnfocusedWrapperProcess$")
		cmd.Env = append(os.Environ(), "PAIR_TEST_MESSAGE_WRAPPER="+string(raw))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		t.Cleanup(func() {
			_ = os.WriteFile(release, []byte("release"), 0600)
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("wrapper process: %v", err)
				}
			case <-testwait.After(t):
				_ = cmd.Process.Kill()
				<-done
				t.Error("wrapper did not stop")
			}
		})
		var live couchmessage.Binding
		if err := json.Unmarshal(messageFixtureFile(t, ready), &live); err != nil {
			t.Fatal(err)
		}
		r.world.mu.Lock()
		slot := r.world.slots[original.Thread()]
		delete(r.world.slots, original.Thread())
		slot.binding, slot.pidFile = live, live.PID
		r.world.slots[live.Thread()] = slot
		r.world.mu.Unlock()
		rt.artifacts.SetDetachedSession(record.Address, live.Session)
		bindings = append(bindings, live)
		deliveries = append(deliveries, delivered)
	}
	start := func() {
		authority := r.world.authority()
		authority.thread = messageThreadResolver(c, nil)
		authority.process = func(b couchmessage.Binding) error {
			identity, err := (couchcore.OSProcOps{}).Identity(b.PID)
			if err != nil || identity != b.Start || (couchcore.OSProcOps{}).Exists(b.PID) != couchcore.Live {
				return errors.New("wrapper process changed")
			}
			return nil
		}
		authority.endpoint = func(b couchmessage.Binding) couchmessage.DeliveryEndpoint {
			return couchmessage.RemoteEndpoint{Namespace: dir, Binding: b}
		}
		r.panes = couchmessage.NewPaneMailbox()
		r.s, err = newMessageService(context.Background(), r.brokerSock, r.regSock, filepath.Dir(r.brokerSock), authority, r.panes, func(string) (couchcore.SlotGitStatus, bool) { return couchcore.SlotGitStatus{}, false }, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(r.s.Close)
	}
	start()
	for _, b := range bindings {
		r.waitConnected(b, true)
	}
	r.s.Close()
	start()
	for _, b := range bindings {
		r.waitConnected(b, true)
	}
	for i, from := range bindings {
		to := bindings[1-i]
		id := fmt.Sprintf("unfocused-%d", i)
		if response := r.send(from, id, to.Slot); response.Code != "accepted" {
			t.Fatalf("unfocused send: %+v", response)
		}
		receipt := r.waitStatus(from, id, terminal)
		if receipt.Status != couchmessage.Submitted || receipt.Message.To != to {
			t.Fatalf("delivery: %+v", receipt)
		}
		if raw := messageFixtureFile(t, deliveries[1-i]); !strings.Contains(string(raw), id) {
			t.Fatalf("wrong recipient delivery: %s", raw)
		}
	}
	if pending := r.panes.Drain(); len(pending) != 0 {
		t.Fatalf("registration fabricated panes: %v", pending)
	}
	if len(rt.runner.Ops) != 0 {
		t.Fatalf("messaging launched or attached a client: %v", rt.runner.Ops)
	}
	for _, before := range records {
		after, err := c.Threads.GetThread(before.Address)
		rawBefore, _ := json.Marshal(before)
		rawAfter, _ := json.Marshal(after)
		if err != nil || string(rawBefore) != string(rawAfter) {
			t.Fatalf("messaging mutated retained thread: %v", err)
		}
	}
}

func TestMessageThreadResolverRefusesUnlistedAndForeignTerminal(t *testing.T) {
	rt := newRT(t, "/repo")
	record := seedDetachedThread(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	resolve := messageThreadResolver(c, nil)
	binding := couchmessage.Binding{Scope: record.Address.RepoScope, Tag: string(record.Address.Tag), Session: "listed-session"}
	if _, err := resolve(context.Background(), binding); err == nil {
		t.Fatal("retained record alone granted message authority")
	}
	rt.artifacts.SetDetachedSession(record.Address, binding.Session)
	if root, err := resolve(context.Background(), binding); err != nil || root != "/repo" {
		t.Fatalf("listed terminal: %q %v", root, err)
	}
	binding.Session = "foreign-session"
	if _, err := resolve(context.Background(), binding); err == nil {
		t.Fatal("foreign listed session accepted")
	}
	binding.Session = "listed-session"
	rt.artifacts.SetDetachedSession(record.Address, "")
	if _, err := resolve(context.Background(), binding); err == nil {
		t.Fatal("disappeared session stayed authorized")
	}
}

// Automatic registration must refuse an interrupted transaction, rather than
// gaining lifecycle authority to replay it merely by reading a retained thread.
func TestMessageThreadResolverLeavesPendingJournalUntouched(t *testing.T) {
	rt := newRT(t, "/repo")
	record := seedDetachedThread(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	recordPath, err := c.Threads.RecordPath(record.Address)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var pending map[string]any
	if err := json.Unmarshal(before, &pending); err != nil {
		t.Fatal(err)
	}
	pending["description"] = "must not publish from broker startup"
	pending["revision"] = record.Revision + 1
	after, err := json.Marshal(pending)
	if err != nil {
		t.Fatal(err)
	}
	storeRoot := filepath.Join(c.Namespace.Dir(), "threadstore")
	relative, err := filepath.Rel(storeRoot, recordPath)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := json.Marshal(map[string]any{"schema_version": 1, "entries": []map[string]any{{"path": relative, "expected": before, "after": after}}})
	if err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(storeRoot, "journal.json")
	if err := os.WriteFile(journalPath, journal, 0600); err != nil {
		t.Fatal(err)
	}
	binding := couchmessage.Binding{Scope: record.Address.RepoScope, Tag: string(record.Address.Tag), Session: "listed-terminal"}
	rt.artifacts.SetDetachedSession(record.Address, binding.Session)
	if _, err := messageThreadResolver(c, nil)(context.Background(), binding); err == nil {
		t.Fatal("automatic resolver accepted pending transaction")
	}
	retained, err := os.ReadFile(recordPath)
	if err != nil || string(retained) != string(before) {
		t.Fatalf("automatic resolver rewrote record: %v", err)
	}
	pendingJournal, err := os.ReadFile(journalPath)
	if err != nil || string(pendingJournal) != string(journal) {
		t.Fatalf("automatic resolver recovered journal: %v", err)
	}
	// Prove this is a replayable transaction, not a malformed fixture that
	// every accessor would reject. Only this explicit lifecycle read may apply it.
	replayed, err := c.Threads.GetThread(record.Address)
	if err != nil || replayed.Description != "must not publish from broker startup" {
		t.Fatalf("explicit lifecycle recovery fixture: %+v %v", replayed, err)
	}
}
