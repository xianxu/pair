package couchcore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type editedRestartPublicationFixture struct {
	env    *testEnv
	result SwitchAgentResult
	before []byte
	writes atomic.Int32
}

func startEditedRestartPublicationFixture(t *testing.T, agent string, missing bool, argv []string) *editedRestartPublicationFixture {
	t.Helper()
	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{"--model", "historical"}})
	for _, savedAgent := range []string{"claude", "codex"} {
		if err := env.Couch.Threads.rememberRepoConfiguration("/repo/.git", LaunchProfile{Agent: savedAgent, Argv: []string{"--model", "previous-" + savedAgent}}, nil); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(env.Couch.Threads.repoPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, bindID(3))
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
	if err != nil {
		t.Fatal(err)
	}
	accepted := prepared.AcceptAgentSwitch(false)
	if missing {
		env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingUnbound, "")
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		env.Artifacts.SetPaneSidecar(source.Address, agent)
		return nil
	}
	f := &editedRestartPublicationFixture{env: env, before: before}
	env.Couch.Threads.hooks.AfterPublicationWrite = func(path string) error {
		if path == env.Couch.Threads.repoPreferencesPath() {
			f.writes.Add(1)
		}
		return nil
	}
	f.result, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Restart: accepted.restart, Address: source.Address, Agent: agent, Argv: argv, AcceptedFingerprint: prepared.Fingerprint})
	if err != nil || f.result.Outcome != SwitchStarted {
		t.Fatalf("actual edited restart: %+v %v", f.result, err)
	}
	t.Cleanup(func() { env.Runner.SetExited(f.result.Handle.ID(), 0) })
	wantShape := StartColdResume
	if missing {
		wantShape = StartFreshExisting
	}
	if f.result.Record.Shape != wantShape || strings.Contains(f.result.Warning(), EditedRestartFallbackNotice) != missing {
		t.Fatalf("wrong restart outcome: shape=%s warning=%q", f.result.Record.Shape, f.result.Warning())
	}
	f.assertUnpublished(t)
	return f
}

func (f *editedRestartPublicationFixture) assertUnpublished(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile(f.env.Couch.Threads.repoPreferencesPath())
	if err != nil || !bytes.Equal(raw, f.before) || f.writes.Load() != 0 {
		t.Fatalf("restart learned before final adoption: writes=%d bytes=%q err=%v", f.writes.Load(), raw, err)
	}
}

// Console's final attach dispatcher owns the publication call. These core
// tests exercise its actual accepted result and abort boundaries; console and
// remote-receipt transport qualification lives in couchcmd (avoiding a cycle).
func TestEditedRestartPublicationAfterAdoption(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, mode := range []string{"resume", "fresh"} {
			for _, parameters := range []string{"empty", "literal"} {
				t.Run(agent+"/"+mode+"/"+parameters, func(t *testing.T) {
					argv := []string{}
					if parameters == "literal" {
						argv = []string{"--model", "a b", "", "$HOME", "$(literal)"}
					}
					f := startEditedRestartPublicationFixture(t, agent, mode == "fresh", argv)
					start, ok := f.result.Started()
					if !ok || start.Configuration == nil {
						t.Fatal("edited restart lost explicit acceptance across StartedChild")
					}
					f.env.Couch.PublishAdoptedConfiguration(start)
					// Result reconstruction must share once-only publication state.
					copy, _ := (StartResult{Record: f.result.Record, Handle: f.result.Handle}).Started()
					f.env.Couch.PublishAdoptedConfiguration(copy)
					doc, err := f.env.Couch.Threads.previewRepoPreferences()
					if err != nil {
						t.Fatal(err)
					}
					preference := doc.Repositories["/repo/.git"]
					got := preference.Agents[agent]
					if f.writes.Load() != 1 || preference.Revision != 3 || !got.Confirmed || !reflect.DeepEqual(got.Argv, argv) {
						t.Fatalf("accepted argv not published once: writes=%d preference=%+v", f.writes.Load(), preference)
					}
					other := "claude"
					if agent == other {
						other = "codex"
					}
					if !reflect.DeepEqual(preference.Agents[other].Argv, []string{"--model", "previous-" + other}) || !f.result.Handle.Alive() {
						t.Fatal("publication damaged other-agent defaults or adopted child")
					}
				})
			}
		}
	}
}

func TestEditedRestartFailedAdoptionDoesNotPublish(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, mode := range []string{"resume", "fresh"} {
			t.Run(agent+"/"+mode, func(t *testing.T) {
				f := startEditedRestartPublicationFixture(t, agent, mode == "fresh", []string{"--model", "rejected edit"})
				start, ok := f.result.Started()
				if !ok {
					t.Fatal("missing started child")
				}
				cause := errors.New("final console attach rejected")
				if err := f.env.Couch.AbortStarted(context.Background(), start, cause); !errors.Is(err, cause) {
					t.Fatalf("failed-adoption cleanup lost original cause: %v", err)
				}
				if f.result.Handle.Alive() {
					t.Fatal("failed adoption retained the exact started helper")
				}
				f.assertUnpublished(t)
			})
		}
	}
}

func TestEditedRestartFallbackCombinesPreferenceWarning(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			f := startEditedRestartPublicationFixture(t, agent, true, []string{"--model", "accepted"})
			malformed := []byte("{broken repository preferences\n")
			if err := os.WriteFile(f.env.Couch.Threads.repoPreferencesPath(), malformed, 0600); err != nil {
				t.Fatal(err)
			}
			start, _ := f.result.Started()
			f.env.Couch.PublishAdoptedConfiguration(start)
			warning := f.result.Warning()
			if !strings.Contains(warning, EditedRestartFallbackNotice) || !strings.Contains(warning, "started, but startup parameters were not remembered:") {
				t.Fatalf("fallback/publication warning lost: %q", warning)
			}
			f.env.Couch.PublishAdoptedConfiguration(start)
			raw, err := os.ReadFile(f.env.Couch.Threads.repoPreferencesPath())
			if err != nil || !bytes.Equal(raw, malformed) || f.writes.Load() != 0 || f.result.Warning() != warning || !f.result.Handle.Alive() {
				t.Fatalf("warning retried publication or damaged child/old bytes: writes=%d alive=%t err=%v", f.writes.Load(), f.result.Handle.Alive(), err)
			}
		})
	}
}
