package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestSelectedListingAcceptsLiveClientsWithoutOwnerMetadata(t *testing.T) {
	for _, state := range []string{"attached", "detached", "exited", ""} {
		t.Run(state, func(t *testing.T) {
			data := t.TempDir()
			const name = "📁selected"
			indexSession(t, data, addressA, name)
			sessions := map[string]string{}
			if state != "" {
				sessions[name] = state
			}
			checker, log := sandboxedChecker(t, data, sessions)
			checker.OwnerProbe = nil
			checker.Servers = unavailableServerSnapshot{}
			got, err := checker.ListedPairSessionContext(t.Context(), addressA, name)
			if err != nil || got.Name != name || got.Present != (state == "attached" || state == "detached") {
				t.Fatalf("selected=%+v err=%v", got, err)
			}
			if n := pairlifecycletest.CountCalls(t, log, "list-clients"); n != 0 {
				t.Fatalf("attachment consulted client count %d times", n)
			}
		})
	}
}

func TestSelectedListingRequiresPositiveUnambiguousRecordedName(t *testing.T) {
	for _, variant := range []string{"record-overrides-index", "conflicting-thread", "listing-failed"} {
		t.Run(variant, func(t *testing.T) {
			data := t.TempDir()
			const name = "📁recorded"
			indexSession(t, data, addressA, "📁stale-index")
			if variant == "conflicting-thread" {
				indexSession(t, data, addressB, name)
			}
			checker, _ := sandboxedChecker(t, data, map[string]string{name: "attached"})
			checker.OwnerProbe = nil
			if variant == "listing-failed" {
				checker.Zellij = launcher.ZellijSource{Path: "/nonexistent/fixture-zellij"}
			}
			got, err := checker.ListedPairSessionContext(t.Context(), addressA, name)
			if variant == "record-overrides-index" {
				if err != nil || !got.Present || got.Name != name {
					t.Fatalf("recorded name must win: %+v %v", got, err)
				}
			} else if err == nil || got.Present {
				t.Fatalf("uncertain listing became Continue: %+v %v", got, err)
			}
		})
	}
}
