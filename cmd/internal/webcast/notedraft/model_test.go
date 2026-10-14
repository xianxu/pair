package notedraft

import "testing"

func TestStepUnknownSaveCannotBeDeleted(t *testing.T) {
	s := State{Phase: Ready}
	s, _, err := Step(s, Event{Kind: Save})
	if err != nil || s.Phase != Attaching {
		t.Fatalf("save: %+v %v", s, err)
	}
	for _, kind := range []EventKind{Cancel, Expire, Resolve} {
		abandoned, effects, err := Step(s, Event{Kind: kind})
		if err != nil || abandoned.Phase != Attaching || !abandoned.Abandoned || hasEffect(effects, DeleteSource) {
			t.Fatalf("uncertain save %s: %+v %v %v", kind, abandoned, effects, err)
		}
		attached, _, err := Step(abandoned, Event{Kind: WriteConfirmed})
		if err != nil || attached.Phase != Attached {
			t.Fatalf("confirmed save lost: %+v %v", attached, err)
		}
		retired, effects, err := Step(abandoned, Event{Kind: WriteAbsent})
		if err != nil || retired.Phase != Retired || !hasEffect(effects, DeleteSource) {
			t.Fatalf("proven absence: %+v %v %v", retired, effects, err)
		}
	}
}

func TestStepTransferPreservesEvidenceUntilConfirmed(t *testing.T) {
	s, _, err := Step(State{Phase: Attached}, Event{Kind: Publish})
	if err != nil || s.Phase != Transferring {
		t.Fatal(s, err)
	}
	s, effects, err := Step(s, Event{Kind: Resolve})
	if err != nil || !s.Abandoned || hasEffect(effects, DeleteSource) {
		t.Fatal(s, effects, err)
	}
	s, effects, err = Step(s, Event{Kind: BundleConfirmed})
	if err != nil || s.Phase != Retired || len(effects) < 2 || effects[0] != RebindBundle || effects[1] != DeleteSource {
		t.Fatal(s, effects, err)
	}
}

func TestStepUnavailableCaptureStillSaves(t *testing.T) {
	s, _, err := Step(State{Phase: Preparing}, Event{Kind: CaptureFailed, Reason: CaptureBusy})
	if err != nil || s.Phase != Ready || s.Unavailable != CaptureBusy {
		t.Fatal(s, err)
	}
	s, _, err = Step(s, Event{Kind: Save})
	if err != nil || s.Phase != Attaching {
		t.Fatal(s, err)
	}
}

func TestStepRetiredNeverRevives(t *testing.T) {
	for _, kind := range []EventKind{CaptureComplete, CaptureFailed, Save, WriteConfirmed, Publish, BundleConfirmed, Cancel, Expire, Resolve} {
		s, _, _ := Step(State{Phase: Retired}, Event{Kind: kind})
		if s.Phase != Retired {
			t.Fatalf("%s revived %+v", kind, s)
		}
	}
}

func TestCapacityReservationsCountAndRelease(t *testing.T) {
	if Capacity(127, 0, 1, MaxPNGBytes) != "" {
		t.Fatal("last reservation refused")
	}
	if Capacity(128, 0, 1, MaxPNGBytes) != StoreFull {
		t.Fatal("record limit bypassed")
	}
	if Capacity(0, MaxStoreBytes-MaxPNGBytes, 1, MaxPNGBytes) != "" {
		t.Fatal("exact byte boundary refused")
	}
	if Capacity(0, MaxStoreBytes-MaxPNGBytes+1, 1, MaxPNGBytes) != StoreFull {
		t.Fatal("byte reservation bypassed")
	}
}

func hasEffect(e []Effect, want Effect) bool {
	for _, got := range e {
		if got == want {
			return true
		}
	}
	return false
}
