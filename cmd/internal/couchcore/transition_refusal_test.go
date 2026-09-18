package couchcore

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// Every named lifecycle transition is driven into its OWN refusal, and a refusal
// writes nothing.
//
// Family `fail-closed-guard-untested`, fifth occurrence (#256 M3 BR, I1). Each
// earlier round added the missing test for the guard it had just been shown;
// M3 then added five transitions whose preconditions nothing entered, and
// deleting all three of the new ones left the package green. The rule:
//
//	A transition introduced so a precondition has somewhere to live is not
//	delivered until a test drives it into that refusal -- and the table's
//	domain is DERIVED from the transition set, not hand-listed.
//
// So the domain is every exported *ThreadStore method that reaches the mutator
// door, computed from the AST the same way TestArbitraryLifecycleMutationHasNoDoor
// finds the door itself. A new transition fails here until it is given a refusal
// case, or is declared to have no precondition beyond the revision CAS -- and
// that declaration has to say why, because "it has none" is a claim too.
//
// "Writes nothing" is asserted on every case because it is what fail-CLOSED
// means: a refusal that advanced the revision, or wrote half a record, would
// pass a bare `err != nil` and still be the bug.
func TestEveryLifecycleTransitionIsDrivenIntoItsOwnRefusal(t *testing.T) {
	domain := lifecycleTransitionsReachingTheDoor(t)

	type refusal struct {
		// shape puts the record into the state the precondition exists to
		// refuse. It may use the door directly: this is a fixture, and the
		// shape is often one production can no longer write.
		shape func(t *testing.T, store *ThreadStore, record ThreadRecord) ThreadRecord
		call  func(t *testing.T, store *ThreadStore, record ThreadRecord) error
		// want is a substring of the transition's own refusal, never the
		// revision CAS's -- DISCRIMINATING, or a stale revision would pass it.
		want string
	}
	noShape := func(_ *testing.T, _ *ThreadStore, record ThreadRecord) ThreadRecord { return record }
	withIncarnation := func(state IncarnationState) func(*testing.T, *ThreadStore, ThreadRecord) ThreadRecord {
		return func(t *testing.T, store *ThreadStore, record ThreadRecord) ThreadRecord {
			t.Helper()
			next, err := store.updateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error {
				r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-x", State: state}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			return next
		}
	}
	withStartClaim := func(t *testing.T, store *ThreadStore, record ThreadRecord) ThreadRecord {
		t.Helper()
		next, err := store.CommitStartClaim(record.Address, record.Revision, "repo", time.Unix(200, 0).UTC(), StartEvent{
			Kind: StartClaimed, Nonce: "start-0123456789abcdef", Shape: StartFreshExisting,
			Owner: SupervisorOwner{PID: 77, Identity: "owner-couch"}, Profile: record.LatestLaunchProfile,
		})
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	withPendingContinuation := func(t *testing.T, store *ThreadStore, record ThreadRecord) ThreadRecord {
		t.Helper()
		next, err := store.PublishContinuation(record.Address, record.Revision, testContinuationRequest(t, record))
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	park := func(record ThreadRecord) ParkIdentity {
		return ParkIdentity{Nonce: "park-0123456789abcdef", Address: record.Address, PID: 42, ProcessIdentity: "pair-x"}
	}

	refusals := map[string]refusal{
		"PublishContinuation": {
			shape: withPendingContinuation,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				cp, err := checkpoint.New("/tmp/other.md", "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nanother\n")
				if err != nil {
					return err
				}
				other := testContinuationRequest(t, r)
				other.Checkpoint = cp
				other.ID = checkpoint.RequestID(r.Address.RepoScope, string(r.Address.Tag), 2, cp.Digest)
				_, err = s.PublishContinuation(r.Address, r.Revision, other)
				return err
			},
			want: "before publishing another",
		},
		"DismissFailedContinuation": {
			shape: withPendingContinuation,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.DismissFailedContinuation(r.Address, r.Revision, r.Continuation.ID)
				return err
			},
			want: "only a failed continuation can be dismissed",
		},
		"BeginContinuationFromRetiredIncarnations": {
			shape: withStartClaim,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.BeginContinuationFromRetiredIncarnations(r.Address, r.Revision, testContinuationRequest(t, r))
				return err
			},
			want: "open start claim",
		},
		"AdvanceContinuation": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.AdvanceContinuation(r.Address, r.Revision, checkpoint.Event{Kind: checkpoint.Begin})
				return err
			},
			want: "no continuation request",
		},
		"BeginPark": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.BeginPark(r.Address, r.Revision, park(r))
				return err
			},
			want: "exactly one live or unknown incarnation",
		},
		"AdvancePark": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.AdvancePark(r.Address, r.Revision, ParkEvent{Kind: ParkRequestCommitted, Identity: park(r)})
				return err
			},
			want: "no active park transaction",
		},
		"AppendParkAttempt": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.AppendParkAttempt(r.Address, r.Revision, park(r))
				return err
			},
			want: "park attempt identity does not match",
		},
		"FinalizePark": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.FinalizePark(r.Address, r.Revision, park(r), 1, time.Unix(300, 0).UTC())
				return err
			},
			want: "does not match an active non-tombstoned transaction",
		},
		"AbandonPark": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.AbandonPark(r.Address, r.Revision, park(r))
				return err
			},
			want: "park abandon identity does not match",
		},
		"CommitStartClaim": {
			shape: withIncarnation(IncarnationLive),
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.CommitStartClaim(r.Address, r.Revision, "repo", time.Unix(200, 0).UTC(), StartEvent{
					Kind: StartClaimed, Nonce: "start-0123456789abcdef", Shape: StartFreshExisting,
				})
				return err
			},
			want: "already has 1 incarnation",
		},
		"AdvanceStart": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.AdvanceStart(r.Address, r.Revision, StartEvent{
					Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
					Helper: ProcessIdentity{PID: 42, Identity: "pair-x"},
				})
				return err
			},
			want: "not found",
		},
		"DeleteStart": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				return s.DeleteStart(r.Address, r.Revision, "start-0123456789abcdef")
			},
			want: "is no longer start",
		},
		"RetireIncarnation": {
			shape: withIncarnation(IncarnationUnknown),
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.RetireIncarnation(r.Address, r.Revision, ProcessIdentity{PID: 42, Identity: "pair-x"}, time.Unix(300, 0).UTC())
				return err
			},
			want: "needs a live incarnation",
		},
		"RetireUnprovenIncarnation": {
			shape: withIncarnation(IncarnationLive),
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.RetireUnprovenIncarnation(r.Address, r.Revision, ProcessIdentity{PID: 42, Identity: "pair-x"}, time.Unix(300, 0).UTC())
				return err
			},
			want: "needs a unknown incarnation",
		},
		"RetireProvedDeadIncarnations": {
			shape: withStartClaim,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.RetireProvedDeadIncarnations(r.Address, r.Revision)
				return err
			},
			want: "open start claim",
		},
		"ClearVerifiedPark": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.ClearVerifiedPark(r.Address, r.Revision)
				return err
			},
			want: "no verified park to clear",
		},
		"MarkIncarnationUnknown": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.MarkIncarnationUnknown(r.Address, ProcessIdentity{PID: 42, Identity: "pair-x"})
				return err
			},
			want: "not found in thread",
		},
		"ReconcileRegisteredTarget": {
			shape: noShape,
			call: func(t *testing.T, s *ThreadStore, r ThreadRecord) error {
				_, err := s.ReconcileRegisteredTarget(r.Address, r.Revision, RegisteredTargetProof{})
				return err
			},
			want: "does not match the current continuation",
		},
	}
	// Transitions with no precondition of their own. Each says WHY, because an
	// empty entry would be exactly the silent omission this table exists to end.
	casOnly := map[string]string{
		"ApplyThreadMetadata": "names, descriptions and summaries are operator text with no lifecycle meaning; " +
			"the pure transition is ApplyThreadMetadata in threadmetadata_model.go and the revision CAS is its only refusal",
	}

	for _, name := range domain {
		_, refused := refusals[name]
		_, exempt := casOnly[name]
		if refused == exempt {
			t.Errorf("transition %s reaches the mutator door and has %s; give it a refusal case, "+
				"or declare why it has no precondition beyond the revision CAS",
				name, map[bool]string{true: "BOTH a refusal and an exemption", false: "neither a refusal case nor an exemption"}[refused])
		}
	}
	inDomain := map[string]bool{}
	for _, name := range domain {
		inDomain[name] = true
	}
	for name := range refusals {
		if !inDomain[name] {
			t.Errorf("refusal case %s names no transition that reaches the door; a stale entry is a guard for nothing", name)
		}
	}
	for name := range casOnly {
		if !inDomain[name] {
			t.Errorf("exemption %s names no transition that reaches the door", name)
		}
	}

	for name, tc := range refusals {
		t.Run(name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := tc.shape(t, store, archivableThread(t, store, "couch-0000000000000001"))
			err := tc.call(t, store, record)
			if err == nil {
				t.Fatalf("%s accepted the shape its precondition exists to refuse", name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%s refused for some other reason (want %q): %v", name, tc.want, err)
			}
			after, readErr := store.GetThread(record.Address)
			if readErr != nil {
				t.Fatalf("%s's refusal lost the record: %v", name, readErr)
			}
			if after.Revision != record.Revision {
				t.Fatalf("%s refused and still wrote: revision %d -> %d", name, record.Revision, after.Revision)
			}
		})
	}
}

