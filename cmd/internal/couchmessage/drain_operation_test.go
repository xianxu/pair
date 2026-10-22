package couchmessage

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func freshDrainOperation(t *testing.T) DrainOperation {
	t.Helper()
	op, err := NewDrainOperation("start", []DrainChild{{Workspace: "pair:1", ID: "one"}, {Workspace: "pair:2", ID: "two"}})
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func TestDrainOperationTransitions(t *testing.T) {
	for _, status := range []Status{Queued, Delivering, Held, Submitted, Expired, Cancelled, Indeterminate, Refused} {
		t.Run(string(status), func(t *testing.T) {
			op := freshDrainOperation(t)
			got, err := ApplyDrainChild(op, DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: status, Detail: "receipt evidence"}})
			if err != nil {
				t.Fatal(err)
			}
			want := DrainChildAdmitted
			if status == Submitted {
				want = DrainChildDelivered
			} else if status.Terminal() {
				want = DrainChildFailed
			}
			if got.Children[0].Phase != want || got.Children[0].ReceiptStatus != status {
				t.Fatalf("child = %+v", got.Children[0])
			}
			if want == DrainChildAdmitted {
				got, err = ApplyDrainChild(got, DrainChildEvent{ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}})
				if err != nil || got.Children[0].Phase != DrainChildDelivered {
					t.Fatalf("delivery: %+v, %v", got, err)
				}
			}
			if DrainOperationStatus(got) != ReceiptRunning {
				t.Fatal("unfinished sibling must keep operation running")
			}
			got, err = ApplyDrainChild(got, DrainChildEvent{ID: "two", Admission: true, Error: "admission deadline: delivery unconfirmed"})
			if err != nil || DrainOperationStatus(got) != ReceiptFailed || DrainNoticeState(got.Children[1]) != "failed" {
				t.Fatalf("failure: %+v %v", got, err)
			}
		})
	}
}

func TestDrainOperationSnapshotAndAdmissionFailures(t *testing.T) {
	for _, event := range []DrainChildEvent{
		{Receipt: nil}, {Error: "receipt unavailable"}, {Receipt: &Receipt{Message: Message{ID: "one"}, Status: Indeterminate}},
		{Receipt: &Receipt{Message: Message{ID: "one"}, Status: Expired}},
	} {
		op := freshDrainOperation(t)
		admitted, err := ApplyDrainChild(op, DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: Queued}})
		if err != nil {
			t.Fatal(err)
		}
		event.ID = "one"
		got, err := ApplyDrainChild(admitted, event)
		if err != nil || got.Children[0].Phase != DrainChildFailed || got.Children[0].Detail == "" {
			t.Fatalf("snapshot = %+v %v", got, err)
		}
	}
	for _, event := range []DrainChildEvent{{ID: "one", Admission: true}, {ID: "one", Admission: true, Error: "unavailable"}} {
		got, err := ApplyDrainChild(freshDrainOperation(t), event)
		if err != nil || got.Children[0].Phase != DrainChildFailed || !strings.Contains(got.Children[0].Detail, "unconfirmed") {
			t.Fatalf("admission = %+v %v", got, err)
		}
	}
}

