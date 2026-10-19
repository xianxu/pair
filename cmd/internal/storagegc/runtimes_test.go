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
	for _, raw := range []string{`null`, `{}`, `{"version":2,"processes":[]}`, `{"version":1,"processes":[]} {}`, `{"version":1,"processes":[{"id":"x","process":{"pid":1,"birth":"bogus"},"role":"couch-runtime"}]}`} {
		if err := os.WriteFile(c.runtimesPath(), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := c.ReadRuntimes(); err == nil {
			t.Fatal("malformed runtimes accepted", raw)
		}
		if err := c.RegisterRuntime(context.Background(), p, "couch-runtime"); err == nil {
			t.Fatal("malformed registry overwritten", raw)
		}
		after, err := os.ReadFile(c.runtimesPath())
		if err != nil || string(after) != raw {
			t.Fatalf("rejected registry changed: %s %v", after, err)
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

func TestRootRuntimePreservesExtensionsAcrossChangedWrites(t *testing.T) {
	c, _ := coordinatorFixture(t)
	birth := "darwin:123.4"
	c.Probe = &FakeProcessProbe{Processes: map[int]string{1: birth, 2: birth, 3: birth}}
	raw := `{"version":1,"future_envelope":{"proof":[1,null,"x"]},"processes":[{"id":"first","process":{"pid":1,"birth":"darwin:123.4","future_process":{"nested":true}},"role":"future-role","future_row":[false,3]},{"id":"dead","process":{"pid":2,"birth":"darwin:123.4"},"role":"future-role"}]}`
	if err := c.WithLock(context.Background(), func(*Locked) error { return os.WriteFile(c.runtimesPath(), []byte(raw), 0600) }); err != nil {
		t.Fatal(err)
	}
	p := ProcessIdentity{PID: 3, Birth: birth}
	if err := c.RegisterRuntime(context.Background(), p, "new-role", RuntimeRegistrationOptions{DiagnosticProtocol: 17}); err != nil {
		t.Fatal(err)
	}
	assertRuntimeExtensions(t, c)
	delete(c.Probe.(*FakeProcessProbe).Processes, 2)
	if err := c.RegisterRuntime(context.Background(), p, "new-role"); err != nil {
		t.Fatal(err)
	}
	assertRuntimeExtensions(t, c)
	entries, err := c.ReadRuntimes()
	if err != nil || len(entries) != 2 {
		t.Fatalf("prune: %+v %v", entries, err)
	}
	if entries[1].Process != p || entries[1].DiagnosticProtocol != 17 {
		t.Fatalf("appended capability lost during prune: %+v", entries)
	}
}

func assertRuntimeExtensions(t *testing.T, c *Coordinator) {
	t.Helper()
	raw, err := os.ReadFile(c.runtimesPath())
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	var envelope any
	if err := json.Unmarshal(root["future_envelope"], &envelope); err != nil {
		t.Fatalf("envelope extension lost: %s", raw)
	}
	if fmt.Sprint(envelope) != "map[proof:[1 <nil> x]]" {
		t.Fatalf("envelope changed: %s", raw)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(root["processes"], &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if string(row["id"]) == `"first"` {
			var proc map[string]json.RawMessage
			if err := json.Unmarshal(row["process"], &proc); err != nil {
				t.Fatal(err)
			}
			if string(row["future_row"]) != "[false,3]" || string(proc["future_process"]) != `{"nested":true}` {
				t.Fatalf("row/process extensions lost: %s", raw)
			}
			return
		}
	}
	t.Fatal("first runtime lost")
}

func TestRootRuntimeDiagnosticCapabilityProjection(t *testing.T) {
	c, _ := coordinatorFixture(t)
	raw := `{"version":1,"processes":[{"id":"first","process":{"pid":1,"birth":"darwin:123.4"},"role":"couch-runtime","diagnostic_protocol":17}]}`
	if err := c.WithLock(context.Background(), func(*Locked) error { return os.WriteFile(c.runtimesPath(), []byte(raw), 0600) }); err != nil {
		t.Fatal(err)
	}
	entries, err := c.ReadRuntimes()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(projection, &rows); err != nil {
		t.Fatal(err)
	}
	if string(rows[0]["diagnostic_protocol"]) != "17" {
		t.Fatalf("capability missing from known projection: %s", projection)
	}
}

func TestRootRuntimeDiagnosticCapabilityUpgradesInPlace(t *testing.T) {
	c, _ := coordinatorFixture(t)
	birth := "darwin:123.4"
	p := ProcessIdentity{PID: 1, Birth: birth}
	c.Probe = &FakeProcessProbe{Processes: map[int]string{1: birth}}
	raw := `{"version":1,"future_envelope":{"proof":[1,null,"x"]},"processes":[{"id":"first","process":{"pid":1,"birth":"darwin:123.4","future_process":{"nested":true}},"role":"couch-runtime","future_row":[false,3]}]}`
	if err := c.WithLock(context.Background(), func(*Locked) error { return os.WriteFile(c.runtimesPath(), []byte(raw), 0600) }); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterRuntime(context.Background(), p, CouchRuntimeRole, RuntimeRegistrationOptions{DiagnosticProtocol: DiagnosticProtocolVersion}); err != nil {
		t.Fatal(err)
	}
	assertRuntimeExtensions(t, c)
	entries, err := c.ReadRuntimes()
	if err != nil || len(entries) != 1 || entries[0].ID != "first" || entries[0].DiagnosticProtocol != DiagnosticProtocolVersion {
		t.Fatalf("in-place upgrade: %+v %v", entries, err)
	}
	// A repeated or old generic caller never writes or downgrades the lease.
	c.BeforePersist = func() error { return errors.New("unexpected idempotent rewrite") }
	for _, options := range [][]RuntimeRegistrationOptions{nil, {{}}, {{DiagnosticProtocol: DiagnosticProtocolVersion}}} {
		if err := c.RegisterRuntime(context.Background(), p, CouchRuntimeRole, options...); err != nil {
			t.Fatal(err)
		}
	}
	c.BeforePersist = nil
	if err := c.RegisterRuntime(context.Background(), p, CouchRuntimeRole, RuntimeRegistrationOptions{DiagnosticProtocol: 17}); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterRuntime(context.Background(), p, CouchRuntimeRole); err != nil {
		t.Fatal(err)
	}
	entries, err = c.ReadRuntimes()
	if err != nil || entries[0].DiagnosticProtocol != 17 {
		t.Fatalf("future capability lost: %+v %v", entries, err)
	}
	// Comparable public values remain suitable as identity-map keys.
	_ = map[ProcessRegistration]bool{entries[0]: true}
	_ = map[ProcessIdentity]bool{p: true}
}

func TestRootRuntimeCapabilityPublicationFailurePreservesPreviousRegistry(t *testing.T) {
	for _, boundary := range []string{"before-persist", "after-temp-write", "cancel-after-temp-write"} {
		t.Run(boundary, func(t *testing.T) {
			c, _ := coordinatorFixture(t)
			p := ProcessIdentity{PID: 1, Birth: "darwin:123.4"}
			c.Probe = &FakeProcessProbe{Processes: map[int]string{1: p.Birth}}
			if err := c.RegisterRuntime(context.Background(), p, CouchRuntimeRole); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(c.runtimesPath())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := errors.New("publication interrupted")
			switch boundary {
			case "before-persist":
				c.BeforePersist = func() error { return want }
			case "after-temp-write":
				c.AfterTempWrite = func(string) error { return want }
			case "cancel-after-temp-write":
				c.AfterTempWrite = func(string) error { cancel(); return nil }
				want = context.Canceled
			}
			err = c.RegisterRuntime(ctx, p, CouchRuntimeRole, RuntimeRegistrationOptions{DiagnosticProtocol: DiagnosticProtocolVersion})
			if !errors.Is(err, want) {
				t.Fatalf("publication error: %v", err)
			}
			after, err := os.ReadFile(c.runtimesPath())
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatalf("failed publication changed registry: %s -> %s", before, after)
			}
			c.BeforePersist = nil
			c.AfterTempWrite = nil
			if err := c.RegisterRuntime(context.Background(), p, CouchRuntimeRole, RuntimeRegistrationOptions{DiagnosticProtocol: DiagnosticProtocolVersion}); err != nil {
				t.Fatal(err)
			}
			entries, err := c.ReadRuntimes()
			if err != nil || entries[0].DiagnosticProtocol != DiagnosticProtocolVersion {
				t.Fatalf("retry: %+v %v", entries, err)
			}
		})
	}
}
