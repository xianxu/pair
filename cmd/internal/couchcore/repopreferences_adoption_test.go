package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func acceptedTrackedFixture(t *testing.T, remember bool) (*testEnv, trackedThreadLaunch) {
	t.Helper()
	env, in := recoveryTrackedFixture(t)
	in.Recovery, in.Fresh = false, false
	var err error
	in.Thread, err = env.Couch.Threads.updateExistingThread(in.Thread.Address, in.Thread.Revision, func(next *ThreadRecord) error {
		inc := &next.Incarnations[0]
		inc.RepoIdentity = "/repo/.git"
		inc.Start.RecoveryIntent = nil
		inc.Start.RememberConfiguration = remember
		inc.Start.LaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{"--model", "accepted"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	in.ProfileRaw, err = launcher.BuildCouchLaunchProfile(string(in.Thread.Address.Tag), "codex", []string{"--model", "accepted"}, "explicit", "explicit")
	if err != nil {
		t.Fatal(err)
	}
	return env, in
}

func TestRepoConfigurationAdoptionPublishesOnceAfterLaunch(t *testing.T) {
	env, in := acceptedTrackedFixture(t, true)
	record, handle, err := env.Couch.launchTrackedThread(in)
	if err != nil {
		t.Fatal(err)
	}
	result := StartResult{Record: record, Handle: handle}
	started, ok := result.Started()
	if !ok || started.Configuration == nil {
		t.Fatal("successful explicit launch lost acceptance payload")
	}
	if _, err := os.Lstat(env.Couch.Threads.repoPreferencesPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published before final adoption: %v", err)
	}
	var writes atomic.Int32
	env.Couch.Threads.hooks.AfterPublicationWrite = func(path string) error {
		if path == env.Couch.Threads.repoPreferencesPath() {
			writes.Add(1)
		}
		return nil
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func(copy StartResult) { defer wg.Done(); env.Couch.PublishAdoptedConfiguration(copy) }(result)
	}
	wg.Wait()
	doc, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	got := doc.Repositories["/repo/.git"]
	if writes.Load() != 1 || got.Revision != 1 || !got.Agents["codex"].Confirmed || !reflect.DeepEqual(got.Agents["codex"].Argv, []string{"--model", "accepted"}) {
		t.Fatalf("publication writes %d record %+v", writes.Load(), got)
	}
	if result.Warning() != "" || !handle.Alive() {
		t.Fatalf("adopted child damaged: warning %q alive %v", result.Warning(), handle.Alive())
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("configuration")) {
		t.Fatalf("ephemeral publication payload leaked into registry: %s", raw)
	}
}

func TestRepoConfigurationAdoptionFailureWarnsWithoutRollbackOrRetry(t *testing.T) {
	for _, failure := range []string{"malformed document", "publication failure"} {
		t.Run(failure, func(t *testing.T) {
			env, in := acceptedTrackedFixture(t, true)
			record, handle, err := env.Couch.launchTrackedThread(in)
			if err != nil {
				t.Fatal(err)
			}
			record.AttachmentNotice = "retained cleanup notice"
			result := StartResult{Record: record, Handle: handle}
			path := env.Couch.Threads.repoPreferencesPath()
			var attempts atomic.Int32
			bad := []byte("{malformed repository document\n")
			if failure == "malformed document" {
				if err := os.WriteFile(path, bad, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				env.Couch.Threads.hooks.AfterPublicationWrite = func(target string) error {
					if target == path {
						attempts.Add(1)
						return errors.New("injected settings publication failure")
					}
					return nil
				}
			}
			env.Couch.PublishAdoptedConfiguration(result)
			warning := result.Warning()
			if !strings.Contains(warning, "retained cleanup notice") || !strings.Contains(warning, "started, but startup parameters were not remembered:") {
				t.Fatalf("lost adoption warning: %q", warning)
			}
			env.Couch.PublishAdoptedConfiguration(result)
			if result.Warning() != warning || !handle.Alive() {
				t.Fatalf("retry/rollback changed outcome: warning %q alive %v", result.Warning(), handle.Alive())
			}
			if failure == "malformed document" {
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(after, bad) {
					t.Fatalf("failure changed old bytes: %q %v", after, err)
				}
			} else if attempts.Load() != 1 {
				t.Fatalf("publication retried %d times", attempts.Load())
			}
			current, err := env.Couch.Threads.GetThread(in.Thread.Address)
			if err != nil || len(current.Incarnations) != 1 || current.Incarnations[0].State != IncarnationLive {
				t.Fatalf("adopted registration rolled back: %+v %v", current, err)
			}
		})
	}
}

func TestRepoConfigurationAcceptanceIndependentOfLaunchShape(t *testing.T) {
	for _, mode := range []string{"automatic", "warm", "recovery", "claim recovery", "fresh", "resume", "ordinary"} {
		t.Run(mode, func(t *testing.T) {
			env, in := acceptedTrackedFixture(t, mode != "automatic")
			switch mode {
			case "warm":
				in.Warm, in.Resume = true, true
			case "recovery":
				in.Recovery = true
			case "claim recovery":
				in.Thread.Incarnations[0].Start.RecoveryIntent = taggedRecoveryIntent(in.Thread, "fresh")
			case "fresh":
				in.Fresh = true
			case "resume":
				in.Resume = true
			}
			config := acceptedConfigurationForStart(in.Thread, in)
			want := mode == "fresh" || mode == "resume" || mode == "ordinary"
			if (config != nil) != want {
				t.Fatalf("mode %s payload=%v, want %v", mode, config != nil, want)
			}
			result := StartResult{Configuration: config}
			if !want {
				env.Couch.PublishAdoptedConfiguration(result)
				if _, err := os.Lstat(env.Couch.Threads.repoPreferencesPath()); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("automatic path learned: %v", err)
				}
			}
		})
	}
}

func TestRepoConfigurationFailedLaunchHasNoAcceptancePayload(t *testing.T) {
	for _, phase := range []string{"spawn", "registry"} {
		t.Run(phase, func(t *testing.T) {
			env, in := acceptedTrackedFixture(t, true)
			if phase == "spawn" {
				env.Runner.FailNextStart(errors.New("injected spawn failure"))
			} else {
				if err := os.MkdirAll(filepath.Join(env.Couch.Store.Dir(), "registry.json"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			record, handle, err := env.Couch.launchTrackedThread(in)
			if err == nil {
				t.Fatal("failed launch returned success")
			}
			result, _ := (StartResult{Record: record, Handle: handle}).Started()
			if result.Configuration != nil {
				t.Fatal("failed launch retained acceptance authority")
			}
			env.Couch.PublishAdoptedConfiguration(result)
			if _, err := os.Lstat(env.Couch.Threads.repoPreferencesPath()); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed launch learned: %v", err)
			}
		})
	}
}

func TestRepoConfigurationAcceptanceClonesExactClaimAndRejectsOtherNonce(t *testing.T) {
	env, in := acceptedTrackedFixture(t, true)
	config := acceptedConfigurationForStart(in.Thread, in)
	if config == nil {
		t.Fatal("explicit accepted claim has no payload")
	}
	in.Thread.Incarnations[0].Start.LaunchProfile.Argv[1] = "mutated"
	env.Couch.PublishAdoptedConfiguration(StartResult{Configuration: config})
	doc, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Repositories["/repo/.git"].Agents["codex"].Argv; !reflect.DeepEqual(got, []string{"--model", "accepted"}) {
		t.Fatalf("payload aliases mutable claim: %v", got)
	}
	in.Nonce = "foreign-attempt"
	if acceptedConfigurationForStart(in.Thread, in) != nil {
		t.Fatal("another attempt acquired acceptance authority")
	}
}

func TestRepoConfigurationAutomaticLaunchDoesNotLearn(t *testing.T) {
	env, in := acceptedTrackedFixture(t, false)
	record, handle, err := env.Couch.launchTrackedThread(in)
	if err != nil {
		t.Fatal(err)
	}
	result, _ := (StartResult{Record: record, Handle: handle}).Started()
	if result.Configuration != nil {
		t.Fatal("automatic launch invented acceptance intent")
	}
	env.Couch.PublishAdoptedConfiguration(result)
	if _, err := os.Lstat(env.Couch.Threads.repoPreferencesPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("automatic launch learned: %v", err)
	}
	if !handle.Alive() {
		t.Fatal("automatic launch child stopped")
	}
}

func TestRepoConfigurationAcceptanceRejectsNativeSelectors(t *testing.T) {
	env, in := acceptedTrackedFixture(t, true)
	in.Thread.Incarnations[0].Start.LaunchProfile.Argv = []string{"resume", "private-native-id"}
	config := acceptedConfigurationForStart(in.Thread, in)
	if config == nil {
		t.Fatal("explicit intent lost before validation")
	}
	result := StartResult{Configuration: config}
	env.Couch.PublishAdoptedConfiguration(result)
	if !strings.Contains(result.Warning(), "not remembered") {
		t.Fatalf("selector refusal hidden: %q", result.Warning())
	}
	if _, err := os.Lstat(env.Couch.Threads.repoPreferencesPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("native selector remembered: %v", err)
	}
}

type adoptionDeadlineGit struct {
	GitRunner
	deadlines []time.Time
}

func (g *adoptionDeadlineGit) RunContext(ctx context.Context, _ string, _ ...string) (string, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return "", errors.New("repository query has no deadline")
	}
	g.deadlines = append(g.deadlines, deadline)
	if len(g.deadlines) == 1 {
		return "/repo/.git", nil
	}
	if deadline != g.deadlines[0] {
		return "", errors.New("repository preview renewed its deadline for another source")
	}
	<-ctx.Done()
	return "", ctx.Err()
}

func TestRepoConfigurationAdoptionSharesOneMigrationDeadline(t *testing.T) {
	env, in := acceptedTrackedFixture(t, true)
	record, handle, err := env.Couch.launchTrackedThread(in)
	if err != nil {
		t.Fatal(err)
	}
	result := StartResult{Record: record, Handle: handle}
	for _, path := range []string{"/repo/one", "/repo/two"} {
		legacyRepoWrite(t, env.Couch.Threads, "/repo/.git", path, []string{"--model", "legacy"})
	}
	git := &adoptionDeadlineGit{GitRunner: env.Couch.Git}
	env.Couch.Git = git
	env.Couch.PublishAdoptedConfiguration(result)
	if len(git.deadlines) != 2 || git.deadlines[0] != git.deadlines[1] {
		t.Fatalf("migration queries renewed the absolute deadline: %v", git.deadlines)
	}
	if !strings.Contains(result.Warning(), context.DeadlineExceeded.Error()) || !strings.Contains(result.Warning(), "not remembered") {
		t.Fatalf("migration cancellation warning = %q", result.Warning())
	}
	if !handle.Alive() {
		t.Fatal("cancelled preference preview stopped adopted child")
	}
	if _, err := os.Lstat(env.Couch.Threads.repoPreferencesPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled preview published settings: %v", err)
	}
}

func TestRepoConfigurationStartResultOmitsEphemeralCapabilityFromJSON(t *testing.T) {
	raw, err := json.Marshal(StartResult{Configuration: &AcceptedConfiguration{}})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for name := range fields {
		if strings.EqualFold(name, "configuration") {
			t.Fatalf("ephemeral acceptance capability serialized: %s", raw)
		}
	}
}
