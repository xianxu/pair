package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func recoveryPreflightFixture(t *testing.T) (*testEnv, ThreadRecord, *storagegc.Coordinator, artifactpath.StorageOwner) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	record := validThreadRecord(t)
	coordinator, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	coordinator.Now = func() time.Time { return env.Now }
	env.Couch.Threads.coordinator = coordinator
	owner, err := artifactpath.NewStorageOwner(coordinator.Root, record.Address.RepoScope, string(record.Address.Tag))
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Initialize(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	return env, record, coordinator, owner
}

func TestPrepareRecoveryLaunchProtectsSourceUntilFinish(t *testing.T) {
	for _, success := range []bool{false, true} {
		t.Run(map[bool]string{false: "failed", true: "successful"}[success], func(t *testing.T) {
			env, record, coordinator, owner := recoveryPreflightFixture(t)
			before, err := coordinator.ReadOwner(owner)
			if err != nil {
				t.Fatal(err)
			}
			now := env.Now.Add(time.Hour)
			coordinator.Now = func() time.Time { return now }
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repo, finish, err := env.Couch.prepareRecoveryLaunch(ctx, record, LaunchProfile{Agent: "codex", Argv: []string{}})
			if err != nil {
				t.Fatal(err)
			}
			if repo != "/repo/.git" || finish == nil {
				t.Fatalf("invalid preparation %q", repo)
			}
			held, err := coordinator.ReadOwner(owner)
			if err != nil {
				t.Fatal(err)
			}
			if len(held.Processes) != 1 || held.Processes[0].Role != "couch-resume" || !held.Activity.LastUse.Equal(before.Activity.LastUse) {
				t.Fatalf("preflight lost source retention: %+v", held)
			}
			cancel()
			if err := finish(success); err != nil {
				t.Fatal(err)
			}
			after, err := coordinator.ReadOwner(owner)
			if err != nil {
				t.Fatal(err)
			}
			expected := before.Activity.LastUse
			if success {
				expected = now
			}
			if len(after.Processes) != 0 || !after.Activity.LastUse.Equal(expected) {
				t.Fatalf("finish violated existing retention semantics: %+v", after)
			}
			if len(env.Runner.Ops) != 0 || len(env.Proc.Signals) != 0 || len(env.Artifacts.TriggeredQuits()) != 0 {
				t.Fatal("preflight launched or stopped source")
			}
		})
	}
}

func TestPrepareRecoveryLaunchRefusesBeforeRetirementOrRetention(t *testing.T) {
	for _, scenario := range []string{"working directory", "agent configuration", "resume arguments", "executable", "repository", "cancelled", "cancelled during check"} {
		t.Run(scenario, func(t *testing.T) {
			env, record, coordinator, owner := recoveryPreflightFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			profile := LaunchProfile{Agent: "codex", Argv: []string{}}
			switch scenario {
			case "working directory":
				env.Couch.Path.(*FakePathOps).Fail(record.WorkingPath)
			case "agent configuration":
				profile.Agent = "unsupported"
			case "resume arguments":
				profile.Argv = []string{"resume", "other-native"}
			case "executable":
				env.Couch.SwitchLaunchCheck = func(string) error { return errors.New("executable unavailable") }
			case "repository":
				env.Couch.Git = NewFakeGit(nil)
			case "cancelled":
				cancel()
			case "cancelled during check":
				env.Couch.SwitchLaunchCheck = func(string) error { cancel(); return nil }
			}
			before, err := coordinator.ReadOwner(owner)
			if err != nil {
				t.Fatal(err)
			}
			if _, finish, err := env.Couch.prepareRecoveryLaunch(ctx, record, profile); err == nil || finish != nil {
				t.Fatalf("invalid preflight was admitted: %v", err)
			}
			after, err := coordinator.ReadOwner(owner)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("failed preflight changed source retention")
			}
			if len(env.Runner.Ops) != 0 || len(env.Proc.Signals) != 0 || len(env.Artifacts.TriggeredQuits()) != 0 {
				t.Fatal("failed preflight launched or stopped source")
			}
		})
	}
}