func TestDrainOperationRejectsStaleContradictoryEventsWithoutMutation(t *testing.T) {
	awaiting := freshDrainOperation(t)
	admitted, err := ApplyDrainChild(awaiting, DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: Queued}})
	if err != nil {
		t.Fatal(err)
	}
	delivered, err := ApplyDrainChild(admitted, DrainChildEvent{ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		op DrainOperation
		e  DrainChildEvent
	}{
		{awaiting, DrainChildEvent{ID: "missing", Admission: true}},
		{awaiting, DrainChildEvent{ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}}},
		{admitted, DrainChildEvent{ID: "one", Admission: true, Error: "late"}},
		{delivered, DrainChildEvent{ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}}},
		{delivered, DrainChildEvent{ID: "one", Error: "late"}},
		{awaiting, DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "other"}, Status: Queued}}},
		{awaiting, DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: "bogus"}}},
		{awaiting, DrainChildEvent{ID: "one", Admission: true, Error: "contradiction", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}}},
	}
	for i, tc := range cases {
		before := append([]DrainChild(nil), tc.op.Children...)
		got, err := ApplyDrainChild(tc.op, tc.e)
		if err == nil || !reflect.DeepEqual(got, tc.op) || !reflect.DeepEqual(before, tc.op.Children) {
			t.Fatalf("case%d: %+v %v", i, got, err)
		}
		got.Children[0].Detail = "mutated"
		if !reflect.DeepEqual(before, tc.op.Children) {
			t.Fatalf("case%d aliases error result", i)
		}
	}
}

func TestDrainOperationConstructorBoundsAndAliasing(t *testing.T) {
	for _, children := range [][]DrainChild{
		{{Workspace: "", ID: "one"}}, {{Workspace: "pair:1", ID: ""}},
		{{Workspace: "pair:1", ID: "one"}, {Workspace: "pair:2", ID: "one"}},
		{{Workspace: "pair:1", ID: "one"}, {Workspace: "pair:1", ID: "two"}},
		{{Workspace: "pair:1", ID: "one", Phase: DrainChildDelivered}},
		{{Workspace: "pair:1", ID: "one", ReceiptStatus: Queued}},
		{{Workspace: "pair:1", ID: "one", Detail: "not fresh"}},
	} {
		if _, err := NewDrainOperation("start", children); err == nil {
			t.Fatalf("accepted %+v", children)
		}
	}
	if _, err := NewDrainOperation("bad", nil); err == nil {
		t.Fatal("accepted action")
	}
	children := make([]DrainChild, MaxActors)
	for i := range children {
		children[i] = DrainChild{Workspace: fmt.Sprintf("pair:%d", i), ID: fmt.Sprintf("child-%d", i)}
	}
	op, err := NewDrainOperation("undrain", children)
	if err != nil {
		t.Fatal(err)
	}
	children[0].ID = "changed"
	if op.Children[0].ID == "changed" {
		t.Fatal("constructor aliases")
	}
	children = append(children, DrainChild{Workspace: "extra", ID: "extra"})
	if _, err := NewDrainOperation("start", children); err == nil {
		t.Fatal("silently truncated children")
	}
	out, err := ApplyDrainChild(op, DrainChildEvent{ID: "child-0", Admission: true, Receipt: &Receipt{Message: Message{ID: "child-0"}, Status: Queued}})
	if err != nil {
		t.Fatal(err)
	}
	out.Children[0].Workspace = "changed"
	if op.Children[0].Workspace == "changed" {
		t.Fatal("success aliases")
	}
	empty, err := NewDrainOperation("start", nil)
	if err != nil || DrainOperationStatus(empty) != ReceiptSucceeded {
		t.Fatal("empty operation not successful")
	}
}

func TestDrainOperationDetailEncodedBound(t *testing.T) {
	for _, detail := range []string{strings.Repeat("\x00\"\\<>&", 1000), strings.Repeat("界", 1000), "invalid\xff" + strings.Repeat("x", 1000)} {
		op, err := ApplyDrainChild(freshDrainOperation(t), DrainChildEvent{ID: "one", Admission: true, Error: detail})
		if err != nil {
			t.Fatal(err)
		}
		got := op.Children[0].Detail
		raw, err := json.Marshal(got)
		if err != nil || len(raw) > 256 || !utf8.ValidString(got) || !strings.Contains(got, "…") {
			t.Fatalf("detail len=%d valid=%v: %q", len(raw), utf8.ValidString(got), got)
		}
	}
}

