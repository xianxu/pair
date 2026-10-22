package couchcore

import (
	"context"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type ownedProofDeleter struct {
	legacy int
	proofs []launcher.SessionOwnerObservation
}

func (d *ownedProofDeleter) DeleteSession(string) error { d.legacy++; return nil }
func (d *ownedProofDeleter) DeleteOwnedSessionContext(_ context.Context, owner launcher.SessionOwnerObservation) error {
	d.proofs = append(d.proofs, owner)
	return nil
}

type nameOnlyDeleter struct{ calls int }

func (d *nameOnlyDeleter) DeleteSession(string) error { d.calls++; return nil }

func TestScopedDeletionRequiresOriginalOwnedProof(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, site := range []string{"indexed", "named", "quit"} {
		for _, supported := range []bool{false, true} {
			t.Run(site+map[bool]string{false: "/unsupported", true: "/owned"}[supported], func(t *testing.T) {
				root := t.TempDir()
				address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
				name := "📁proof-1"
				indexSession(t, root, address, name)
				checker := NewScopedThreadArtifactCollisionChecker(root)
				checker.OwnerProbe = testOwnerProbe(root, name, address)
				owned := &ownedProofDeleter{}
				legacy := &nameOnlyDeleter{}
				if supported {
					checker.Sessions = owned
				} else {
					checker.Sessions = legacy
				}
				binding, err := checker.NamedPairSessionContext(t.Context(), address, name)
				if err != nil {
					t.Fatal(err)
				}
				switch site {
				case "indexed":
					err = checker.Quiesce(address)
				case "named":
					err = checker.QuiesceNamed(t.Context(), address, name)
				case "quit":
					err = checker.TriggerBoundQuit(t.Context(), binding, launcher.QuitIntent{
						Version: launcher.QuitIntentVersion, Kind: launcher.QuitIntentCouch,
						Request: &launcher.QuitRequestReference{DataDir: root, RepoScope: address.RepoScope, Tag: string(address.Tag), Nonce: "proof-quit", Attempt: 1},
					})
				}
				if supported {
					if err != nil || !reflect.DeepEqual(owned.proofs, []launcher.SessionOwnerObservation{*binding.Owner}) || owned.legacy != 0 {
						t.Fatalf("exact proof handoff: err=%v proofs=%+v want=%+v legacy=%d", err, owned.proofs, binding.Owner, owned.legacy)
					}
				} else if err == nil || legacy.calls != 0 {
					t.Fatalf("unsupported deleter: err=%v name effects=%d", err, legacy.calls)
				}
			})
		}
	}
}