func TestRecoveryLaunchProfileSavedPrecedesFreshDefaults(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := validThreadRecord(t)
	record.LatestLaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{"--no-alt-screen"}}
	env.Couch.RootAgent = "claude"
	for _, kind := range []RecoverySelectionKind{RecoveryFresh, RecoveryProven, RecoveryInferred} {
		profile, err := env.Couch.recoveryLaunchProfile(context.Background(), record, kind)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(profile, *record.LatestLaunchProfile) {
			t.Fatal("saved conversation profile replaced by defaults")
		}
		profile.Argv[0] = "changed"
		if record.LatestLaunchProfile.Argv[0] != "--no-alt-screen" {
			t.Fatal("profile clone aliases record")
		}
	}
}

func TestRecoveryLaunchProfileNativeRequiresSavedConfiguration(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := validThreadRecord(t)
	for _, kind := range []RecoverySelectionKind{RecoveryProven, RecoveryInferred} {
		if _, err := env.Couch.recoveryLaunchProfile(context.Background(), record, kind); err == nil {
			t.Fatal("native selection substituted defaults")
		}
	}
	if len(env.Git.Ops) != 0 {
		t.Fatal("native missing profile read defaults")
	}
}

func TestRecoveryFreshProfileUsesRepositoryArgumentsAndRootAgentReadOnly(t *testing.T) {
	for _, saved := range []bool{false, true} {
		t.Run(map[bool]string{false: "root-empty", true: "repository-configured"}[saved], func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			record := validThreadRecord(t)
			env.Couch.RootAgent = "claude"
			expected := LaunchProfile{Agent: "claude", Argv: []string{}}
			if saved {
				expected.Argv = []string{"--verbose"}
				for _, profile := range []LaunchProfile{expected, {Agent: "codex", Argv: []string{"--no-alt-screen"}}} {
					if err := env.Couch.Threads.rememberRepoConfiguration("/repo/.git", profile, nil); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, err := readRecoveryProfileFiles(env.Couch.Threads)
			if err != nil {
				t.Fatal(err)
			}
			profile, err := env.Couch.recoveryLaunchProfile(context.Background(), record, RecoveryFresh)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(profile, expected) {
				t.Fatalf("fresh profile %+v want %+v", profile, expected)
			}
			after, err := readRecoveryProfileFiles(env.Couch.Threads)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("profile preview mutated store")
			}
		})
	}
}

func readRecoveryProfileFiles(store *ThreadStore) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(store.root, func(path string, d os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(b)
		return nil
	})
	return files, err
}

func TestRecoveryFreshSlotProfileUsesRepositoryArgumentsReadOnly(t *testing.T) {
	local := testLocalThreadStore(t)
	env := newTestEnv(t, "/repo")
	env.Couch.Threads = local
	env.Couch.RootAgent = "codex"
	want := LaunchProfile{Agent: "codex", Argv: []string{"--model", "root-configuration"}}
	if err := local.rememberRepoConfiguration(local.slot.RepoIdentity, want, nil); err != nil {
		t.Fatal(err)
	}
	globalBefore, err := os.ReadFile(local.repoPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	record := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	if _, err := local.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	preference := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: local.slot.RepoIdentity, PhysicalPath: record.WorkingPath, LastAgent: "muse", ArgvByAgent: map[string][]string{"muse": {}}, Revision: 1}
	if err := writePathLaunchPreferenceForTest(local, preference); err != nil {
		t.Fatal(err)
	}
	before, err := readRecoveryProfileFiles(local)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := env.Couch.recoveryLaunchProfile(context.Background(), record, RecoveryFresh)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(profile, want) {
		t.Fatalf("repository/root profile = %+v, want %+v", profile, want)
	}
	after, err := readRecoveryProfileFiles(local)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("local profile preview mutated store")
	}
	globalAfter, err := os.ReadFile(local.repoPreferencesPath())
	if err != nil || string(globalBefore) != string(globalAfter) {
		t.Fatalf("repository profile preview mutated store: %v", err)
	}
}

func TestRecoveryFreshProfileDoesNotReplayPendingJournal(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := validThreadRecord(t)
	record, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	interrupted := errors.New("interrupted write")
	env.Couch.Threads.hooks.AfterJournal = func() error { return interrupted }
	_, err = env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Name = "pending"; return nil })
	if !errors.Is(err, interrupted) {
		t.Fatal(err)
	}
	before, err := os.ReadFile(env.Couch.Threads.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.recoveryLaunchProfile(context.Background(), record, RecoveryFresh); err == nil {
		t.Fatal("profile preview accepted pending journal")
	}
	after, err := os.ReadFile(env.Couch.Threads.journalPath())
	if err != nil || string(before) != string(after) {
		t.Fatalf("profile preview replayed pending journal: %v", err)
	}
}

