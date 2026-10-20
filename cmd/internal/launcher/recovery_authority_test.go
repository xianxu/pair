package launcher

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"strings"
	"testing"
	"time"
)

func testRecoveryAuthority() RecoveryAuthority {
	at := time.Unix(100, 0).UTC()
	id := artifactpath.CaptureFileIdentity{Size: 10, ModifiedAt: at, Device: 1, Inode: 2}
	c := threadrecord.RecoveryContext{Version: 1, Address: threadrecord.Address{RepoScope: "scope", Tag: "work"}, Capture: threadrecord.RecoveryCapture{Raw: "/capture/raw", Events: "/capture/events", RawIdentity: id, EventsIdentity: id, CapturedAt: at}, Generation: "generation", Digest: strings.Repeat("a", 64), CreatedAt: at, ExpiresAt: at.Add(time.Hour), Candidates: []threadrecord.RecoveryCandidate{{Key: "native-key", Agent: "codex", NativeID: "native", ArtifactRoot: "codex-sessions", ArtifactRelativePath: "session.jsonl", StableFileID: "file", MutationToken: "stamp", Size: 10, ModTime: at, LastActivity: at, Eligible: true}}}
	return RecoveryAuthority{Version: 1, Address: c.Address, Attempt: "attempt", RecordRevision: 5, RecordPath: "/store/thread.json", Intent: threadrecord.RecoveryIntent{Context: c, SelectedKey: "native-key", NativeID: "native"}}
}
func TestRecoveryLaunchProfileIsExplicitAndAttemptBound(t *testing.T) {
	for _, fresh := range []bool{false, true} {
		a := testRecoveryAuthority()
		if fresh {
			a.Intent.NativeID = ""
			a.Intent.SelectedKey = "fresh-key"
		}
		raw, err := BuildCouchRecoveryLaunchProfile("work", "codex", []string{"--no-alt-screen"}, a)
		if err != nil {
			t.Fatal(err)
		}
		args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, raw)
		if err != nil {
			t.Fatal(err)
		}
		if args.Recovery == nil || args.FreshRequired != fresh || args.ResumeRequired == fresh || args.RequiredSessionID != a.Intent.NativeID {
			t.Fatalf("wrong recovery launch flags: %+v", args)
		}
		p := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "codex", Argv: []string{}, Recovery: &a, FreshRequired: fresh, FreshNonce: args.FreshNonce, ResumeRequired: !fresh, RequiredSessionID: a.Intent.NativeID}
		if fresh {
			p.AgentSource, p.ArgvSource = "explicit", "explicit"
			p.FreshNonce = "wrong"
		} else {
			p.AgentSource, p.ArgvSource = "saved", "saved"
			p.RequiredSessionID = "wrong"
		}
		if ValidateTrustedLaunchProfile(p) == nil {
			t.Fatal("accepted recovery profile with a changed launch identity")
		}
	}
	ordinary := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "codex", Argv: []string{}, AgentSource: "saved", ArgvSource: "saved", ResumeRequired: true, RequiredSessionID: "native", Prompt: "context"}
	if ValidateTrustedLaunchProfile(ordinary) == nil {
		t.Fatal("ordinary resume gained a launch prompt")
	}
}

func recoveryAuthorityRecord(a RecoveryAuthority, cwd, agent string, pid int, identity string) threadrecord.Record {
	at := a.Intent.Context.CreatedAt
	if a.Intent.Kind() != threadrecord.RecoveryInferred {
		at = a.Intent.CreatedAt
	}
	return threadrecord.Record{SchemaVersion: threadrecord.SchemaVersion, Address: a.Address, StartingPath: cwd, WorkingPath: cwd, CreatedAt: at, Revision: a.RecordRevision + 2, Incarnations: []threadrecord.Incarnation{{State: "creating", PID: pid, Identity: identity, StartedAt: at, Start: &threadrecord.StartClaim{Nonce: a.Attempt, OwnerPID: 1, OwnerIdentity: "owner", RecoveryIntent: threadrecord.CloneRecoveryIntent(&a.Intent), LaunchProfile: &threadrecord.LaunchProfile{Agent: agent, Argv: []string{}}}}}}
}

