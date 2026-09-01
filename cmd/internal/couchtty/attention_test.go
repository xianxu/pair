package couchtty

import (
	"math"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func attentionAddress(tag string) couchcore.ThreadAddress {
	return couchcore.ThreadAddress{RepoScope: "repo", Tag: couchcore.ThreadTag(tag)}
}

func attentionTexts(rows []AttentionMessage) []string {
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].Text
	}
	return out
}

func TestAttentionLedgerMarkBoundsDeduplicatesAndOrders(t *testing.T) {
	var ledger AttentionLedger
	a, b := attentionAddress("a"), attentionAddress("b")
	for _, message := range []string{"one", "two", "three", "four", "two"} {
		ledger.Mark(a, message)
	}
	ledger.Mark(b, "other")

	if got, want := attentionTexts(ledger.Projection(a)), []string{"three", "four", "two"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("projection = %v, want %v", got, want)
	}
	if got := ledger.NewestActor(); got != b {
		t.Fatalf("NewestActor = %+v, want %+v", got, b)
	}

	projection := ledger.Projection(a)
	projection[0].Text = "mutated"
	if got := attentionTexts(ledger.Projection(a)); got[0] != "three" {
		t.Fatalf("projection aliases ledger: %v", got)
	}
}

func TestAttentionLedgerCaptureAcknowledgesOnlyCapturedIdentities(t *testing.T) {
	var ledger AttentionLedger
	a := attentionAddress("a")
	ledger.Mark(a, "first")
	ledger.Mark(a, "second")
	capture := ledger.Capture(a)
	ledger.Mark(a, "later")
	ledger.Acknowledge(capture)
	if got, want := attentionTexts(ledger.Projection(a)), []string{"later"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("acknowledge = %v, want %v", got, want)
	}

	keep := ledger.Capture(a)
	ledger.Cancel(keep)
	ledger.Acknowledge(keep)
	if got := attentionTexts(ledger.Projection(a)); !reflect.DeepEqual(got, []string{"later"}) {
		t.Fatalf("cancelled capture cleared attention: %v", got)
	}
}

func TestAttentionLedgerOverflowRebasesRecordsAndCapturesAtomically(t *testing.T) {
	var ledger AttentionLedger
	a, b := attentionAddress("a"), attentionAddress("b")
	ledger.Mark(a, "captured")
	capture := ledger.Capture(a)
	ledger.nextSequence = math.MaxUint64
	ledger.Mark(b, "newest")
	ledger.Acknowledge(capture)

	if got := ledger.Projection(a); len(got) != 0 {
		t.Fatalf("rebased capture did not clear original identity: %+v", got)
	}
	if got := ledger.NewestActor(); got != b {
		t.Fatalf("NewestActor after rebase = %+v, want %+v", got, b)
	}
}

func TestAttentionLedgerDropActorRemovesOnlyItsEphemeralState(t *testing.T) {
	var ledger AttentionLedger
	a, b := attentionAddress("a"), attentionAddress("b")
	ledger.Mark(a, "gone")
	ledger.Mark(b, "kept")
	capture := ledger.Capture(a)
	ledger.DropActor(a)
	ledger.Acknowledge(capture)
	if got := ledger.Projection(a); len(got) != 0 {
		t.Fatalf("dropped actor remains: %+v", got)
	}
	if got := attentionTexts(ledger.Projection(b)); !reflect.DeepEqual(got, []string{"kept"}) {
		t.Fatalf("other actor changed: %v", got)
	}
}
