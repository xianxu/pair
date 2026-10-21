package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// This equivalent in-process Pair boundary consumes the producer's actual
// child arguments/environment. Only external terminal observations are
// controlled. Address claims, persisted records and executing helper identity
// use the production OS validators. It deliberately stops before terminal
// creation; the independent native fleet qualification proves registration.
type recordlessBoundaryRuntime struct {
	launcher.Runtime
	os      launcher.OSRuntime
	scope   launcher.RepoScope
	tag     string
	reached bool
}

func (*recordlessBoundaryRuntime) StartProofMigration()                         {}
func (*recordlessBoundaryRuntime) SetEnv(string, string)                        {}
func (*recordlessBoundaryRuntime) InZellijPane() bool                           { return false }
func (*recordlessBoundaryRuntime) CommandExists(string) bool                    { return true }
func (*recordlessBoundaryRuntime) SessionLiveness() ([]launcher.Session, error) { return nil, nil }
func (r *recordlessBoundaryRuntime) SessionLivenessContext(context.Context) ([]launcher.Session, error) {
	if launcher.RegisterExistingCouchThread(r.os.GlobalDataDir, r.scope, r.tag) == nil {
		r.reached = true
		return nil, errors.New("controlled stop before external terminal handoff")
	}
	return nil, nil
}
func (*recordlessBoundaryRuntime) ReadSessionNameIndex() (launcher.SessionNameIndex, error) {
	return launcher.SessionNameIndex{}, nil
}
func (*recordlessBoundaryRuntime) SweepOrphanNvim([]string) {}
func (*recordlessBoundaryRuntime) ScanHistory(string, time.Time) ([]launcher.HistoricalTag, error) {
	return nil, nil
}
func (*recordlessBoundaryRuntime) ProbeSessionNameContext(context.Context, string) error { return nil }
func (r *recordlessBoundaryRuntime) EnsureThreadAddress(s launcher.RepoScope, t string, owned bool) error {
	return launcher.EnsureThreadAddressForPair(r.os.GlobalDataDir, s, t, owned)
}
func (r *recordlessBoundaryRuntime) RegisterExistingCouchThread(s launcher.RepoScope, t string) error {
	return launcher.RegisterExistingCouchThread(r.os.GlobalDataDir, s, t)
}
func (r *recordlessBoundaryRuntime) ValidateRecoveryAuthority(ctx context.Context, a launcher.RecoveryAuthority, s launcher.RepoScope, tag, agent, cwd string) error {
	return r.os.ValidateRecoveryAuthority(ctx, a, s, tag, agent, cwd)
}

type recordlessBoundaryArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	data string
}

func (a recordlessBoundaryArtifacts) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	return launcher.ClaimNewThreadAddress(a.data, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
}

// RunLaunch executes inside the acknowledged fake helper's caller. Reporting
// that actual caller as the helper makes OS validation check its real kernel
// identity; the fake still owns all signals and cannot signal this process.
type recordlessBoundaryRunner struct {
	*FakeRunner
	identity ProcessIdentity
}
type recordlessBoundaryHandle struct {
	BlockedHandle
	identity ProcessIdentity
}

