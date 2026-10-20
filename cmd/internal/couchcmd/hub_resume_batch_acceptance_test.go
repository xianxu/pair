package couchcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Tier A for pair#726: the real CLI and broker resume three parked slots
// concurrently through the console queue, recovery, launch and adoption.
// Count admission's repository Discover separately from the fresh per-effect
// authority proofs: caching those proofs would make the count look better
// while weakening ownership checks.
func TestHubResumeBatchSocketAcceptance(t *testing.T) {
	for _, mode := range []string{"startup-prewarm", "cold", "cold-restart-held"} {
		t.Run(mode, func(t *testing.T) {
			prewarm, held := mode != "cold", mode == "cold-restart-held"
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			primary := filepath.Join(root, "repo")
			if err := os.MkdirAll(primary, 0700); err != nil {
				t.Fatal(err)
			}
			zero, address, branch := 0, "repo:0", "main"
			identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: root, EnvironmentRoot: root, WorktreeRoot: primary, Kind: "primary", Slot: &zero, Address: &address, RestingBranch: &branch}
			repository := couchcore.SlotRepository{Identity: identity}
			catalog := &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{primary: identity, ".": identity}, Repositories: map[string]couchcore.SlotRepository{}}
			paths := []string{primary}
			for n := 1; n <= 3; n++ {
				layout := couchcore.NewSlotLayout(primary, identity.RepoIdentity, n)
				if err := os.MkdirAll(layout.Host(), 0700); err != nil {
					t.Fatal(err)
				}
				slot := couchcore.SlotIdentity{Repo: "repo", RepoIdentity: identity.RepoIdentity, PrimaryRoot: primary, EnvironmentRoot: layout.Env(), WorktreeRoot: layout.Host(), Number: n}
				repository.Slots = append(repository.Slots, couchcore.SlotCandidate{Identity: slot, Verified: true})
				slotID := identity
				number, ref, rest := n, fmt.Sprintf("repo:%d", n), fmt.Sprintf("main-slot%d", n)
				slotID.Kind, slotID.Slot, slotID.Address, slotID.RestingBranch = "slot", &number, &ref, &rest
				slotID.EnvironmentRoot, slotID.WorktreeRoot = layout.Env(), layout.Host()
				slotID.EnvironmentHost = &couchcore.EnvironmentHost{Repo: "repo", Slot: n, RepoIdentity: identity.RepoIdentity, PrimaryRoot: primary, WorktreeRoot: layout.Host()}
				head := strings.Repeat("a", 40)
				slotID.Head = &head
				catalog.Workspaces[layout.Host()] = slotID
				paths = append(paths, layout.Host())
			}
			catalog.Repositories[primary] = repository
			rt := newRT(t, paths...)
			rt.runner = couchcore.NewFakeRunner()
			replies := map[couchcore.GitCall]string{}
			for _, path := range paths {
				replies[couchcore.GitCall{Dir: path, Args: "rev-parse --git-common-dir"}] = identity.RepoIdentity
				replies[couchcore.GitCall{Dir: path, Args: "rev-parse --show-toplevel"}] = path
			}
			rt.git = couchcore.NewFakeGit(replies)
			c, err := rt.NewCouch()
			if err != nil {
				t.Fatal(err)
			}
			c.Slots = &hubBatchCatalog{SlotCatalogFake: catalog}
			c.Workspaces = hubBatchReadiness{primary: primary}
			if err := c.Threads.EnrollSlotRepository(t.Context(), repository); err != nil {
				t.Fatal(err)
			}
			parked := make(map[string]couchcore.ThreadRecord)
			for _, path := range paths[1:] {
				record := seedHubBatchPark(t, c, path, identity.RepoIdentity)
				parked[path] = record
				rt.artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-"+record.Address.RepoScope)
				rt.artifacts.SetPairSession(record.Address, "pair-"+record.Address.RepoScope, false)
			}
			rt.runner.AfterAcknowledge = func(id string) error {
				record := parked[rt.runner.Child(id).Dir]
				rt.artifacts.SetPaneSidecar(record.Address, "claude")
				rt.artifacts.SetPairSession(record.Address, managedChildSession(t, rt.runner, id), true)
				rt.artifacts.SetSessionPresence(record.Address, couchcore.SessionObservation{State: couchcore.SessionPresent})
				return nil
			}
			host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})
			reader, input := io.Pipe()
			console := couchtty.New(host, reader)
			initial := ptychild.NewFakeChild(nil)
			console.Attach("initial", "initial", initial)
			t.Cleanup(func() { console.Stop(); _ = input.Close(); _ = reader.Close(); initial.Exit(0) })
			rt.runner.AfterBlockedStart = func(id string) {
				rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
			}
			ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{{Root: primary, Workspace: root, Slot: couchcore.HubSlot{Repository: primary, Number: 0}}})
			if err != nil {
				t.Fatal(err)
			}
			hubs := &hubRuntime{state: ownership}
			wireResolver(console, c, hubs)
			done := make(chan int, 1)
			go func() { done <- console.Run() }()
			t.Cleanup(func() {
				console.Stop()
				_ = input.Close()
				select {
				case <-done:
				case <-testwait.After(t):
					t.Error("console did not stop")
				}
			})

			// The held startup is released by observed wire responses, never by
			// a latency assumption: each CLI must first see its definite no-queue
			// response, then retry admission itself after the repository warms.
			releaseWarm := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseWarm) }) }
			t.Cleanup(release)
			var discovers, freshProofs, workspaceProofs atomic.Int64
			var trackWorkspace atomic.Bool
			r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}, slotOps: consoleSlotOperations(console, c, nil)}
			r.decorateAuthority = func(a messageAuthority) messageAuthority {
				workspace := a.workspace
				a.workspace = func(ctx context.Context, path string) (couchcore.WorkspaceIdentity, error) {
					if trackWorkspace.Load() {
						workspaceProofs.Add(1)
					}
					return workspace(ctx, path)
				}
				a.hubTarget = func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
					freshProofs.Add(1)
					return hubTargetResolver(c)(ctx, ref)
				}
				a.hubRepositoryNames = c.Threads.RepositoryNamesContext
				a.hubRepositorySlots = func(ctx context.Context, primary string) (map[int]couchcore.HubSlot, error) {
					discovers.Add(1)
					if held {
						select {
						case <-releaseWarm:
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}
					return c.RepositoryHubSlots(ctx, primary)
				}
				if prewarm {
					a.hubs = hubs
				}
				return a
			}
			r.init()
			if prewarm && !held {
				// Observe production startup, before any CLI request; explicitly
				// warming here would hide a missing startup Prewarm call.
				testwait.Until(t, "startup fills all three admission targets", func() bool {
					m := r.s.hubTargets
					m.mu.Lock()
					defer m.mu.Unlock()
					return len(m.entries) >= 3
				})
				if n := discovers.Load(); n != 1 {
					t.Fatalf("startup repository Discovers = %d, want 1 before the first CLI request", n)
				}
			} else if !held {
				if n := discovers.Load(); n != 0 {
					t.Fatalf("cold fixture already discovered %d repositories", n)
				}
				r.s.authority.hubs = hubs
			}
			caller := r.connect(0)
			r.world.set(caller, func(s *worldSlot) { s.root, s.workspace = primary, identity })
			r.s.mu.Lock()
			r.s.workspaces[caller] = identity
			r.s.mu.Unlock()
			env := testRT{env: map[string]string{"COUCH_STORE_DIR": "/couch/namespace", "COUCH_THREAD_SCOPE": caller.Scope, "COUCH_THREAD_TAG": caller.Tag, "PAIR_SESSION_NAME": caller.Session, "PAIR_LAUNCH_NONCE": caller.Nonce}}
			trackWorkspace.Store(true)
			warming := make(chan struct{}, 3)
			var warmingMu sync.Mutex
			seenWarming := make(map[string]bool)
			call := func(ctx context.Context, _ string, request, response any) error {
				err := couchmessage.Call(ctx, r.brokerSock, request, response)
				// Inspect only after Call has decoded the actual broker response.
				if err == nil && held {
					req := request.(couchmessage.Request)
					resp := response.(*couchmessage.Response)
					if req.Op == "resume" && resp.Code == couchmessage.CodeWarming {
						if resp.RetryAfterMillis <= 0 || resp.Operation != nil || !strings.Contains(resp.Error, "nothing was queued") {
							t.Errorf("warming must give a retry-after and no queued operation: %+v", resp)
						}
						warmingMu.Lock()
						if !seenWarming[req.Target] {
							seenWarming[req.Target] = true
							warming <- struct{}{}
						}
						warmingMu.Unlock()
					}
				}
				return err
			}
			type result struct {
				ref      string
				code     int
				out, err string
			}
			results := make(chan result, 3)
			start := make(chan struct{})
			for n := 1; n <= 3; n++ {
				go func(n int) {
					<-start
					ref := fmt.Sprintf("repo:%d", n)
					var out, errout bytes.Buffer
					code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume", ref: ref}, env, &out, &errout, call)
					results <- result{ref, code, out.String(), errout.String()}
				}(n)
			}
			close(start)
			if held {
				for n := 0; n < 3; n++ {
					select {
					case <-warming:
					case got := <-results:
						t.Fatalf("%s stopped before all three warming responses: exit %d, stdout %q, stderr %q", got.ref, got.code, got.out, got.err)
					case <-testwait.After(t):
						t.Fatal("cold supervisor did not return warming to all three CLI resumes")
					}
				}
				if n := workspaceProofs.Load(); n != 0 {
					t.Errorf("cold admission made %d synchronous workspace proofs, want zero", n)
				}
				if n := freshProofs.Load(); n != 0 {
					t.Errorf("cold admission made %d raw target resolutions, want zero", n)
				}
				r.slotOps.mu.Lock()
				heldReceipts := len(r.slotOps.receipts)
				r.slotOps.mu.Unlock()
				if heldReceipts != 0 {
					t.Errorf("warming refusals left %d queued receipts, want zero", heldReceipts)
				}
				release()
			}
			for n := 0; n < 3; n++ {
				select {
				case got := <-results:
					if got.code != 0 || !strings.Contains(got.out, got.ref+" resume succeeded") || (mode == "startup-prewarm" && strings.Contains(got.out+got.err, "warming")) || strings.Contains(got.out+got.err, "uncertain") {
						t.Errorf("%s exit %d, stdout %q, stderr %q", got.ref, got.code, got.out, got.err)
					}
					if held && (!strings.Contains(got.out+got.err, "waiting") || !strings.Contains(got.out+got.err, "warm")) {
						t.Errorf("%s did not explain waiting for warm-up: stdout %q, stderr %q", got.ref, got.out, got.err)
					}
				case <-testwait.After(t):
					t.Fatal("concurrent CLI resume batch dropped a result")
				}
			}
			if n := discovers.Load(); n != 1 {
				t.Errorf("admission repository Discovers = %d, want exactly one for the batch", n)
			}
			if n := freshProofs.Load(); n < 6 {
				t.Errorf("fresh authority proofs = %d, want prepare and effect proof for each resume", n)
			}
			for path, record := range parked {
				current, err := c.Threads.GetThread(record.Address)
				if err != nil {
					t.Fatal(err)
				}
				live := false
				for _, inc := range current.Incarnations {
					live = live || inc.State == couchcore.IncarnationLive
				}
				if !live {
					t.Errorf("%s remained parked after successful CLI receipt", path)
				}
			}
		})
	}
}