// lifecycleTransitionsReachingTheDoor is the domain: exported *ThreadStore
// methods whose call graph, within the store, reaches updateExistingThread.
// A fixed point rather than one hop, because two transitions reach it through
// an unexported helper (retireIncarnation, advanceSuccessfulStart).
func lifecycleTransitionsReachingTheDoor(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRootFrom(t), "cmd", "internal", "couchcore")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	calls := map[string]map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			if ident, ok := star.X.(*ast.Ident); !ok || ident.Name != "ThreadStore" {
				continue
			}
			callees := map[string]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						callees[sel.Sel.Name] = true
					}
				}
				return true
			})
			calls[fn.Name.Name] = callees
		}
	}
	reaches := map[string]bool{"updateExistingThread": true}
	for changed := true; changed; {
		changed = false
		for method, callees := range calls {
			if reaches[method] {
				continue
			}
			for callee := range callees {
				if reaches[callee] {
					reaches[method], changed = true, true
					break
				}
			}
		}
	}
	var domain []string
	for method := range reaches {
		if ast.IsExported(method) {
			domain = append(domain, method)
		}
	}
	sort.Strings(domain)
	if len(domain) < 15 {
		t.Fatalf("derived %d transitions; the store's shape changed and this table stopped covering it", len(domain))
	}
	return domain
}