func TestRecoveryRecordRequiresExactDurableIntentAndHelper(t *testing.T) {
	for name, change := range map[string]func(*threadrecord.Record){
		"none":               func(*threadrecord.Record) {},
		"revision":           func(r *threadrecord.Record) { r.Revision = 1 },
		"cwd":                func(r *threadrecord.Record) { r.WorkingPath = "/other" },
		"helper":             func(r *threadrecord.Record) { r.Incarnations[0].PID = 43 },
		"helper incarnation": func(r *threadrecord.Record) { r.Incarnations[0].Identity = "other" },
		"attempt":            func(r *threadrecord.Record) { r.Incarnations[0].Start.Nonce = "other" },
		"intent": func(r *threadrecord.Record) {
			r.Incarnations[0].Start.RecoveryIntent.Context.Digest = strings.Repeat("b", 64)
		},
		"committed": func(r *threadrecord.Record) { r.Incarnations[0].Start = nil },
		"agent":     func(r *threadrecord.Record) { r.Incarnations[0].Start.LaunchProfile.Agent = "claude" },
	} {
		t.Run(name, func(t *testing.T) {
			a := testRecoveryAuthority()
			a.Intent.Context.ExpiresAt = a.Intent.Context.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
			record := recoveryAuthorityRecord(a, "/repo", "codex", 42, "helper")
			change(&record)
			err := ValidateRecoveryRecord(a, record, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", a.Intent.Context.CreatedAt.Add(time.Minute))
			if (err == nil) != (name == "none") {
				t.Fatalf("validation=%v", err)
			}
		})
	}
}

type recoveryGateRuntime struct {
	*couchSessionRuntime
	record threadrecord.Record
	checks int
}

func (f *recoveryGateRuntime) ValidateRecoveryAuthority(ctx context.Context, a RecoveryAuthority, scope RepoScope, tag, agent, cwd string) error {
	f.checks++
	return ValidateRecoveryRecord(a, f.record, scope, tag, agent, cwd, 42, "helper", a.Intent.Context.CreatedAt.Add(time.Minute))
}
func TestRecoveryLaunchSecondGateRefusesChangedClaimBeforeEffects(t *testing.T) {
	for _, scenario := range []string{"capture", "native", "capture changed", "native changed"} {
		changed := strings.Contains(scenario, "changed")
		t.Run(scenario, func(t *testing.T) {
			a := testRecoveryAuthority()
			if strings.HasPrefix(scenario, "native") {
				a = nativeOnlyRecoveryAuthority()
			}
			opts := baseOpts(LaunchArgs{ForcedTag: "work"})
			scope, _ := ResolveRepoScope(opts.Env.Cwd)
			a.Address.RepoScope = scope.Key
			a.Intent.Context.Address = a.Address
			if a.Intent.Context.Version == 1 {
				a.Intent.Context.ExpiresAt = a.Intent.Context.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
			}
			raw, err := BuildCouchRecoveryLaunchProfile("work", "codex", []string{}, a)
			if err != nil {
				t.Fatal(err)
			}
			opts.Args, _, err = ApplyCouchLaunchProfile(opts.Args, raw)
			if err != nil {
				t.Fatal(err)
			}
			opts.Args.CouchSessionV1 = true
			opts.Args.CouchSession = &CouchSessionIntent{Scope: scope.Key, Tag: "work", Name: "📁1-3", Nonce: a.Attempt, Disposition: "create"}
			opts.Env.CouchThreadScope = scope.Key
			opts.Env.CouchThreadTag = "work"
			rt := &recoveryGateRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}, record: recoveryAuthorityRecord(a, opts.Env.Cwd, "codex", 42, "helper")}
			if changed {
				rt.record.Incarnations[0].Start.RecoveryIntent.Context.Digest = strings.Repeat("b", 64)
			}
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			if opts.Args.Prompt != "" {
				t.Fatal("explicit recovery injected prompt")
			}
			if rt.checks != 1 {
				t.Fatalf("gate calls %d: %s", rt.checks, stderr.String())
			}
			if changed {
				if code != 1 || rt.launchCount != 0 || len(rt.preparedLaunches) != 0 || len(rt.ledger["work"]) != 0 || rt.defaultReads != 0 {
					t.Fatalf("effects after refusal: code %d launch %d prepared %v", code, rt.launchCount, rt.preparedLaunches)
				}
			} else {
				if code != 0 || rt.launchCount != 1 || rt.env["PAIR_SESSION_ID"] != "native" || rt.env["PAIR_LAUNCH_NONCE"] != a.Attempt || launchArgsText(t, rt.env) != "--disable shell_snapshot resume native --no-alt-screen" {
					t.Fatalf("explicit exact resume failed: code %d launch %d args %q: %s", code, rt.launchCount, launchArgsText(t, rt.env), stderr.String())
				}
			}
		})
	}
}

