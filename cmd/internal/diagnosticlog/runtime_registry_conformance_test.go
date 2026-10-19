package diagnosticlog

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestRuntimeRegistryOwnedChildConformance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := t.TempDir()
	path := filepath.Join(root, "owned-diagnostic")
	if err := os.WriteFile(path, []byte("owned"), 0600); err != nil {
		t.Fatal(err)
	}
	child := exec.CommandContext(ctx, "sh", "-c", `exec 3<"$1"; printf 'ready\n'; read line; exec 3<&-; printf 'closed\n'; read line`, "owned-reader", path)
	input, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { input.Close(); _ = child.Process.Kill(); _ = child.Wait() }()
	reader := bufio.NewReader(output)
	ready := make(chan error, 1)
	go func() { _, err := reader.ReadString('\n'); ready <- err }()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	identity, err := storagegc.CurrentProcessIdentity(child.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterRuntime(ctx, identity, storagegc.CouchRuntimeRole, storagegc.RuntimeRegistrationOptions{DiagnosticProtocol: storagegc.DiagnosticProtocolVersion}); err != nil {
		t.Fatal(err)
	}
	// Native FD inventory and lease publication/liveness are real. The portable
	// process catalog selects only our owned child, independent of other slots.
	world := &processWorld{rows: []procutil.Observation{{Process: procutil.Process{PID: identity.PID, Name: "couch", Identity: "owned"}, CommandInfo: procutil.CommandInfo{Executable: "/owned/couch"}}}, generation: "owned", strict: identity.Birth, env: map[string]string{}}
	native := OSInspection{processes: world}
	inspector := WithRuntimeRegistry(c, native)
	snapshot, err := inspector.(SweepSnapshot).Snapshot(ctx, []string{path})
	if err != nil {
		requireLsofObservation(t, err)
	}
	holders, err := snapshot.OpenFiles(ctx, path)
	requireLsofObservation(t, err)
	assertLsofHolder(t, holders, identity.PID, identity.Birth)
	if err := VerifyWriters(ctx, path, nil, snapshot); err == nil {
		t.Fatal("registered open holder was accepted")
	}
	if _, err := fmt.Fprintln(input, "close"); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	snapshot, err = inspector.(SweepSnapshot).Snapshot(ctx, []string{path})
	if err != nil {
		requireLsofObservation(t, err)
	}
	if err := VerifyWriters(ctx, path, nil, snapshot); err != nil {
		requireLsofObservation(t, err)
	}
	// The same PID cannot borrow a lease for a different birth.
	world.strict = "darwin:1.1"
	if err := VerifyWriters(ctx, path, nil, inspector); err == nil {
		t.Fatal("replacement incarnation borrowed lease")
	}
}