func TestPrepareRecoveryLaunchUsesVerifiedPrimaryForBrokenNumberedHost(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	record := slotRecordFixture(t, env, local)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	fixture := env.Couch.Slots.(*OSSlotCatalog).IO.(*ProvisionFixture)
	admin := fixture.git(local.slot.WorktreeRoot, "rev-parse", "--absolute-git-dir")
	pointerPath := filepath.Join(local.slot.WorktreeRoot, ".git")
	pointer, err := os.ReadFile(pointerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(admin); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.resolveRepoIdentity(context.Background(), record.WorkingPath); err == nil {
		t.Fatal("fixture Git still resolves broken host")
	}
	repo, finish, err := env.Couch.prepareRecoveryLaunch(context.Background(), record, *record.LatestLaunchProfile)
	if err != nil {
		t.Fatalf("verified primary identity did not preflight broken host: %v", err)
	}
	if repo != local.slot.RepoIdentity || finish == nil {
		t.Fatalf("repository %q want %q", repo, local.slot.RepoIdentity)
	}
	if err := finish(false); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(pointerPath)
	if err != nil || string(after) != string(pointer) {
		t.Fatal("preflight rewrote broken host Git pointer")
	}
	if _, err := os.Stat(admin); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preflight repaired Git administrative directory: %v", err)
	}
	if len(env.Proc.Signals) != 0 || len(env.Runner.Ops) != 0 || len(env.Artifacts.TriggeredQuits()) != 0 {
		t.Fatal("preflight acted on live source")
	}
}

func TestPrepareRecoveryLaunchRejectsUnprovedNumberedFallback(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	source := slotRecordFixture(t, env, local)
	repository, err := env.Couch.Slots.Discover(context.Background(), local.slot.PrimaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"no catalog", "no candidate", "foreign candidate", "foreign primary", "foreign scope", "foreign recorded common", "escaped working path", "escaped starting path", "unverified primary", "catalog failure"} {
		t.Run(scenario, func(t *testing.T) {
			couch := &Couch{Threads: local, Path: OSPathOps{}, Git: NewFakeGit(nil)}
			record := cloneThreadRecord(source)
			repo := repository
			repo.Slots = append([]SlotCandidate(nil), repository.Slots...)
			catalog := &SlotCatalogFake{Repositories: map[string]SlotRepository{}, Errors: map[string]error{}}
			couch.Slots = catalog
			switch scenario {
			case "no catalog":
				couch.Slots = nil
			case "no candidate":
				repo.Slots = nil
			case "foreign candidate":
				repo.Slots[0].Identity.RepoIdentity = "/foreign/.git"
			case "foreign primary":
				repo.Identity.RepoIdentity = "/foreign/.git"
			case "foreign scope":
				record.Address.RepoScope = "ffffffffffffffff"
			case "foreign recorded common":
				record.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown, PID: 42, Identity: "foreign", RepoIdentity: "/foreign/.git"}}
			case "escaped working path":
				record.WorkingPath = t.TempDir()
			case "escaped starting path":
				record.StartingPath = t.TempDir()
			case "unverified primary":
				repo.Identity.Kind = "slot"
			case "catalog failure":
				catalog.Errors[local.slot.PrimaryRoot] = errors.New("primary observation unavailable")
			}
			catalog.Repositories[local.slot.PrimaryRoot] = repo
			before, err := readRecoveryProfileFiles(local)
			if err != nil {
				t.Fatal(err)
			}
			if _, finish, err := couch.prepareRecoveryLaunch(context.Background(), record, *source.LatestLaunchProfile); err == nil || finish != nil {
				t.Fatalf("unproved repository fallback admitted: %v", err)
			}
			after, err := readRecoveryProfileFiles(local)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("failed fallback changed source store")
			}
			if len(env.Proc.Signals) != 0 || len(env.Runner.Ops) != 0 {
				t.Fatal("failed fallback acted on source")
			}
		})
	}
}