func TestRecoveryLaunchRequiresTerminalAttemptAndValidator(t *testing.T) {
	for _, scenario := range []string{"missing terminal", "wrong terminal attempt", "missing validator"} {
		t.Run(scenario, func(t *testing.T) {
			opts := baseOpts(LaunchArgs{ForcedTag: "work"})
			scope, _ := ResolveRepoScope(opts.Env.Cwd)
			a := testRecoveryAuthority()
			a.Address.RepoScope = scope.Key
			a.Intent.Context.Address = a.Address
			raw, err := BuildCouchRecoveryLaunchProfile("work", "codex", []string{}, a)
			if err != nil {
				t.Fatal(err)
			}
			opts.Args, _, err = ApplyCouchLaunchProfile(opts.Args, raw)
			if err != nil {
				t.Fatal(err)
			}
			opts.Env.CouchThreadScope = scope.Key
			opts.Env.CouchThreadTag = "work"
			if scenario != "missing terminal" {
				opts.Args.CouchSessionV1 = true
				opts.Args.CouchSession = &CouchSessionIntent{Scope: scope.Key, Tag: "work", Name: "📁1-3", Nonce: a.Attempt, Disposition: "create"}
			}
			if scenario == "wrong terminal attempt" {
				opts.Args.CouchSession.Nonce = "other"
			}
			rt := newFakeRuntime()
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if err != nil || code != 1 || rt.launchCount != 0 || len(rt.preparedLaunches) != 0 || len(rt.ledger["work"]) != 0 {
				t.Fatalf("unsafe recovery effects: code=%d err=%v stderr=%s", code, err, stderr.String())
			}
			if scenario == "missing validator" && !strings.Contains(stderr.String(), "runtime lacks recovery authority validation") {
				t.Fatal(stderr.String())
			}
		})
	}
}

func taggedAuthority(mode threadrecord.RecoveryMode) RecoveryAuthority {
	a := testRecoveryAuthority()
	a.Intent = threadrecord.RecoveryIntent{Mode: mode, Address: a.Address, SourceRevision: 3, CreatedAt: time.Unix(100, 0).UTC(), Agent: "codex", SelectedKey: "explicit"}
	if mode == threadrecord.RecoveryProven {
		a.Intent.NativeID = "native"
	}
	return a
}

func TestTaggedRecoveryProfileAndClaimWithoutCapture(t *testing.T) {
	for _, mode := range []threadrecord.RecoveryMode{threadrecord.RecoveryProven, threadrecord.RecoveryFresh} {
		t.Run(string(mode), func(t *testing.T) {
			a := taggedAuthority(mode)
			raw, err := BuildCouchRecoveryLaunchProfile("work", "codex", []string{}, a)
			if err != nil {
				t.Fatal(err)
			}
			args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, raw)
			if err != nil {
				t.Fatal(err)
			}
			if args.Recovery.Intent.Kind() != mode {
				t.Fatal("tagged intent lost")
			}
			record := recoveryAuthorityRecord(a, "/repo", "codex", 42, "helper")
			if err := ValidateRecoveryRecord(a, record, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", a.Intent.CreatedAt.Add(time.Minute)); err != nil {
				t.Fatalf("requires absent capture: %v", err)
			}
			if _, err := BuildCouchRecoveryLaunchProfile("work", "claude", []string{}, a); err == nil {
				t.Fatal("profile accepted different agent")
			}
			record.Incarnations[0].Start.RecoveryIntent.SourceRevision++
			if err := ValidateRecoveryRecord(a, record, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", a.Intent.CreatedAt.Add(time.Minute)); err == nil {
				t.Fatal("accepted claim drift")
			}
		})
	}
}

