package launcher

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

type recoveryFreshClaimRuntime struct {
	*recoveryGateRuntime
	global string
}

func (r *recoveryFreshClaimRuntime) ValidateRecoveryAuthority(ctx context.Context, a RecoveryAuthority, scope RepoScope, tag, agent, cwd string) error {
	at := a.Intent.CreatedAt
	if a.Intent.Kind() == threadrecord.RecoveryInferred {
		at = a.Intent.Context.CreatedAt
	}
	return validateRecoveryRecord(a, r.record, scope, tag, agent, cwd, 42, "helper", at.Add(time.Minute))
}

func (r *recoveryFreshClaimRuntime) EnsureThreadAddress(scope RepoScope, tag string, owned bool) error {
	return EnsureThreadAddressForPair(r.global, scope, tag, owned)
}

func (r *recoveryFreshClaimRuntime) RegisterExistingCouchThread(scope RepoScope, tag string) error {
	return RegisterExistingCouchThread(r.global, scope, tag)
}

func TestRecoveryFreshEstablishesOnlyAuthorizedReservedAddress(t *testing.T) {
	for _, scenario := range []string{"valid", "tagged valid", "changed attempt", "wrong owner", "missing marker", "resume", "retry after refusal"} {
		t.Run(scenario, func(t *testing.T) {
			a := testRecoveryAuthority()
			opts := baseOpts(LaunchArgs{ForcedTag: "work"})
			scope, err := ResolveRepoScope(opts.Env.Cwd)
			if err != nil {
				t.Fatal(err)
			}
			a.Address.RepoScope = scope.Key
			a.Intent.Context.Address = a.Address
			a.Intent.Context.ExpiresAt = a.Intent.Context.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
			if scenario != "resume" {
				a.Intent.NativeID, a.Intent.SelectedKey = "", "fresh-key"
			}
			if scenario == "tagged valid" {
				a.Intent = threadrecord.RecoveryIntent{Mode: threadrecord.RecoveryFresh, Address: a.Address, SourceRevision: 3, CreatedAt: time.Unix(100, 0).UTC(), Agent: "codex", SelectedKey: "fresh:old-tag:3"}
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
			opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, "work"
			if scenario == "wrong owner" {
				opts.Env.CouchThreadTag = "other"
			}
			rt := &recoveryFreshClaimRuntime{global: t.TempDir(), recoveryGateRuntime: &recoveryGateRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}, record: recoveryAuthorityRecord(a, opts.Env.Cwd, "codex", 42, "helper")}}
			if scenario == "changed attempt" || scenario == "retry after refusal" {
				rt.record.Incarnations[0].Start.Nonce = "other"
			}
			marker := NewScopedPaths(rt.global, scope, "work").ThreadClaim()
			var before []byte
			if scenario != "missing marker" {
				if _, err := ClaimNewThreadAddress(rt.global, scope, "work"); err != nil {
					t.Fatal(err)
				}
				before, err = os.ReadFile(marker)
				if err != nil {
					t.Fatal(err)
				}
			}
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "retry after refusal" {
				// pair#597: a refused Fresh leaves its reserved marker as it was,
				// so the next validated attempt on the same address starts.
				after, readErr := os.ReadFile(marker)
				if code != 1 || rt.launchCount != 0 || readErr != nil || !bytes.Equal(before, after) {
					t.Fatalf("first attempt: code=%d launches=%d marker=%s %v", code, rt.launchCount, after, readErr)
				}
				rt.record.Incarnations[0].Start.Nonce = a.Attempt
				stderr.Reset()
				if code, err = RunLaunch(opts, rt, &stderr); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "valid" || scenario == "tagged valid" || scenario == "retry after refusal" {
				if code != 0 || rt.launchCount != 1 {
					t.Fatalf("fresh recovery failed: code=%d launches=%d stderr=%s", code, rt.launchCount, &stderr)
				}
				if err := RegisterExistingCouchThread(rt.global, scope, "work"); err != nil {
					t.Fatalf("fresh registration not established: %v", err)
				}
				return
			}
			if code != 1 || rt.launchCount != 0 || len(rt.preparedLaunches) != 0 {
				t.Fatalf("unsafe launch: code=%d launches=%d stderr=%s", code, rt.launchCount, &stderr)
			}
			after, readErr := os.ReadFile(marker)
			if scenario == "missing marker" {
				if !os.IsNotExist(readErr) {
					t.Fatalf("missing marker was adopted: %s %v", after, readErr)
				}
			} else if readErr != nil || !bytes.Equal(before, after) {
				t.Fatalf("refused launch changed reserved marker: %s %v", after, readErr)
			}
		})
	}
}