func TestDrainOperationSuccessAndPendingProjection(t *testing.T) {
	op := freshDrainOperation(t)
	if DrainNoticeState(op.Children[0]) != "pending" {
		t.Fatal("awaiting not pending")
	}
	for _, id := range []string{"two", "one"} {
		var err error
		op, err = ApplyDrainChild(op, DrainChildEvent{ID: id, Admission: true, Receipt: &Receipt{Message: Message{ID: id}, Status: Submitted}})
		if err != nil {
			t.Fatal(err)
		}
	}
	if DrainOperationStatus(op) != ReceiptSucceeded || DrainNoticeState(op.Children[0]) != "delivered" {
		t.Fatalf("operation %+v", op)
	}
}

func TestDrainOperationIdentityEncodedBounds(t *testing.T) {
	for _, child := range []DrainChild{
		{Workspace: "pair:1", ID: strings.Repeat("x", 129)},
		{Workspace: "pair:1", ID: "invalid\xff"},
		{Workspace: "invalid\xff", ID: "one"},
		{Workspace: strings.Repeat("\x00", MaxBindingBytes/6+1), ID: "one"},
	} {
		if _, err := NewDrainOperation("start", []DrainChild{child}); err == nil {
			t.Fatalf("accepted invalid identity %+v", child)
		}
	}
	// JSON quotes count toward the workspace bound; no identity truncation.
	workspace := strings.Repeat("x", MaxBindingBytes-2)
	op, err := NewDrainOperation("start", []DrainChild{{Workspace: workspace, ID: strings.Repeat("x", 128)}})
	if err != nil || op.Children[0].Workspace != workspace {
		t.Fatalf("boundary: %+v %v", op, err)
	}
	if _, err := NewDrainOperation("start", []DrainChild{{Workspace: workspace + "x", ID: "one"}}); err == nil {
		t.Fatal("accepted overbound workspace")
	}
}

func TestDrainOperationAdmittedSnapshotsRemainPendingAndFailedIsImmutable(t *testing.T) {
	op := freshDrainOperation(t)
	op, err := ApplyDrainChild(op, DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: Queued}})
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []Status{Queued, Held, Delivering} {
		op, err = ApplyDrainChild(op, DrainChildEvent{ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: status}})
		if err != nil || op.Children[0].Phase != DrainChildAdmitted || DrainNoticeState(op.Children[0]) != "pending" {
			t.Fatalf("snapshot %+v %v", op, err)
		}
	}
	failed, err := ApplyDrainChild(op, DrainChildEvent{ID: "one", Error: "read lost"})
	if err != nil || failed.Children[0].ReceiptStatus != Delivering {
		t.Fatalf("lost snapshot must preserve last observation: %+v %v", failed, err)
	}
	got, err := ApplyDrainChild(failed, DrainChildEvent{ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}})
	if err == nil || !reflect.DeepEqual(got, failed) {
		t.Fatalf("failed child resurrected: %+v %v", got, err)
	}
}

func TestDrainOperationRetainsShortDiagnostics(t *testing.T) {
	detail := "literal <note> with \"quotes\" and 界"
	got, err := ApplyDrainChild(freshDrainOperation(t), DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: Held, Detail: detail}})
	if err != nil || got.Children[0].Detail != detail {
		t.Fatalf("diagnostic changed: %+v %v", got, err)
	}
}