func TestTaggedRecoveryClaimRejectsChangedAuthority(t *testing.T) {
	for name, change := range map[string]func(*RecoveryAuthority, *threadrecord.Record){
		"future source": func(a *RecoveryAuthority, r *threadrecord.Record) {
			a.Intent.SourceRevision = a.RecordRevision + 1
			r.Incarnations[0].Start.RecoveryIntent = threadrecord.CloneRecoveryIntent(&a.Intent)
		},
		"premature start": func(a *RecoveryAuthority, r *threadrecord.Record) {
			r.Incarnations[0].StartedAt = a.Intent.CreatedAt.Add(-time.Second)
		},
		"wrong agent": func(a *RecoveryAuthority, r *threadrecord.Record) {
			a.Intent.Agent = "claude"
			r.Incarnations[0].Start.RecoveryIntent = threadrecord.CloneRecoveryIntent(&a.Intent)
		},
		"changed nonce":  func(a *RecoveryAuthority, r *threadrecord.Record) { r.Incarnations[0].Start.Nonce = "changed" },
		"changed helper": func(a *RecoveryAuthority, r *threadrecord.Record) { r.Incarnations[0].Identity = "changed" },
	} {
		t.Run(name, func(t *testing.T) {
			a := taggedAuthority(threadrecord.RecoveryFresh)
			record := recoveryAuthorityRecord(a, "/repo", "codex", 42, "helper")
			change(&a, &record)
			if ValidateRecoveryRecord(a, record, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", a.Intent.CreatedAt.Add(time.Minute)) == nil {
				t.Fatal("accepted changed tagged authority")
			}
		})
	}
}

func nativeOnlyRecoveryAuthority() RecoveryAuthority {
	a := testRecoveryAuthority()
	c := &a.Intent.Context
	c.Version = 2
	c.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: c.CreatedAt, Revision: 3}
	c.Capture = threadrecord.RecoveryCapture{}
	c.Generation = ""
	c.ExpiresAt = c.CreatedAt.Add(threadrecord.NativeRecoverySelectionLifetime)
	c.Candidates[0].CWD = "/repo"
	a.Intent.SourceRevision = c.Source.Revision
	return a
}

func TestNativeOnlyRecoveryRecordAuthority(t *testing.T) {
	for _, scenario := range []string{"valid", "at cutoff", "at expiry", "before cutoff", "revision mismatch", "future revision", "changed cutoff", "renewed expiry", "copied address", "changed helper", "changed nonce", "changed claim", "changed cwd"} {
		t.Run(scenario, func(t *testing.T) {
			a := nativeOnlyRecoveryAuthority()
			now := a.Intent.Context.CreatedAt.Add(time.Minute)
			switch scenario {
			case "at cutoff":
				now = a.Intent.Context.CreatedAt
			case "at expiry":
				now = a.Intent.Context.ExpiresAt
			case "before cutoff":
				now = a.Intent.Context.CreatedAt.Add(-time.Nanosecond)
			case "revision mismatch":
				a.Intent.SourceRevision++
			case "future revision":
				a.Intent.SourceRevision = a.RecordRevision + 1
				a.Intent.Context.Source.Revision = a.Intent.SourceRevision
			case "changed cutoff":
				a.Intent.Context.Source.Cutoff = now
			case "renewed expiry":
				a.Intent.Context.ExpiresAt = a.Intent.Context.ExpiresAt.Add(time.Minute)
			}
			r := recoveryAuthorityRecord(a, "/repo", "codex", 42, "helper")
			switch scenario {
			case "copied address":
				r.Address.Tag = "other"
			case "changed helper":
				r.Incarnations[0].Identity = "other"
			case "changed nonce":
				r.Incarnations[0].Start.Nonce = "other"
			case "changed claim":
				r.Incarnations[0].Start.RecoveryIntent.Context.Digest = strings.Repeat("b", 64)
			case "changed cwd":
				r.WorkingPath = "/other"
			}
			err := ValidateRecoveryRecord(a, r, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", now)
			want := scenario == "valid" || scenario == "at cutoff"
			if (err == nil) != want {
				t.Fatalf("authority=%v, want allowed=%v", err, want)
			}
		})
	}
}