// The fixture's numbered directories already exist; provisioning external
// tools are outside this broker/admission acceptance boundary.
type hubBatchReadiness struct{ primary string }

// SlotCatalogFake records Calls in a slice. Protect that bookkeeping when
// the real console runs different slot jobs concurrently.
type hubBatchCatalog struct {
	*couchcore.SlotCatalogFake
	mu sync.Mutex
}

func (f *hubBatchCatalog) Discover(ctx context.Context, primary string) (couchcore.SlotRepository, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.SlotCatalogFake.Discover(ctx, primary)
}

func (f hubBatchReadiness) Ensure(ctx context.Context, req couchcore.ProvisionRequest) (couchcore.ProvisionResult, error) {
	if err := ctx.Err(); err != nil {
		return couchcore.ProvisionResult{}, err
	}
	if req.Path != f.primary || req.Slot < 1 || req.Slot > 3 {
		return couchcore.ProvisionResult{}, fmt.Errorf("unexpected batch readiness request: %+v", req)
	}
	return couchcore.ProvisionResult{SchemaVersion: 1, Address: fmt.Sprintf("repo:%d", req.Slot), Path: couchcore.NewSlotLayout(f.primary, "", req.Slot).Host(), RestingBranch: fmt.Sprintf("main-slot%d", req.Slot), Disposition: "ready"}, nil
}

func seedHubBatchPark(t *testing.T, c *couchcore.Couch, path, repoIdentity string) couchcore.ThreadRecord {
	t.Helper()
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	profile := &couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	record, err := c.Threads.CreateThread(couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0102030405060708"}, StartingPath: path, WorkingPath: path, CreatedAt: time.Unix(1, 0).UTC(), Revision: 1, LatestLaunchProfile: profile, Incarnations: []couchcore.ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: couchcore.IncarnationLive, RepoIdentity: repoIdentity, LaunchProfile: profile}}})
	if err != nil {
		t.Fatal(err)
	}
	park := couchcore.ParkIdentity{Nonce: "park-batch", Address: record.Address, PID: 42, ProcessIdentity: "pair-helper"}
	begun, err := c.Threads.BeginPark(record.Address, record.Revision, park)
	if err != nil {
		t.Fatal(err)
	}
	record, err = c.Threads.FinalizePark(t.Context(), record.Address, begun.Revision, park, 1, time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	return record
}
