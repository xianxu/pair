package couchcore

import (
	"strings"
	"testing"
	"time"
)

// RetireIncarnation is FinalizePark's removal half without the park
// transaction: detach kills the client but leaves the zellij session, so the
// record must lose its incarnation WITHOUT gaining a verified park.
//
// Exact process identity is the authorization, the same rule observeExactProcess
// and MarkIncarnationUnknown already use, so a recycled PID cannot retire a
// thread that is genuinely live.
func TestThreadStoreRetireIncarnation(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	identity := ProcessIdentity{PID: 4242, Identity: "start-token"}
	profile := &LaunchProfile{Agent: "claude", Argv: []string{"--flag"}}

	live := func() ThreadIncarnation {
		return ThreadIncarnation{State: IncarnationLive, PID: identity.PID, Identity: identity.Identity, StartedAt: time.Unix(10, 0).UTC()}
	}

	tests := []struct {
		name     string
		mutate   func(*ThreadRecord)
		identity ProcessIdentity
		wantErr  string
	}{
		{name: "retires the exactly matching live incarnation", identity: identity},
		{
			name:     "refuses a recycled PID whose start token differs",
			identity: ProcessIdentity{PID: identity.PID, Identity: "other-token"},
			wantErr:  "incarnation",
		},
		{
			name:     "refuses a different PID",
			identity: ProcessIdentity{PID: 9999, Identity: identity.Identity},
			wantErr:  "incarnation",
		},
		{
			name:     "refuses a record with no incarnation",
			mutate:   func(r *ThreadRecord) { r.Incarnations = nil },
			identity: identity,
			wantErr:  "incarnation",
		},
		{
			name: "refuses an unknown incarnation -- detach must not retire unproven state",
			mutate: func(r *ThreadRecord) {
				r.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown, PID: identity.PID, Identity: identity.Identity, StartedAt: time.Unix(10, 0).UTC()}}
			},
			identity: identity,
			wantErr:  "incarnation",
		},
		{
			name: "refuses while a park transaction is open",
			mutate: func(r *ThreadRecord) {
				r.Park = &ParkTransaction{
					Identity:       ParkIdentity{Nonce: "park-0001020304050607", Address: address, PID: identity.PID, ProcessIdentity: identity.Identity},
					BaseRevision:   1,
					RecordRevision: 2,
					Phase:          ParkRequested,
					Attempts:       []ParkAttempt{{Number: 1}},
				}
			},
			identity: identity,
			wantErr:  "park",
		},
		{
			name: "refuses while a start transaction is open",
			mutate: func(r *ThreadRecord) {
				r.Incarnations[0].State = IncarnationCreating
				r.Incarnations[0].Start = &ThreadStartClaim{
					Nonce: "start-0001020304050607", OwnerPID: 7, OwnerIdentity: "owner-token",
				}
			},
			identity: identity,
			wantErr:  "incarnation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			seed := validThreadRecord(t)
			seed.Address, seed.StartingPath, seed.WorkingPath = address, ns.Dir(), ns.Dir()
			record, err := store.CreateThread(seed)
			if err != nil {
				t.Fatal(err)
			}
			record, err = store.UpdateExistingThread(address, record.Revision, func(next *ThreadRecord) error {
				next.Reservation = false
				next.Incarnations = []ThreadIncarnation{live()}
				next.LatestLaunchProfile = profile
				next.Name = "keep-me"
				if test.mutate != nil {
					test.mutate(next)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}

			got, err := store.RetireIncarnation(address, record.Revision, test.identity)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("RetireIncarnation() error = %v, want one mentioning %q", err, test.wantErr)
				}
				after, readErr := store.GetThread(address)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if after.Revision != record.Revision {
					t.Fatalf("a refused retire still wrote: revision %d -> %d", record.Revision, after.Revision)
				}
				return
			}
			if err != nil {
				t.Fatalf("RetireIncarnation() = %v", err)
			}
			if len(got.Incarnations) != 0 {
				t.Fatalf("incarnations = %+v, want none", got.Incarnations)
			}
			if got.VerifiedPark != nil {
				t.Fatal("retire wrote a verified park -- detach is not a park")
			}
			if got.LatestLaunchProfile == nil || got.LatestLaunchProfile.Agent != "claude" {
				t.Fatalf("launch profile lost: %+v -- it is what reattach needs", got.LatestLaunchProfile)
			}
			if got.Name != "keep-me" {
				t.Fatalf("name lost: %q", got.Name)
			}
		})
	}

	t.Run("refuses a stale revision", func(t *testing.T) {
		store, ns := newTestThreadStore(t)
		seed := validThreadRecord(t)
		seed.Address, seed.StartingPath, seed.WorkingPath = address, ns.Dir(), ns.Dir()
		record, err := store.CreateThread(seed)
		if err != nil {
			t.Fatal(err)
		}
		record, err = store.UpdateExistingThread(address, record.Revision, func(next *ThreadRecord) error {
			next.Reservation = false
			next.Incarnations = []ThreadIncarnation{live()}
			next.LatestLaunchProfile = profile
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.RetireIncarnation(address, record.Revision-1, identity); err == nil {
			t.Fatal("a stale revision was accepted")
		}
	})
}

// A detached resume has no verified park, and DeleteStart's no-verified-park
// branch used to fall through to deleteThreadIf. threadHasMetadata protects a
// NAMED record; the exposure was the unnamed one, whose LatestLaunchProfile
// nothing guarded -- so a post-claim rollback deleted the agent and argv needed
// to reattach while the zellij session it names kept running.
func TestDeleteStartKeepsARecordThatHasEverStarted(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	profile := &LaunchProfile{Agent: "claude", Argv: []string{"--flag"}}

	for _, test := range []struct {
		name       string
		profile    *LaunchProfile
		named      bool
		wantRecord bool
	}{
		{name: "unnamed but previously started -- the exposed case", profile: profile, wantRecord: true},
		{name: "named and previously started", profile: profile, named: true, wantRecord: true},
		{name: "never started and unnamed is still deleted", wantRecord: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			seed := validThreadRecord(t)
			seed.Address, seed.StartingPath, seed.WorkingPath = address, ns.Dir(), ns.Dir()
			record, err := store.CreateThread(seed)
			if err != nil {
				t.Fatal(err)
			}
			record, err = store.UpdateExistingThread(address, record.Revision, func(next *ThreadRecord) error {
				next.Reservation = false
				next.LatestLaunchProfile = test.profile
				if test.named {
					next.Name = "compiler"
				}
				next.Incarnations = []ThreadIncarnation{{
					State: IncarnationCreating, StartedAt: time.Unix(10, 0).UTC(),
					Start: &ThreadStartClaim{Nonce: "start-0001020304050607", OwnerPID: 7, OwnerIdentity: "owner"},
				}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}

			if err := store.DeleteStart(address, record.Revision, "start-0001020304050607"); err != nil {
				t.Fatalf("DeleteStart() = %v", err)
			}

			after, err := store.GetThread(address)
			if !test.wantRecord {
				if err == nil {
					t.Fatalf("record survived: %+v", after)
				}
				return
			}
			if err != nil {
				t.Fatalf("record was deleted, losing the profile reattach needs: %v", err)
			}
			if len(after.Incarnations) != 0 {
				t.Fatalf("start claim was not rolled back: %+v", after.Incarnations)
			}
			if after.LatestLaunchProfile == nil || after.LatestLaunchProfile.Agent != "claude" {
				t.Fatalf("launch profile lost: %+v", after.LatestLaunchProfile)
			}
		})
	}
}
