package couchcmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The nested launcher helper avoids a couchcore -> launcher -> couchcore test
// import cycle. Its only input is the actual env value produced by SwitchAgent.
func TestSwitchAgentProducesOrientationThroughLayoutAndWrapper(t *testing.T) {
	rt := newRT(t, "/repo")
	source := seedVerifiedPark(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	c.FreshRegistration = func(context.Context, couchcore.ThreadAddress, string, string) (bool, error) { return true, nil }
	data := t.TempDir()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: source.Address.RepoScope, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(data, "outgoing-native.jsonl")
	for path, body := range map[string]string{paths.Log(): "operator source prompt\n", paths.Draft(): "unsent draft\n", native: "native conversation\n"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	c.SwitchContext = couchcore.OSSwitchContextResolver{DataDir: data, HomeDir: data, Query: func(context.Context, sessioninventory.Runtime, string, string, sessioninventory.Agent) (sessioninventory.SessionQuery, error) {
		return sessioninventory.SessionQuery{Status: sessioninventory.BindingEstablished, Root: &sessioninventory.Node{NativeID: "outgoing-root", Artifacts: []sessioninventory.Artifact{{Kind: sessioninventory.ArtifactTranscript}}}}, nil
	}, NativePath: func(sessioninventory.Runtime, sessioninventory.Artifact) (string, error) { return native, nil }}

	rt.runner.AfterAcknowledge = func(string) error { rt.artifacts.SetPairSession(source.Address, "pair-switched", true); return nil }
	args := []string{"--model", "test model"}
	prepared, err := c.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.SwitchAgent(context.Background(), couchcore.SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: prepared.Fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := result.Started()
	if !ok || result.Orientation == nil {
		t.Fatalf("no orientation child: %+v", result)
	}
	profile := ""
	for _, entry := range rt.runner.Child(child.Handle.ID()).Env {
		if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
			profile = strings.TrimPrefix(entry, launcher.CouchLaunchProfileEnv+"=")
		}
	}
	if profile == "" {
		t.Fatal("core omitted launch profile")
	}
	output := t.TempDir() + "/received"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "../launcher", "-run", "^TestCouchOrientationTransportHelper$", "-count=1", "-v")
	command.Env = append(os.Environ(), "PAIR184_ACCEPT_PROFILE="+profile, "PAIR184_ACCEPT_OUTPUT="+output)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("production transport failed: %v\n%s", err, out)
	}
	received, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Orientation.Body, paths.Log()) || !strings.Contains(result.Orientation.Body, native) {
		t.Fatal("orientation lost source paths")
	}
	for path, body := range map[string]string{paths.Log(): "operator source prompt\n", paths.Draft(): "unsent draft\n", native: "native conversation\n"} {
		after, err := os.ReadFile(path)
		if err != nil || string(after) != body {
			t.Fatalf("source artifact changed: %s", path)
		}
	}
	want := "\x1b[200~" + result.Orientation.Body + "\x1b[201~\r"
	if string(received) != want {
		t.Fatalf("delivery differs from actual core prompt: got %q want %q", received, want)
	}
}
