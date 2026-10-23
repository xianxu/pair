package couchcore

import (
	"reflect"
	"testing"
)

func editedRestartFixture() (EditedRestartIntent, ResumeAttemptEvidence) {
	intent := EditedRestartIntent{OperationID: "operation", Address: ThreadAddress{RepoScope: "scope", Tag: "thread"}, SourceRevision: 7, SourceFingerprint: "accepted", Profile: LaunchProfile{Agent: "codex", Argv: []string{"--model", "chosen"}}, ExpectedNativeID: "native-A"}
	event := ResumeAttemptEvidence{Kind: EditedRestartAdmission, OperationID: intent.OperationID, Address: intent.Address, SourceRevision: intent.SourceRevision, SourceFingerprint: intent.SourceFingerprint, NativeKind: EditedRestartNativeFound, NativeID: "native-A", Admission: EditedRestartSourceAdmitted, Occupancy: EditedRestartVacant, RecordRevision: 10, NextNonce: "resume-1"}
	return intent, event
}

func editedRestartResumeClaim(t *testing.T, intent EditedRestartIntent, event ResumeAttemptEvidence) EditedRestartState {
	t.Helper()
	state, request, err := AdvanceEditedRestart(intent, EditedRestartState{}, event)
	if err != nil || request == nil || request.Shape != StartColdResume {
		t.Fatalf("resume admission: state=%+v request=%+v err=%v", state, request, err)
	}
	claim := event
	claim.Kind, claim.AttemptNonce, claim.ClaimRevision, claim.RecordRevision = EditedRestartClaimRecorded, request.Nonce, 11, 11
	state, request, err = AdvanceEditedRestart(intent, state, claim)
	if err != nil || request != nil || state.ClaimRevision != 11 {
		t.Fatalf("claim recording: state=%+v request=%+v err=%v", state, request, err)
	}
	return state
}

func editedRestartMissingAttempt(event ResumeAttemptEvidence) ResumeAttemptEvidence {
	event.Kind, event.NativeKind = EditedRestartAttemptFailed, EditedRestartNativeMissing
	event.AttemptNonce, event.ClaimRevision = "resume-1", 11
	event.RecordRevision, event.CleanupRevision = 15, 15
	event.Cleanup, event.NextNonce = EditedRestartCleanupConfirmed, "fresh-2"
	return event
}

func TestEditedRestartExactResumeAndFreshRequests(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, empty := range []bool{false, true} {
			t.Run(agent+map[bool]string{false: "/flags", true: "/empty"}[empty], func(t *testing.T) {
				intent, event := editedRestartFixture()
				intent.Profile.Agent = agent
				if empty {
					intent.Profile.Argv = []string{}
				}
				state, request, err := AdvanceEditedRestart(intent, EditedRestartState{}, event)
				if err != nil || request == nil || request.Shape != StartColdResume || request.RequiredNativeID != "native-A" || request.Nonce != "resume-1" || request.ExpectedRevision != 10 || !reflect.DeepEqual(request.Profile, intent.Profile) {
					t.Fatalf("resume: %+v, %v", request, err)
				}
				if state.Phase != EditedRestartResumeIssued || state.AttemptNonce != request.Nonce {
					t.Fatalf("request was not consumed: %+v", state)
				}
				if len(request.Profile.Argv) > 0 {
					request.Profile.Argv[0] = "mutated-return"
					if intent.Profile.Argv[0] == "mutated-return" {
						t.Fatal("request aliases accepted argv")
					}
				}
				state = editedRestartResumeClaim(t, intent, event)
				missing := editedRestartMissingAttempt(event)
				state, request, err = AdvanceEditedRestart(intent, state, missing)
				if err != nil || request == nil || request.Shape != StartFreshExisting || request.RequiredNativeID != "" || request.Nonce != "fresh-2" || request.ExpectedRevision != 15 || request.Notice != EditedRestartFallbackNotice || !reflect.DeepEqual(request.Profile, intent.Profile) || state.Phase != EditedRestartFreshIssued {
					t.Fatalf("fallback: state=%+v request=%+v err=%v", state, request, err)
				}
				for range 3 {
					var replay *EditedRestartLaunch
					state, replay, err = AdvanceEditedRestart(intent, state, missing)
					if replay != nil {
						t.Fatal("replayed absence launched twice")
					}
				}
			})
		}
	}
}