// The one production caller of BeginContinuationFromRetiredIncarnations never
// reaches its start-claim refusal, and this is the evidence rather than the
// claim (#256 M3 BR, I1).
//
// The reviewer's concern was that absent-source recovery now fails where it used
// to write. It cannot reach the write with a claim open: RecoverThread asks
// DecideRecovery, and so does prepareAbsentContinuation inside it, and both
// refuse any incarnation whose Start is set (`recovery.go`'s
// "helper start or ownership is unresolved") before FromCheckpoint can be true.
// The write then carries the revision from that same read, so a claim that
// appears afterwards fails the CAS and the retry is refused by DecideRecovery
// again. The store precondition is defence in depth for callers that do not
// screen -- which is exactly why it belongs to the transition and not the caller.
//
// What the old callback did with this shape was drop the claim underneath its
// own rollback, silently; so "fails where it previously wrote" was never
// reachable either, and the one path that could have written refuses first.
func TestAbsentSourceRecoveryRefusesAnOpenStartClaimBeforeTheStoreDoes(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	pid := f.source.Incarnations[0].PID
	f.env.Proc.Kill(pid)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)

	current, err := c.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	// Fatal, not Skip, on any error: a skip that swallows every failure is how
	// a stale-revision mistake in this very fixture first read as "the
	// validator refuses the shape".
	claimed, err := c.Threads.updateExistingThread(current.Address, current.Revision, func(r *ThreadRecord) error {
		// What a start interrupted after its helper was recorded leaves:
		// `creating`, a claim, the helper's identity, and -- because the
		// validator refuses one before registration -- no launch profile.
		r.Incarnations[0].State = IncarnationCreating
		r.Incarnations[0].LaunchProfile = nil
		r.Incarnations[0].Start = &ThreadStartClaim{Nonce: "start-00000000000000aa", OwnerPID: 991, OwnerIdentity: "gone-couch"}
		return nil
	})
	if err != nil {
		t.Fatalf("building the open-claim shape: %v", err)
	}

	_, err = c.RecoverThread(context.Background(), claimed.Address, "")
	if err == nil {
		t.Fatal("absent-source recovery wrote through an open start claim")
	}
	// DISCRIMINATING: the store's refusal would also fail this call. The
	// assertion is that the caller's own gate answered first.
	if !strings.Contains(err.Error(), "helper start or ownership is unresolved") {
		t.Fatalf("refused by something other than DecideRecovery: %v", err)
	}
	after, err := c.Threads.GetThread(claimed.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != claimed.Revision {
		t.Fatalf("a refused recovery still wrote: revision %d -> %d", claimed.Revision, after.Revision)
	}
}
