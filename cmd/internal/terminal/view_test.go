package terminal

import "testing"

func TestViewAdmitsOnlyMatchingSuccessfulPresentation(t *testing.T) {
	v, _, err := Transition(View{}, ViewEvent{Kind: SelectView, EndpointID: "a", Token: 1, GeometryEpoch: 2})
	if err != nil || v.Admitted != "" || v.State != Presenting {
		t.Fatalf("select %+v %v", v, err)
	}
	stale, _, err := Transition(v, ViewEvent{Kind: PresentView, EndpointID: "a", Token: 0, GeometryEpoch: 2})
	if err == nil || stale.Admitted != "" {
		t.Fatal("stale completion admitted")
	}
	v, _, err = Transition(v, ViewEvent{Kind: PresentView, EndpointID: "a", Token: 1, GeometryEpoch: 2, Generation: 3})
	if err != nil || v.Admitted != "a" || v.State != Ready {
		t.Fatalf("completion %+v %v", v, err)
	}
	if _, _, err = Transition(v, ViewEvent{Kind: PublishFrame, EndpointID: "b", Generation: 4, GeometryEpoch: 2}); err == nil {
		t.Fatal("wrong identity accepted")
	}
	if _, _, err = Transition(v, ViewEvent{Kind: PublishFrame, EndpointID: "a", Generation: 2, GeometryEpoch: 2}); err == nil {
		t.Fatal("stale generation accepted")
	}
	v, _, _ = Transition(v, ViewEvent{Kind: FailView})
	if v.Admitted != "" || v.State != Failed {
		t.Fatal("failure retains admission")
	}
	v, _, _ = Transition(v, ViewEvent{Kind: ReleaseView})
	if _, _, err = Transition(v, ViewEvent{Kind: SelectView, EndpointID: "a", Token: 2}); err == nil {
		t.Fatal("released view revived")
	}
}

func TestViewSwitchCancelsDragAndSuppressesRemainder(t *testing.T) {
	v := View{State: Ready, Selected: "a", Admitted: "a", Token: 1}
	v, _, _ = Transition(v, ViewEvent{Kind: PressMouse})
	v, e, err := Transition(v, ViewEvent{Kind: SelectView, EndpointID: "b", Token: 2})
	if err != nil || e.CancelDrag != "a" || v.Gesture != GestureParent || v.DragDestination != "" {
		t.Fatalf("cancel %+v %+v %v", v, e, err)
	}
	v, _, _ = Transition(v, ViewEvent{Kind: SettleMouseCancellation, EndpointID: "a"})
	v, _, _ = Transition(v, ViewEvent{Kind: PresentView, EndpointID: "b", Token: 2})
	if _, _, err = Transition(v, ViewEvent{Kind: PressMouse}); err == nil {
		t.Fatal("old gesture restarted")
	}
	v, _, _ = Transition(v, ViewEvent{Kind: ReleaseMouse})
	v, _, err = Transition(v, ViewEvent{Kind: PressMouse})
	if err != nil || v.DragDestination != "b" {
		t.Fatalf("new press %+v %v", v, err)
	}
}

func TestViewPanelAndResizeRejectStaleEpoch(t *testing.T) {
	v := View{State: Ready, Selected: "a", Admitted: "a", Token: 1, GeometryEpoch: 1}
	v, _, err := Transition(v, ViewEvent{Kind: SelectView, Token: 2, GeometryEpoch: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Transition(v, ViewEvent{Kind: PresentView, Token: 2, GeometryEpoch: 1}); err == nil {
		t.Fatal("old geometry admitted")
	}
	v, e, err := Transition(v, ViewEvent{Kind: PresentView, Token: 2, GeometryEpoch: 2})
	if err != nil || v.State != Ready || v.Admitted != "" || e.Admit {
		t.Fatalf("panel admits input %+v %+v %v", v, e, err)
	}
	if _, _, err := Transition(v, ViewEvent{Kind: PressMouse}); err == nil {
		t.Fatal("panel press admitted to child")
	}
}

func TestViewParentGestureSurvivesSelectionAndRequiresMatchingRelease(t *testing.T) {
	v, _, err := Transition(View{State: Ready}, ViewEvent{Kind: ParentPressMouse, Button: 1})
	if err != nil || v.Gesture != GestureParent || v.DragDestination != "" {
		t.Fatalf("parent press:%+v %v", v, err)
	}
	v, _, err = Transition(v, ViewEvent{Kind: SelectView, EndpointID: "child", Token: 1})
	if err != nil || v.Gesture != GestureParent {
		t.Fatalf("selection lost ownership:%+v %v", v, err)
	}
	v, _, _ = Transition(v, ViewEvent{Kind: PresentView, EndpointID: "child", Token: 1})
	if _, _, err := Transition(v, ViewEvent{Kind: PressMouse, Button: 1}); err == nil {
		t.Fatal("parent gesture became child")
	}
	if _, _, err := Transition(v, ViewEvent{Kind: ReleaseMouse, Button: 2}); err == nil {
		t.Fatal("different button ended gesture")
	}
	v, _, err = Transition(v, ViewEvent{Kind: ReleaseMouse, Button: 1})
	if err != nil || v.Gesture != GestureNone || v.Button != 0 {
		t.Fatalf("release:%+v %v", v, err)
	}
}

func TestViewCancellationRevocationPrecedesDeliveryAndSurvivesRelease(t *testing.T) {
	v := View{State: Ready, Selected: "a", Admitted: "a", Gesture: GestureChild, DragDestination: "a", Button: 1}
	v, e, err := Transition(v, ViewEvent{Kind: CancelMouse})
	if err != nil || e.CancelDrag != "a" || v.Gesture != GestureParent || v.PendingMouseRelease != "a" {
		t.Fatalf("cancel:%+v %+v %v", v, e, err)
	}
	v, e, err = Transition(v, ViewEvent{Kind: CancelMouse})
	if err != nil || e.CancelDrag != "" || v.PendingMouseRelease != "a" {
		t.Fatal("retry duplicated cancellation")
	}
	v, _, _ = Transition(v, ViewEvent{Kind: ReleaseMouse, Button: 1})
	if v.PendingMouseRelease != "a" {
		t.Fatal("physical release erased pending delivery")
	}
	if _, _, err := Transition(v, ViewEvent{Kind: PressMouse, Button: 1}); err == nil {
		t.Fatal("new child gesture admitted before cancellation settled")
	}
	v, _, err = Transition(v, ViewEvent{Kind: SettleMouseCancellation, EndpointID: "a"})
	if err != nil || v.PendingMouseRelease != "" {
		t.Fatalf("settle:%+v %v", v, err)
	}
}
