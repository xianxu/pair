package couchcore

// NoticeClass identifies a producer, independently of display text and feed kind.
type NoticeClass string

const (
	NoticeUnknown                      NoticeClass = "unknown"
	NoticeRoutine                      NoticeClass = "routine"
	NoticeHubRequested                 NoticeClass = "hub-requested"
	NoticeCastStartFailed              NoticeClass = "cast-start-failed"
	NoticeCastEndedUnexpectedly        NoticeClass = "cast-ended-unexpectedly"
	NoticeChromeStartFailed            NoticeClass = "chrome-start-failed"
	NoticeChromeEndedUnexpectedly      NoticeClass = "chrome-ended-unexpectedly"
	NoticeChromeRecovery               NoticeClass = "chrome-recovery"
	NoticeCastDiagnostic               NoticeClass = "cast-diagnostic"
	NoticeCastCodeLockout              NoticeClass = "cast-code-lockout"
	NoticeOperatorCapabilityFailed     NoticeClass = "operator-capability-failed"
	NoticeOperatorDriveFailed          NoticeClass = "operator-drive-failed"
	NoticeChildExited                  NoticeClass = "child-exited"
	NoticeExitCleanupFailed            NoticeClass = "exit-cleanup-failed"
	NoticeInputTraceFailed             NoticeClass = "input-trace-failed"
	NoticeMouseTraceFailed             NoticeClass = "mouse-trace-failed"
	NoticeEventTraceFailed             NoticeClass = "event-trace-failed"
	NoticeContinuationScannerFailed    NoticeClass = "continuation-scanner-failed"
	NoticeContinuationMissed           NoticeClass = "continuation-missed"
	NoticeContinuationFailed           NoticeClass = "continuation-failed"
	NoticeLifecycleAttemptDiagnostic   NoticeClass = "lifecycle-attempt-diagnostic"
	NoticeLifecycleResultWarning       NoticeClass = "lifecycle-result-warning"
	NoticeRemoteLifecycleDiagnostic    NoticeClass = "remote-lifecycle-diagnostic"
	NoticeDispatcherDiagnostic         NoticeClass = "dispatcher-diagnostic"
	NoticeFailedAttachCleanup          NoticeClass = "failed-attach-cleanup"
	NoticeStartupConfigurationWarning  NoticeClass = "startup-configuration-warning"
	NoticeHubStartFailed               NoticeClass = "hub-start-failed"
	NoticeHubClaimConflict             NoticeClass = "hub-claim-conflict"
	NoticeAgentPriorityWarning         NoticeClass = "agent-priority-warning"
	NoticeScratchMetadataWarning       NoticeClass = "scratch-metadata-warning"
	NoticeSupervisionEscalation        NoticeClass = "supervision-escalation"
	NoticeStartupResendUnavailable     NoticeClass = "startup-resend-unavailable"
	NoticeWorkerCueUnavailable         NoticeClass = "worker-cue-unavailable"
	NoticeAutomaticInputStall          NoticeClass = "automatic-input-stall"
	NoticeLifecyclePublisherDiagnostic NoticeClass = "lifecycle-publisher-diagnostic"
	NoticeArchiveInventoryWarning      NoticeClass = "archive-inventory-warning"
	NoticeArchiveMaintenanceWarning    NoticeClass = "archive-maintenance-warning"
	NoticePriorCouchCrash              NoticeClass = "prior-couch-crash"
	NoticeCrashCaptureSetupFailed      NoticeClass = "crash-capture-setup-failed"
	NoticeLivePicker                   NoticeClass = "live-picker"
	NoticeJudgedQuestion               NoticeClass = "judged-question"
)

type NoticeOrigin uint8

const (
	NoticeSystemOrigin NoticeOrigin = iota
	NoticeOperatorOrigin
)

type NoticeOwnerState uint8

const (
	NoticeOwnerBooting NoticeOwnerState = iota
	NoticeOwnerEnabled
	NoticeOwnerDisabled
	NoticeOwnerUnavailable
)

type NoticeRouteInput struct {
	Class      NoticeClass
	Origin     NoticeOrigin
	Owner      NoticeOwnerState
	HubHandled bool
}

type NoticeRoute struct{ Feed, Bell, Hub, Defer bool }

// RouteNotice grants immediate human attention only to explicit exceptions or
// operator-owned operations. Unknown classes follow owning-hub routing.
func RouteNotice(in NoticeRouteInput) NoticeRoute {
	out := NoticeRoute{Feed: true}
	switch in.Class {
	case NoticeRoutine:
		return out
	case NoticeHubRequested, NoticePriorCouchCrash, NoticeHubStartFailed:
		out.Bell = true
		return out
	case NoticeCastStartFailed, NoticeCastEndedUnexpectedly, NoticeChromeStartFailed, NoticeChromeEndedUnexpectedly, NoticeChromeRecovery, NoticeCastDiagnostic, NoticeCastCodeLockout, NoticeOperatorCapabilityFailed, NoticeOperatorDriveFailed:
		if in.Origin == NoticeOperatorOrigin {
			out.Bell = true
			return out
		}
	}
	switch in.Owner {
	case NoticeOwnerEnabled:
		out.Hub = !in.HubHandled
	case NoticeOwnerUnavailable:
		out.Bell = true
	case NoticeOwnerDisabled:
	default:
		out.Defer = true // Unrecognized owner readiness cannot prove absence.
	}
	return out
}