func TestEditedRestartInitialPositiveAbsence(t *testing.T) {
	for _, expected := range []string{"", "native-A"} {
		intent, event := editedRestartFixture()
		intent.ExpectedNativeID = expected
		event.NativeID = expected
		event.NativeKind = EditedRestartNativeMissing
		state, request, err := AdvanceEditedRestart(intent, EditedRestartState{}, event)
		if err != nil || request == nil || request.Shape != StartFreshExisting || request.Notice != EditedRestartFallbackNotice || state.Phase != EditedRestartFreshIssued {
			t.Fatalf("initial absence %q: %+v %v", expected, request, err)
		}
	}
}

func TestEditedRestartFailedAttemptRequiresEveryCleanupFence(t *testing.T) {
	mutations := map[string]func(*ResumeAttemptEvidence){
		"unknown-native":      func(e *ResumeAttemptEvidence) { e.NativeKind = EditedRestartNativeUnknown },
		"changed-native":      func(e *ResumeAttemptEvidence) { e.NativeKind = EditedRestartNativeChanged },
		"replacement-found":   func(e *ResumeAttemptEvidence) { e.NativeKind = EditedRestartNativeFound; e.NativeID = "native-B" },
		"replacement-missing": func(e *ResumeAttemptEvidence) { e.NativeID = "native-B" },
		"cleanup-unknown":     func(e *ResumeAttemptEvidence) { e.Cleanup = EditedRestartCleanupUnknown },
		"occupied":            func(e *ResumeAttemptEvidence) { e.Occupancy = EditedRestartOccupancyUnknown },
		"occupancy-unknown":   func(e *ResumeAttemptEvidence) { e.Occupancy = EditedRestartOccupancyUnknown },
		"source-unknown":      func(e *ResumeAttemptEvidence) { e.Admission = EditedRestartAdmissionUnknown },
		"source-refused":      func(e *ResumeAttemptEvidence) { e.Admission = EditedRestartAdmissionUnknown },
		"operation":           func(e *ResumeAttemptEvidence) { e.OperationID = "other" },
		"address":             func(e *ResumeAttemptEvidence) { e.Address.Tag = "other" },
		"source-revision":     func(e *ResumeAttemptEvidence) { e.SourceRevision++ },
		"source-fingerprint":  func(e *ResumeAttemptEvidence) { e.SourceFingerprint = "other" },
		"attempt":             func(e *ResumeAttemptEvidence) { e.AttemptNonce = "other" },
		"claim-revision":      func(e *ResumeAttemptEvidence) { e.ClaimRevision++ },
		"cleanup-revision":    func(e *ResumeAttemptEvidence) { e.CleanupRevision-- },
		"old-record":          func(e *ResumeAttemptEvidence) { e.RecordRevision = 11; e.CleanupRevision = 11 },
		"nonce-reused":        func(e *ResumeAttemptEvidence) { e.NextNonce = e.AttemptNonce },
		"nonce-missing":       func(e *ResumeAttemptEvidence) { e.NextNonce = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			intent, event := editedRestartFixture()
			state := editedRestartResumeClaim(t, intent, event)
			missing := editedRestartMissingAttempt(event)
			mutate(&missing)
			_, request, _ := AdvanceEditedRestart(intent, state, missing)
			if request != nil {
				t.Fatalf("unsafe fresh request: %+v", request)
			}
		})
	}
}

func TestEditedRestartUnknownCleanupCanOnlyWaitForSameAttempt(t *testing.T) {
	intent, event := editedRestartFixture()
	state := editedRestartResumeClaim(t, intent, event)
	missing := editedRestartMissingAttempt(event)
	pending := missing
	pending.Cleanup = EditedRestartCleanupUnknown
	pending.Occupancy = EditedRestartOccupancyUnknown
	next, request, _ := AdvanceEditedRestart(intent, state, pending)
	if request != nil || next.Phase != EditedRestartResumeIssued {
		t.Fatal("unconfirmed attempt was replaced")
	}
	next, request, err := AdvanceEditedRestart(intent, next, missing)
	if err != nil || request == nil || next.Phase != EditedRestartFreshIssued {
		t.Fatalf("confirmed cleanup failed: %+v %v", request, err)
	}
}

