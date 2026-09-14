package storagegc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestRootRuntimeRegistrationIsDurableIdempotentAndReadOnly(t *testing.T) {
	c, _ := coordinatorFixture(t)
	if _, err := c.ReadRuntimes(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing runtimes: %v", err)
	}
	files, _ := os.ReadDir(c.Root)
	if len(files) != 0 {
		t.Fatal("read initialized metadata")
	}
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := c.ReadRuntimes()
	if err != nil || len(entries) != 1 || entries[0].Process != p || entries[0].Role != "couch-runtime" || entries[0].ID == "" {
		t.Fatalf("runtime registration: %+v %v", entries, err)
	}
}

func TestRootRuntimePrunesOnlyVerifiedDeadProcesses(t *testing.T) {
	c, _ := coordinatorFixture(t)
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	probe := &FakeProcessProbe{Processes: map[int]string{}, Unknown: map[int]bool{}}
	c.Probe = probe
	for i := 1; i <= 2; i++ {
		p.PID = i
		probe.Processes[i] = p.Birth
		if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err != nil {
			t.Fatal(err)
		}
	}
	probe.Unknown[1] = true
	p.PID = 3
	probe.Processes[3] = p.Birth
	if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err != nil {
		t.Fatal(err)
	}
	entries, err := c.ReadRuntimes()
	if err != nil || len(entries) != 3 {
		t.Fatal("unknown runtime discarded", entries, err)
	}
	delete(probe.Unknown, 1)
	delete(probe.Processes, 1)
	if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err != nil {
		t.Fatal(err)
	}
	entries, err = c.ReadRuntimes()
	if err != nil || len(entries) != 2 {
		t.Fatal("dead runtime not reaped", entries, err)
	}
}

func TestRootRuntimeRejectsMalformedAndStaleIdentity(t *testing.T) {
	c, _ := coordinatorFixture(t)
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	stale := p
	stale.Birth = "bogus"
	if err := c.RegisterRuntime(context.Background(), stale, "couch-runtime"); err == nil {
		t.Fatal("malformed birth registered")
	}
	if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`null`, `{}`, `{"version":2,"processes":[]}`, `{"version":1,"processes":[],"extra":true}`, `{"version":1,"processes":[]} {}`, `{"version":1,"processes":[{"id":"x","process":{"pid":1,"birth":"bogus"},"role":"couch-runtime"}]}`} {
		if err := os.WriteFile(c.runtimesPath(), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := c.ReadRuntimes(); err == nil {
			t.Fatal("malformed runtimes accepted", raw)
		}
		if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err == nil {
			t.Fatal("malformed registry overwritten", raw)
		}
	}
}

func TestRootRuntimeLimitRetainsUnknownEntries(t *testing.T) {
	c, _ := coordinatorFixture(t)
	p, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err != nil {
		t.Fatal(err)
	}
	entries := make([]ProcessRegistration, 256)
	probe := &FakeProcessProbe{Processes: map[int]string{}, Unknown: map[int]bool{}}
	for i := range entries {
		p.PID = i + 1
		entries[i] = ProcessRegistration{ID: fmt.Sprint(i + 1), Process: p, Role: "couch-runtime"}
		probe.Unknown[p.PID] = true
	}
	raw, err := json.Marshal(runtimeRegistry{Version: 1, Processes: entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.runtimesPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	p.PID = 10000
	probe.Processes[p.PID] = p.Birth
	c.Probe = probe
	if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err == nil {
		t.Fatal("runtime bound exceeded")
	}
	got, err := c.ReadRuntimes()
	if err != nil || len(got) != 256 {
		t.Fatal("unknown entries discarded", err)
	}
}