func (h recordlessBoundaryHandle) PID() int         { return h.identity.PID }
func (h recordlessBoundaryHandle) Identity() string { return h.identity.Identity }
func (r recordlessBoundaryRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (BlockedHandle, error) {
	h, err := r.FakeRunner.StartBlocked(ctx, dir, argv, env, timeout)
	if err != nil {
		return nil, err
	}
	return recordlessBoundaryHandle{h, r.identity}, nil
}

func TestRecordlessFreshLaunchCrossesRealClaimBoundary(t *testing.T) {
	for _, scenario := range []string{"approved", "missing authority", "changed attempt", "changed helper", "wrong hosted address", "wrong hosted scope"} {
		t.Run(scenario, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			data := t.TempDir()
			env.Couch.Artifacts = recordlessBoundaryArtifacts{env.Artifacts, data}
			owner, err := (OSProcOps{}).Current()
			if err != nil {
				t.Fatal(err)
			}
			helper := owner
			if scenario == "changed helper" {
				helper.Identity += "-obsolete-generation"
			}
			env.Couch.Runner = recordlessBoundaryRunner{env.Runner, helper}
			env.Proc.Set(owner.PID, owner.Identity)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var runtime *recordlessBoundaryRuntime
			var before, after []byte
			var diagnostic bytes.Buffer
			var boundaryErr error
			env.Runner.AfterAcknowledge = func(id string) error {
				defer cancel()
				child := env.Runner.Child(id)
				args, err := launcher.ParseArgs(child.Argv[1:])
				if err != nil {
					boundaryErr = err
					return err
				}
				args, _, err = launcher.ApplyCouchLaunchProfile(args, environmentValue(child.Env, launcher.CouchLaunchProfileEnv))
				if err != nil {
					boundaryErr = err
					return err
				}
				// ParseArgs leaves the private terminal intent for process entry. Decode
				// the actual emitted payload without creating or changing authority.
				var intent launcher.CouchSessionIntent
				if err = json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchSessionIntentEnv)), &intent); err != nil {
					boundaryErr = err
					return err
				}
				args.CouchSession = &intent
				if scenario == "missing authority" {
					args.Recovery = nil
				}
				if scenario == "changed attempt" && args.Recovery != nil {
					copy := *args.Recovery
					copy.Attempt = "start-obsolete"
					args.Recovery = &copy
				}
				scope, err := launcher.ResolveRepoScope(child.Dir)
				if err != nil {
					boundaryErr = err
					return err
				}
				paths := launcher.NewScopedPaths(data, scope, args.ForcedTag)
				before, err = os.ReadFile(paths.ThreadClaim())
				if err != nil {
					boundaryErr = err
					return err
				}
				hostedScope, hostedTag := environmentValue(child.Env, "COUCH_THREAD_SCOPE"), environmentValue(child.Env, "COUCH_THREAD_TAG")
				if scenario == "wrong hosted address" {
					hostedTag = "another-tag"
				}
				if scenario == "wrong hosted scope" {
					hostedScope = "0123456789abcdef"
				}
				runtime = &recordlessBoundaryRuntime{os: launcher.OSRuntime{GlobalDataDir: data, DataDir: paths.ScopeDir()}, scope: scope, tag: args.ForcedTag}
				_, boundaryErr = launcher.RunLaunch(launcher.LaunchOptions{Args: args, Env: launcher.Env{Home: t.TempDir(), Cwd: child.Dir, RepoRoot: child.Dir, DataDir: paths.ScopeDir(), Now: time.Now(), CouchThreadScope: hostedScope, CouchThreadTag: hostedTag}, GlobalDataDir: data}, runtime, &diagnostic)
				after, err = os.ReadFile(paths.ThreadClaim())
				if err != nil {
					boundaryErr = err
				}
				return errors.New("boundary fixture stops before native terminal creation")
			}
			_, _ = env.Couch.startFreshRecoveryWithoutRecord(ctx, RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
			if boundaryErr != nil {
				t.Fatal(boundaryErr)
			}
			if runtime == nil {
				t.Fatalf("producer did not reach acknowledged helper: %s", &diagnostic)
			}
			if scenario == "approved" {
				if !runtime.reached || !strings.Contains(diagnostic.String(), "controlled stop before external terminal handoff") {
					t.Fatalf("approved producer could not establish its reserved address: %s", &diagnostic)
				}
				if bytes.Equal(before, after) {
					t.Fatal("reserved marker did not advance")
				}
			} else if runtime.reached || !bytes.Equal(before, after) {
				t.Fatalf("unproved launch advanced its marker: before=%s after=%s diagnostic=%s", before, after, &diagnostic)
			}
			if len(env.Runner.order) != 1 {
				t.Fatalf("launch count=%d", len(env.Runner.order))
			}
		})
	}
}