func TestEditedRestartCancellationAndIntentMutation(t *testing.T) {
	for _, mutation := range []string{"cancel", "argv", "native-id", "operation", "source"} {
		t.Run(mutation, func(t *testing.T) {
			intent, event := editedRestartFixture()
			state := editedRestartResumeClaim(t, intent, event)
			if mutation == "cancel" {
				cancel := event
				cancel.Kind = EditedRestartCancelled
				state, _, _ = AdvanceEditedRestart(intent, state, cancel)
			} else {
				switch mutation {
				case "argv":
					intent.Profile.Argv[1] = "other"
				case "native-id":
					intent.ExpectedNativeID = "other"
				case "operation":
					intent.OperationID = "other"
				case "source":
					intent.SourceRevision++
				}
			}
			_, request, _ := AdvanceEditedRestart(intent, state, editedRestartMissingAttempt(event))
			if request != nil {
				t.Fatal("changed/cancelled intent launched")
			}
		})
	}
}

func TestEditedRestartEventPermutationsNeverDuplicateOrOverlap(t *testing.T) {
	// Exhaust every ordering of admission, claim, positive absence, cancellation,
	// an uncertain failure and a stale attempt; then replay the entire sequence.
	var visit func([]int, []int)
	visit = func(prefix, remaining []int) {
		if len(remaining) > 0 {
			for i, n := range remaining {
				visit(append(append([]int{}, prefix...), n), append(append([]int{}, remaining[:i]...), remaining[i+1:]...))
			}
			return
		}
		intent, admit := editedRestartFixture()
		claim := admit
		claim.Kind = EditedRestartClaimRecorded
		claim.AttemptNonce = "resume-1"
		claim.RecordRevision = 11
		claim.ClaimRevision = 11
		missing := editedRestartMissingAttempt(admit)
		cancel := admit
		cancel.Kind = EditedRestartCancelled
		uncertain := missing
		uncertain.NativeKind = EditedRestartNativeUnknown
		stale := missing
		stale.AttemptNonce = "other-attempt"
		events := []ResumeAttemptEvidence{admit, claim, missing, cancel, uncertain, stale}
		state := EditedRestartState{}
		resumes, fresh := 0, 0
		for _, index := range append(prefix, prefix...) {
			next, request, _ := AdvanceEditedRestart(intent, state, events[index])
			if request != nil {
				switch request.Shape {
				case StartColdResume:
					resumes++
					if request.RequiredNativeID != intent.ExpectedNativeID {
						t.Fatal("native ID lost")
					}
				case StartFreshExisting:
					fresh++
					if state.Phase == EditedRestartResumeIssued && (events[index].Cleanup != EditedRestartCleanupConfirmed || events[index].Occupancy != EditedRestartVacant || events[index].AttemptNonce != state.AttemptNonce || events[index].ClaimRevision != state.ClaimRevision) {
						t.Fatal("overlapping attempt")
					}
				default:
					t.Fatal("unexpected launch")
				}
				if !reflect.DeepEqual(request.Profile, intent.Profile) {
					t.Fatal("accepted profile changed")
				}
			}
			state = next
		}
		if resumes > 1 || fresh > 1 {
			t.Fatalf("duplicate emissions %v: resume=%d fresh=%d", prefix, resumes, fresh)
		}
	}
	visit(nil, []int{0, 1, 2, 3, 4, 5})
}

func TestEditedRestartAdmissionRequiresPositiveEvidence(t *testing.T) {
	for _, fault := range []string{"zero", "unknown-native", "changed-native", "wrong-id", "unbound-found", "source-unknown", "occupied", "occupancy-unknown", "old-revision", "no-nonce", "nil-argv", "selector"} {
		t.Run(fault, func(t *testing.T) {
			intent, event := editedRestartFixture()
			switch fault {
			case "zero":
				event = ResumeAttemptEvidence{}
			case "unknown-native":
				event.NativeKind = EditedRestartNativeUnknown
			case "changed-native":
				event.NativeKind = EditedRestartNativeChanged
			case "wrong-id":
				event.NativeID = "native-B"
			case "unbound-found":
				intent.ExpectedNativeID = ""
			case "source-unknown":
				event.Admission = EditedRestartAdmissionUnknown
			case "occupied":
				event.Occupancy = EditedRestartOccupancyUnknown
			case "occupancy-unknown":
				event.Occupancy = EditedRestartOccupancyUnknown
			case "old-revision":
				event.RecordRevision = 6
			case "no-nonce":
				event.NextNonce = ""
			case "nil-argv":
				intent.Profile.Argv = nil
			case "selector":
				intent.Profile.Argv = []string{"resume", "native-B"}
			}
			_, request, _ := AdvanceEditedRestart(intent, EditedRestartState{}, event)
			if request != nil {
				t.Fatalf("unconfirmed admission emitted %+v", request)
			}
		})
	}
}

