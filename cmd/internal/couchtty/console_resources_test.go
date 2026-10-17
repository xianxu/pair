package couchtty

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestResourceInventoryNotificationDoesNotCallIOUnderLock(t *testing.T) {
	c := &Console{}
	changes := c.ResourceChanges()
	c.mu.Lock()
	c.resourceRows = []couchcore.ActionableThreadSummary{{Name: "one", Recovery: &couchcore.RecoveryDecision{Diagnosis: "proof"}}}
	c.resourcesChangedLocked()
	c.mu.Unlock()
	select {
	case <-changes:
	default:
		t.Fatal("missing change")
	}
	snapshot := c.ResourceInventory()
	snapshot.Rows[0].Name = "mutated"
	snapshot.Rows[0].Recovery.Diagnosis = "mutated"
	if c.ResourceInventory().Rows[0].Recovery.Diagnosis != "proof" {
		t.Fatal("nested metadata aliases snapshot")
	}
	if c.ResourceInventory().Rows[0].Name != "one" {
		t.Fatal("snapshot aliases inventory")
	}
	c.mu.Lock()
	c.resourceRows = nil
	c.resourcesChangedLocked()
	c.mu.Unlock()
	if len(c.ResourceInventory().Rows) != 0 || c.ResourceInventory().Generation <= snapshot.Generation {
		t.Fatal("removal not published")
	}
}

func TestResourceInventoryRefreshIgnoresStaleResults(t *testing.T) {
	c := &Console{refreshRequests: make(chan struct{}, 1), refreshResults: make(chan menuRefreshResult, 1)}
	c.ResourceChanges()
	// Invoke the snapshot publication section through the real refresh finisher.
	c.refreshSchedule.Running = 2
	c.finishMenuRefresh(menuRefreshResult{generation: 1, inventory: []couchcore.ActionableThreadSummary{{Name: "stale"}}})
	if len(c.ResourceInventory().Rows) != 0 {
		t.Fatal("stale inventory published")
	}
}

func TestHostedIngestSnapshotTracksChildIncarnations(t *testing.T) {
	first := ptychild.NewFakeChild([]byte("first"))
	second := ptychild.NewFakeChild([]byte("hidden"))
	replacement := ptychild.NewFakeChild([]byte("new"))
	for _, child := range []*ptychild.Child{first, second, replacement} {
		t.Cleanup(func() { child.Close() })
	}
	address := couchcore.ThreadAddress{RepoScope: "pair", Tag: "1"}
	c := &Console{order: []string{"missing", "one", "nil-child", "two"}, panes: map[string]*pane{
		"one":       {child: first, thread: address, process: couchcore.ProcessIdentity{PID: 12, Identity: "old"}},
		"nil-child": {},
		"two":       {child: second, thread: couchcore.ThreadAddress{RepoScope: "pair", Tag: "2"}},
	}}
	before := time.Now()
	snapshot := c.HostedIngestSnapshot()
	if snapshot.SampledAt.Before(before) || snapshot.SampledAt.After(time.Now()) {
		t.Fatalf("invalid sample time %v", snapshot.SampledAt)
	}
	if len(snapshot.Clients) != 2 || snapshot.Omitted != 0 {
		t.Fatalf("snapshot: %+v", snapshot)
	}
	old := snapshot.Clients[0]
	if old.EndpointID != first.Endpoint().ID() || old.Thread != address || old.Process.PID != 12 || old.Process.Identity != "old" || old.IngestBytes != 5 {
		t.Fatalf("wrong first identity/count: %+v", old)
	}
	if snapshot.Clients[1].EndpointID != second.Endpoint().ID() || snapshot.Clients[1].IngestBytes != 6 {
		t.Fatalf("hidden client missing: %+v", snapshot.Clients)
	}
	c.mu.Lock()
	c.panes["one"] = &pane{child: replacement, thread: address, process: couchcore.ProcessIdentity{PID: 12, Identity: "new"}}
	delete(c.panes, "two")
	c.mu.Unlock()
	next := c.HostedIngestSnapshot()
	if len(next.Clients) != 1 || next.Clients[0].EndpointID == old.EndpointID || next.Clients[0].Process.Identity != "new" || next.Clients[0].IngestBytes != 3 {
		t.Fatalf("replacement conflated: %+v", next)
	}
	if !reflect.DeepEqual(snapshot.Clients[0], old) {
		t.Fatal("snapshot changed after replacement")
	}
	next.Clients[0].EndpointID = "mutated"
	if c.HostedIngestSnapshot().Clients[0].EndpointID != replacement.Endpoint().ID() {
		t.Fatal("snapshot aliases console state")
	}
	replacement.Exit(0)
	<-replacement.Exited()
	if got := c.HostedIngestSnapshot(); len(got.Clients) != 0 || got.Clients == nil {
		t.Fatalf("exited child retained, or empty support absent: %+v", got)
	}
}

func TestHostedIngestSnapshotCapsLiveInventory(t *testing.T) {
	c := &Console{panes: make(map[string]*pane)}
	// Distinct handles can share this controlled child: census bounds concern
	// admitted pane rows, independently of byte totals and process availability.
	child := ptychild.NewFakeChild(nil)
	t.Cleanup(func() { child.Close() })
	for i := 0; i < couchcore.ResourceMaxSlots+7; i++ {
		id := fmt.Sprint(i)
		c.order = append(c.order, id)
		c.panes[id] = &pane{child: child, thread: couchcore.ThreadAddress{Tag: couchcore.ThreadTag(id)}}
	}
	got := c.HostedIngestSnapshot()
	if len(got.Clients) != couchcore.ResourceMaxSlots || got.Omitted != 7 {
		t.Fatalf("bounded inventory len=%d omitted=%d", len(got.Clients), got.Omitted)
	}
	for i, row := range got.Clients {
		if string(row.Thread.Tag) != fmt.Sprint(i) {
			t.Fatalf("unstable order at %d: %+v", i, row)
		}
	}
}

func TestHostedIngestSnapshotDoesNotWaitForParser(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	child, err := ptychild.Start(ptychild.Options{Argv: []string{"/bin/cat"}, Size: ptychild.Size{Rows: 24, Cols: 80}, Observer: func(observation terminal.Observation) {
		if observation.Kind == "endpoint-feed" {
			once.Do(func() { close(entered); <-release })
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer child.Close()
	defer unblock()
	fed := make(chan struct{})
	go func() { child.Feed([]byte("busy")); close(fed) }()
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("parser not entered")
	}
	c := &Console{order: []string{"busy"}, panes: map[string]*pane{"busy": {child: child}}}
	observed := make(chan couchcore.HostedIngestSnapshot, 1)
	go func() { observed <- c.HostedIngestSnapshot() }()
	select {
	case got := <-observed:
		if len(got.Clients) != 1 || got.Clients[0].IngestBytes != 4 {
			t.Fatalf("busy parser snapshot: %+v", got)
		}
	case <-testwait.After(t):
		t.Fatal("diagnostic snapshot waited on parser lock")
	}
	unblock()
	select {
	case <-fed:
	case <-testwait.After(t):
		t.Fatal("feed did not finish")
	}
}
