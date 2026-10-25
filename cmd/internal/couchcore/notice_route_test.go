package couchcore

import "testing"

func TestNoticeRouteExhaustive(t *testing.T) {
	direct := map[NoticeClass]bool{NoticeHubRequested: true, NoticePriorCouchCrash: true, NoticeHubStartFailed: true}
	operator := map[NoticeClass]bool{NoticeCastStartFailed: true, NoticeCastEndedUnexpectedly: true, NoticeChromeStartFailed: true, NoticeChromeEndedUnexpectedly: true, NoticeChromeRecovery: true, NoticeCastDiagnostic: true, NoticeCastCodeLockout: true, NoticeOperatorCapabilityFailed: true, NoticeOperatorDriveFailed: true}
	classes := []NoticeClass{NoticeHubRequested, NoticeCastStartFailed, NoticeCastEndedUnexpectedly, NoticeChromeStartFailed, NoticeChromeEndedUnexpectedly, NoticeChromeRecovery, NoticeCastDiagnostic, NoticeCastCodeLockout, NoticeOperatorCapabilityFailed, NoticeOperatorDriveFailed, NoticeChildExited, NoticeExitCleanupFailed, NoticeInputTraceFailed, NoticeMouseTraceFailed, NoticeEventTraceFailed, NoticeContinuationScannerFailed, NoticeContinuationMissed, NoticeContinuationFailed, NoticeLifecycleAttemptDiagnostic, NoticeLifecycleResultWarning, NoticeRemoteLifecycleDiagnostic, NoticeDispatcherDiagnostic, NoticeFailedAttachCleanup, NoticeStartupConfigurationWarning, NoticeHubStartFailed, NoticeHubClaimConflict, NoticeAgentPriorityWarning, NoticeScratchMetadataWarning, NoticeSupervisionEscalation, NoticeStartupResendUnavailable, NoticeWorkerCueUnavailable, NoticeAutomaticInputStall, NoticeLifecyclePublisherDiagnostic, NoticeArchiveInventoryWarning, NoticeArchiveMaintenanceWarning, NoticePriorCouchCrash, NoticeCrashCaptureSetupFailed, NoticeLivePicker, NoticeJudgedQuestion, NoticeRoutine, NoticeUnknown, "", "future"}
	seen := map[NoticeClass]bool{}
	for _, class := range classes {
		if seen[class] {
			t.Fatalf("duplicate class %q", class)
		}
		seen[class] = true
		for _, origin := range []NoticeOrigin{NoticeSystemOrigin, NoticeOperatorOrigin, 255} {
			for _, owner := range []NoticeOwnerState{NoticeOwnerBooting, NoticeOwnerEnabled, NoticeOwnerDisabled, NoticeOwnerUnavailable} {
				for _, handled := range []bool{false, true} {
					got := RouteNotice(NoticeRouteInput{Class: class, Origin: origin, Owner: owner, HubHandled: handled})
					routine := class == NoticeRoutine
					explicit := direct[class] || operator[class] && origin == NoticeOperatorOrigin
					want := NoticeRoute{Feed: true, Bell: !routine && (explicit || owner == NoticeOwnerUnavailable), Hub: !routine && !explicit && owner == NoticeOwnerEnabled && !handled, Defer: !routine && !explicit && owner == NoticeOwnerBooting}
					if got != want {
						t.Fatalf("class=%s origin=%d owner=%d handled=%t: %+v want %+v", class, origin, owner, handled, got, want)
					}
				}
			}
		}
	}
}

func FuzzNoticeRouteUnknown(f *testing.F) {
	f.Add("future", uint8(0), uint8(0), false)
	f.Add("", uint8(255), uint8(255), true)
	f.Fuzz(func(t *testing.T, class string, origin, owner uint8, handled bool) {
		// Prefixing creates a class outside the finite producer domain.
		got := RouteNotice(NoticeRouteInput{Class: NoticeClass("future:" + class), Origin: NoticeOrigin(origin), Owner: NoticeOwnerState(owner), HubHandled: handled})
		if !got.Feed || got.Bell != (NoticeOwnerState(owner) == NoticeOwnerUnavailable) || got.Hub != (NoticeOwnerState(owner) == NoticeOwnerEnabled && !handled) {
			t.Fatalf("unknown class acquired privilege: %+v", got)
		}
	})
}
