package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The nested launcher helper avoids a couchcore -> launcher -> couchcore test
// import cycle. Its only input is the actual env value produced by SwitchAgent.
func TestSwitchAgentProducesOrientationThroughLayoutAndWrapper(t *testing.T) {
	for _, live := range []bool{false, true} {
		t.Run(fmt.Sprintf("live=%t", live), func(t *testing.T) { runSwitchAgentTransportAcceptance(t, live) })
	}
}

func runSwitchAgentTransportAcceptance(t *testing.T, liveSource bool) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	var source couchcore.ThreadRecord
	var sourceHandle couchcore.Handle
	if !liveSource {
		source = seedVerifiedPark(t, rt, "/repo")
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	c.FreshRegistration = func(context.Context, couchcore.ThreadAddress, string, string) (bool, error) { return true, nil }
	if liveSource {
		c.RootAgent = "claude"
		actor, handle, err := c.Spawn(couchcore.StartArgs{Worktree: "/repo"})
		if err != nil {
			t.Fatal(err)
		}
		sourceHandle = handle
		rt.proc.Set(handle.PID(), handle.Identity())
		source, err = c.Threads.GetThread(actor.Thread)
		if err != nil {
			t.Fatal(err)
		}
		rt.artifacts.SetPairSession(source.Address, "pair-live-source", true)
	}

	data := t.TempDir()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: source.Address.RepoScope, Tag: string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(data, "outgoing-native.jsonl")
	queue := filepath.Join(paths.QueueDir(), "000001.md")
	if err := os.MkdirAll(filepath.Dir(queue), 0700); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{paths.Log(): "operator source prompt\n", paths.Draft(): "unsent draft\n", queue: "queued prompt\n", native: "native conversation\n"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	renderer := filepath.Join(data, "pair-renderer")
	if out, err := exec.Command("go", "build", "-o", renderer, "../../pair-go").CombinedOutput(); err != nil {
		t.Fatalf("build renderer %v: %s", err, out)
	}
	c.SwitchContext = couchcore.OSSwitchContextResolver{DataDir: data, HomeDir: data, Renderer: renderer, Query: func(context.Context, sessioninventory.Runtime, string, string, sessioninventory.Agent) (sessioninventory.SessionQuery, error) {
		return sessioninventory.SessionQuery{Status: sessioninventory.BindingEstablished, Root: &sessioninventory.Node{NativeID: "outgoing-root", Artifacts: []sessioninventory.Artifact{{Kind: sessioninventory.ArtifactTranscript}}}}, nil
	}, NativePath: func(sessioninventory.Runtime, sessioninventory.Artifact) (string, error) { return native, nil }}

	var captured *pairlifecycle.PreservedScrollback
	if liveSource {
		live, _ := paths.ScrollbackArtifacts("claude")
		for path, body := range map[string]string{live.Raw: "outgoing live Pair transcript\n", live.Events: "{\"offset\":0}\n"} {
			if err := os.WriteFile(path, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
		c.PairLifecycle = &couchcore.PairLifecycleController{Threads: c.Threads, DataDir: data, Lifecycle: couchcore.PairLifecycleStoreIO{Store: pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}}, Sessions: rt.artifacts, Proc: rt.proc, Clock: couchcore.FixedClock{T: time.Now().UTC()}, Nonce: func() (string, error) { return "live-acceptance-park", nil }}
		rt.artifacts.TriggerQuitHook = func(session string, intent launcher.QuitIntent) error {
			if session != "pair-live-source" || intent.Request == nil || intent.Request.Tag != string(source.Address.Tag) || intent.Request.RepoScope != source.Address.RepoScope {
				return fmt.Errorf("quit targeted another source: %s %+v", session, intent)
			}
			if !sourceHandle.Alive() || rt.proc.Exists(sourceHandle.PID()) != couchcore.Live {
				return fmt.Errorf("source was not alive at quit trigger")
			}
			encoded, err := json.Marshal(intent)
			if err != nil {
				return err
			}
			output := filepath.Join(data, "cleanup-result.json")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "go", "test", "../launcher", "-run", "^TestCouchLiveCleanupHelper$", "-count=1", "-v")
			command.Env = append(os.Environ(), "PAIR184_CLEANUP_INTENT="+string(encoded), "PAIR184_CLEANUP_SESSION="+session, "PAIR184_CLEANUP_OUTPUT="+output)
			if out, err := command.CombinedOutput(); err != nil {
				return fmt.Errorf("live cleanup: %w: %s", err, out)
			}
			raw, err := os.ReadFile(output)
			if err != nil {
				return err
			}
			var result pairlifecycle.CleanupResult
			if err := json.Unmarshal(raw, &result); err != nil {
				return err
			}
			if result.Outcome != pairlifecycle.CompletionSuccess || result.Scrollback == nil {
				return fmt.Errorf("cleanup omitted actual capture: %+v", result)
			}
			captured = pairlifecycle.ClonePreservedScrollback(result.Scrollback)
			rt.artifacts.SetPairSession(source.Address, session, false)
			rt.runner.SetExited(sourceHandle.ID(), 0)
			rt.proc.Kill(sourceHandle.PID())
			return nil
		}
		rt.runner.AfterBlockedStart = func(string) {
			parked, err := c.Threads.GetThread(source.Address)
			if err != nil || captured == nil || parked.VerifiedPark == nil || parked.VerifiedPark.Identity.PID != sourceHandle.PID() || parked.VerifiedPark.Identity.ProcessIdentity != sourceHandle.Identity() || !reflect.DeepEqual(parked.VerifiedPark.Scrollback, captured) || parked.Park != nil || sourceHandle.Alive() || rt.proc.Exists(sourceHandle.PID()) != couchcore.Dead {
				t.Errorf("target started before exact live source park completed: %+v %v", parked, err)
			}
		}
	}
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
	command.Env = append(os.Environ(), "PAIR184_ACCEPT_PROFILE="+profile, "PAIR184_ACCEPT_OUTPUT="+output, "PAIR184_ACCEPT_BINARY="+renderer)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("production transport failed: %v\n%s", err, out)
	}
	received, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if liveSource {
		if captured == nil {
			t.Fatal("live source never produced archive descriptor")
		}
		archived, err := paths.ParkedScrollbackArtifacts(captured.Token)
		if err != nil {
			t.Fatal(err)
		}
		if captured.Agent != "claude" || !captured.Events || !strings.Contains(result.Orientation.Body, archived.Raw) || !strings.Contains(result.Orientation.Body, archived.Events) {
			t.Fatalf("actual cleanup descriptor never reached prompt: %+v %s", captured, result.Orientation.Body)
		}
		for path, body := range map[string]string{archived.Raw: "outgoing live Pair transcript\n", archived.Events: "{\"offset\":0}\n"} {
			raw, err := os.ReadFile(path)
			if err != nil || string(raw) != body {
				t.Fatalf("archive lost source bytes: %s %v", path, err)
			}
		}
	}
	if !strings.Contains(result.Orientation.Body, paths.Log()) || !strings.Contains(result.Orientation.Body, native) {
		t.Fatal("orientation lost source paths")
	}
	for path, body := range map[string]string{paths.Log(): "operator source prompt\n", paths.Draft(): "unsent draft\n", queue: "queued prompt\n", native: "native conversation\n"} {
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
