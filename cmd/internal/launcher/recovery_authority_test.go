package launcher

import (
	"bytes"
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/orientation"
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
		p := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "codex", Argv: []string{}, AgentSource: "saved", ArgvSource: "saved", ResumeRequired: true, RequiredSessionID: "native", Recovery: &a, Orientation: &orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "codex", Attempt: "wrong", Body: "context"}}
		if ValidateTrustedLaunchProfile(p) == nil {
			t.Fatal("accepted mismatched orientation attempt")
		}
	}
	ordinary := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "codex", Argv: []string{}, AgentSource: "saved", ArgvSource: "saved", ResumeRequired: true, RequiredSessionID: "native", Orientation: &orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "codex", Attempt: "attempt", Body: "context"}}
	if ValidateTrustedLaunchProfile(ordinary) == nil {
		t.Fatal("ordinary resume gained orientation")
	}
}

func recoveryAuthorityRecord(a RecoveryAuthority, cwd, agent string, pid int, identity string) threadrecord.Record {
	return threadrecord.Record{SchemaVersion: threadrecord.SchemaVersion, Address: a.Address, StartingPath: cwd, WorkingPath: cwd, CreatedAt: a.Intent.Context.CreatedAt, Revision: a.RecordRevision + 2, Incarnations: []threadrecord.Incarnation{{State: "creating", PID: pid, Identity: identity, StartedAt: a.Intent.Context.CreatedAt, Start: &threadrecord.StartClaim{Nonce: a.Attempt, OwnerPID: 1, OwnerIdentity: "owner", RecoveryIntent: threadrecord.CloneRecoveryIntent(&a.Intent), LaunchProfile: &threadrecord.LaunchProfile{Agent: agent, Argv: []string{}}}}}}
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
			err := validateRecoveryRecord(a, record, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", a.Intent.Context.CreatedAt.Add(time.Minute))
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
	return validateRecoveryRecord(a, f.record, scope, tag, agent, cwd, 42, "helper", a.Intent.Context.CreatedAt.Add(time.Minute))
}
func TestRecoveryLaunchSecondGateRefusesChangedClaimBeforeEffects(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprint(changed), func(t *testing.T) {
			a := testRecoveryAuthority()
			opts := baseOpts(LaunchArgs{ForcedTag: "work"})
			scope, _ := ResolveRepoScope(opts.Env.Cwd)
			a.Address.RepoScope = scope.Key
			a.Intent.Context.Address = a.Address
			a.Intent.Context.ExpiresAt = a.Intent.Context.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
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
			opts.Args.Orientation = &orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "codex", Attempt: a.Attempt, Body: "recovery context"}
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
			if rt.checks != 1 {
				t.Fatalf("gate calls %d: %s", rt.checks, stderr.String())
			}
			if changed {
				if code != 1 || rt.launchCount != 0 || len(rt.preparedLaunches) != 0 || len(rt.ledger["work"]) != 0 || rt.defaultReads != 0 {
					t.Fatalf("effects after refusal: code %d launch %d prepared %v", code, rt.launchCount, rt.preparedLaunches)
				}
			} else {
				if code != 0 || rt.launchCount != 1 || rt.env["PAIR_SESSION_ID"] != "native" || rt.env["PAIR_LAUNCH_NONCE"] != a.Attempt || rt.env[orientation.Env] == "" {
					t.Fatalf("explicit exact resume failed: code %d launch %d: %s", code, rt.launchCount, stderr.String())
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