func TestDrainOperationBatchAppliesEveryChildAndDetaches(t *testing.T) {
	children := make([]DrainChild, MaxActors)
	events := make([]DrainChildEvent, MaxActors)
	for i := range children {
		id := fmt.Sprintf("child-%d", i)
		children[i] = DrainChild{Workspace: fmt.Sprintf("pair:%d", i), ID: id}
		events[len(events)-1-i] = DrainChildEvent{ID: id, Admission: true, Receipt: &Receipt{Message: Message{ID: id}, Status: Queued}}
	}
	op, err := NewDrainOperation("start", children)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyDrainChildren(op, events)
	if err != nil {
		t.Fatal(err)
	}
	admitted := 0
	for _, child := range got.Children {
		if child.Phase == DrainChildAdmitted {
			admitted++
		}
	}
	if admitted != MaxActors {
		t.Fatalf("applied %d children, want %d", admitted, MaxActors)
	}
	for i := range events {
		events[i].Admission = false
		events[i].Receipt.Status = Submitted
	}
	final, err := ApplyDrainChildren(got, events)
	if err != nil || DrainOperationStatus(final) != ReceiptSucceeded {
		t.Fatalf("final %+v %v", final, err)
	}
	if op.Children[0].Phase != DrainChildAwaitingAdmission || got.Children[0].Phase != DrainChildAdmitted {
		t.Fatal("batch mutated input snapshot")
	}
	final.Children[0].Detail = "mutated"
	if got.Children[0].Detail != "" {
		t.Fatal("batch result aliases input")
	}
}

func TestDrainOperationBatchFailureIsAtomicAndDetached(t *testing.T) {
	admitted := DrainChildEvent{ID: "one", Admission: true, Receipt: &Receipt{Message: Message{ID: "one"}, Status: Queued}}
	for _, events := range [][]DrainChildEvent{
		{admitted, admitted},
		{admitted, {ID: "unknown", Admission: true}},
		{admitted, {ID: "two", Admission: true, Receipt: &Receipt{Message: Message{ID: "wrong"}, Status: Queued}}},
		// Even a transition-valid repeated ID is malformed in one snapshot batch.
		{admitted, {ID: "one", Receipt: &Receipt{Message: Message{ID: "one"}, Status: Submitted}}},
	} {
		op := freshDrainOperation(t)
		got, err := ApplyDrainChildren(op, events)
		if err == nil || !reflect.DeepEqual(got, op) {
			t.Fatalf("non-atomic batch: %+v %v", got, err)
		}
		got.Children[0].Detail = "changed"
		if op.Children[0].Detail != "" {
			t.Fatal("failed batch aliases original")
		}
	}
	op := freshDrainOperation(t)
	got, err := ApplyDrainChildren(op, nil)
	if err != nil || !reflect.DeepEqual(got, op) {
		t.Fatalf("empty batch %+v %v", got, err)
	}
	got.Children[0].Detail = "changed"
	if op.Children[0].Detail != "" {
		t.Fatal("empty batch aliases original")
	}
}

func TestDrainOperationMaximumResponseFitsTransport(t *testing.T) {
	children := make([]DrainChild, MaxActors)
	for i := range children {
		children[i] = DrainChild{Workspace: fmt.Sprintf("%03d", i) + strings.Repeat("<", 650), ID: fmt.Sprintf("%03d", i) + strings.Repeat("x", 125)}
	}
	op, err := NewDrainOperation("start", children)
	if err != nil {
		t.Fatal(err)
	}
	events := make([]DrainChildEvent, len(children))
	for i, child := range children {
		events[i] = DrainChildEvent{ID: child.ID, Admission: true, Error: strings.Repeat("\x00", 1000)}
	}
	op, err = ApplyDrainChildren(op, events)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(Response{Code: "ok", Operation: &OperationReceipt{ID: strings.Repeat("x", 128), Op: "drain", Status: ReceiptFailed, Drain: &op}})
	if err != nil || len(raw) > MaxFrameBytes {
		t.Fatalf("response size=%d limit=%d error=%v", len(raw), MaxFrameBytes, err)
	}
}

func TestDrainOperationReceiptOwnsCohort(t *testing.T) {
	op := freshDrainOperation(t)
	event := ReceiptEvent{Kind: ReceiptAdmit, ID: "parent", Op: "drain", Drain: &op}
	receipt, err := ApplyReceiptEvent(OperationReceipt{}, event)
	if err != nil {
		t.Fatal(err)
	}
	op.Children[0].ID = "mutated"
	if receipt.Drain.Children[0].ID != "one" {
		t.Fatal("registered cohort aliases request")
	}
}