func TestEditedRestartPostClaimPreHelperAbsenceRequiresRollback(t *testing.T) {
	intent, event := editedRestartFixture()
	issued, _, err := AdvanceEditedRestart(intent, EditedRestartState{}, event)
	if err != nil {
		t.Fatal(err)
	}
	missing := editedRestartMissingAttempt(event)
	if _, request, _ := AdvanceEditedRestart(intent, issued, missing); request != nil {
		t.Fatal("unrecorded claim permitted fallback")
	}
	state := editedRestartResumeClaim(t, intent, event)
	// No helper was forked. The existing owner must still positively roll back
	// its exact claim before the absence can authorize a second claim.
	missing.Cleanup = EditedRestartCleanupUnknown
	state, request, _ := AdvanceEditedRestart(intent, state, missing)
	if request != nil {
		t.Fatal("absence before claim rollback permitted fallback")
	}
	missing.Cleanup = EditedRestartCleanupConfirmed
	_, request, err = AdvanceEditedRestart(intent, state, missing)
	if err != nil || request == nil || request.Shape != StartFreshExisting {
		t.Fatalf("exact no-helper rollback: %+v %v", request, err)
	}
}

func TestEditedRestartUnknownFailureCannotLaterBecomeFallback(t *testing.T) {
	for _, kind := range []EditedRestartNativeKind{EditedRestartNativeUnknown, EditedRestartNativeChanged, EditedRestartNativeFound} {
		intent, event := editedRestartFixture()
		state := editedRestartResumeClaim(t, intent, event)
		failure := editedRestartMissingAttempt(event)
		failure.NativeKind = kind
		state, request, _ := AdvanceEditedRestart(intent, state, failure)
		if request != nil || state.Phase != EditedRestartFailed {
			t.Fatal("generic failure was not terminal")
		}
		_, request, _ = AdvanceEditedRestart(intent, state, editedRestartMissingAttempt(event))
		if request != nil {
			t.Fatal("generic failure became speculative fallback")
		}
	}
}

func TestEditedRestartAdoptionAndStaleClaimsAreInert(t *testing.T) {
	intent, event := editedRestartFixture()
	state := editedRestartResumeClaim(t, intent, event)
	stale := event
	stale.Kind = EditedRestartClaimRecorded
	stale.AttemptNonce = "other"
	stale.ClaimRevision = 99
	stale.RecordRevision = 99
	next, request, err := AdvanceEditedRestart(intent, state, stale)
	if err != nil || request != nil || next != state {
		t.Fatal("foreign claim changed issued decision")
	}
	stale.AttemptNonce = "resume-1"
	next, request, err = AdvanceEditedRestart(intent, state, stale)
	if err != nil || request != nil || next != state {
		t.Fatal("replayed claim revision replaced the pinned claim")
	}
	adopted := event
	adopted.Kind = EditedRestartAdopted
	adopted.AttemptNonce = "resume-1"
	adopted.ClaimRevision = 11
	adopted.RecordRevision = 14
	state, request, err = AdvanceEditedRestart(intent, state, adopted)
	if err != nil || request != nil || state.Phase != EditedRestartPhaseAdopted {
		t.Fatal("exact adoption not terminal")
	}
	for _, replay := range []ResumeAttemptEvidence{event, stale, editedRestartMissingAttempt(event), adopted} {
		next, request, _ = AdvanceEditedRestart(intent, state, replay)
		if request != nil || next != state {
			t.Fatal("adopted operation replay changed state")
		}
	}
}
