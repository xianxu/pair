package couchcore

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestSelectedListingEvidenceDescribesTheJudgedProjection(t *testing.T) {
	for _, state := range []string{"attached", "exited", ""} {
		t.Run(state, func(t *testing.T) {
			data := t.TempDir()
			const name = "📁selected"
			indexSession(t, data, addressA, name)
			sessions := map[string]string{}
			if state != "" {
				sessions[name] = state
			}
			checker, _ := sandboxedChecker(t, data, sessions)
			got, err := checker.ListedPairSessionContext(t.Context(), addressA, name)
			if err != nil {
				t.Fatal(err)
			}
			e := got.ListingEvidence
			if e == nil || e.Address != addressA || e.Name != name || e.Claims != 1 || e.Listing.At.IsZero() || e.Listing.SHA256 != fmt.Sprintf("%x", sha256.Sum256(e.Listing.Raw)) {
				t.Fatalf("missing exact projection evidence: %+v", e)
			}
			want := "missing"
			if state != "" {
				want = "live"
				if state == "exited" {
					want = "exited"
				}
			}
			if e.SelectedState != want || (e.Projection == SessionPresent) != got.Present {
				t.Fatalf("evidence contradicts decision: %+v binding=%+v", e, got)
			}
		})
	}
}
