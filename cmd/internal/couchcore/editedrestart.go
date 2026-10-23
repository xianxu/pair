package couchcore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// EditedRestartNativeKind keeps absence distinct from an unavailable observation.
type EditedRestartNativeKind uint8

const (
	EditedRestartNativeUnknown EditedRestartNativeKind = iota
	EditedRestartNativeFound
	EditedRestartNativeMissing
	EditedRestartNativeChanged
)

type EditedRestartEventKind uint8

const (
	EditedRestartEventUnknown EditedRestartEventKind = iota
	EditedRestartAdmission
	EditedRestartClaimRecorded
	EditedRestartAttemptFailed
	EditedRestartAdopted
	EditedRestartCancelled
)

type EditedRestartPhase uint8

const (
	EditedRestartPrepared EditedRestartPhase = iota
	EditedRestartResumeIssued
	EditedRestartFreshIssued
	EditedRestartPhaseAdopted
	EditedRestartFailed
	EditedRestartPhaseCancelled
)

type EditedRestartAdmissionKind uint8

const (
	EditedRestartAdmissionUnknown EditedRestartAdmissionKind = iota
	EditedRestartSourceAdmitted
)

type EditedRestartOccupancy uint8

const (
	EditedRestartOccupancyUnknown EditedRestartOccupancy = iota
	EditedRestartVacant
)

type EditedRestartCleanupKind uint8

const (
	EditedRestartCleanupUnknown EditedRestartCleanupKind = iota
	EditedRestartCleanupConfirmed
)
const EditedRestartFallbackNotice = "no previous conversation found; started fresh with the new parameters"

// EditedRestartIntent is the accepted source and command, including the native
// conversation chosen at preview. Empty ExpectedNativeID permits only positive
// unbound evidence, never discovery of a different conversation at execution.
type EditedRestartIntent struct {
	OperationID       string
	Address           ThreadAddress
	SourceRevision    uint64
	SourceFingerprint string
	SourceState       ActionableThreadState
	SourceReason      ThreadReason
	Profile           LaunchProfile
	ExpectedNativeID  string
}

// EditedRestartState belongs to one synchronous switch operation. Its digest
// fences caller-owned intent slices without retaining aliases to them.
type EditedRestartState struct {
	Phase          EditedRestartPhase
	AttemptNonce   string
	ClaimRevision  uint64
	intentDigest   string
	issuedRevision uint64
}

// ResumeAttemptEvidence is produced by existing source, binding, claim and
// cleanup owners. RecordRevision is their current CAS result, not an arbitrary
// newer record. For a failed attempt, CleanupRevision must name that exact
// cleanup result and ClaimRevision must name this operation's original claim.
// Missing.NativeID identifies the absent accepted conversation; it is empty
// only when the accepted intent itself positively selected an unbound source.
type ResumeAttemptEvidence struct {
	Kind              EditedRestartEventKind
	OperationID       string
	Address           ThreadAddress
	SourceRevision    uint64
	SourceFingerprint string
	NativeKind        EditedRestartNativeKind
	NativeID          string
	Admission         EditedRestartAdmissionKind
	Occupancy         EditedRestartOccupancy
	Cleanup           EditedRestartCleanupKind
	RecordRevision    uint64
	CleanupRevision   uint64
	AttemptNonce      string
	ClaimRevision     uint64
	NextNonce         string
}

// EditedRestartLaunch requests only the existing start claim and tracked launch
// path. It conveys no independent process or cleanup authority.
type EditedRestartLaunch struct {
	Shape            StartShape
	Profile          LaunchProfile
	RequiredNativeID string
	Nonce            string
	ExpectedRevision uint64
	Notice           string
}

// AdvanceEditedRestart consumes a launch request before its caller performs the
// existing claim CAS and tracked launch. State is request-local; reconciliation
// of an interrupted claim never reconstructs permission for a fresh fallback.
func AdvanceEditedRestart(intent EditedRestartIntent, state EditedRestartState, event ResumeAttemptEvidence) (EditedRestartState, *EditedRestartLaunch, error) {
	refuse := func(reason string) (EditedRestartState, *EditedRestartLaunch, error) {
		state.Phase = EditedRestartFailed
		return state, nil, errors.New("edited restart: " + reason)
	}
	switch state.Phase {
	case EditedRestartPhaseAdopted, EditedRestartFailed, EditedRestartPhaseCancelled:
		return state, nil, nil
	case EditedRestartPrepared, EditedRestartResumeIssued, EditedRestartFreshIssued:
	default:
		return refuse("unknown decision state")
	}
	if intent.OperationID == "" || intent.SourceRevision == 0 || intent.SourceFingerprint == "" || intent.Profile.Argv == nil {
		return refuse("accepted intent is incomplete")
	}
	if err := validateThreadAddress(intent.Address); err != nil {
		return refuse("accepted address is invalid")
	}
	if !launcher.IsSupportedAgent(intent.Profile.Agent) {
		return refuse("accepted agent is unsupported")
	}
	if err := launcher.ValidateFreshAgentArgs(intent.Profile.Agent, intent.Profile.Argv); err != nil {
		return refuse("accepted arguments contain native selectors")
	}
	raw, _ := json.Marshal(intent)
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	if state.intentDigest != "" && state.intentDigest != digest {
		return refuse("accepted intent changed")
	}
	if state.Phase != EditedRestartPrepared && (state.intentDigest == "" || state.AttemptNonce == "" || state.issuedRevision == 0) {
		return refuse("issued request identity is incomplete")
	}
	// Foreign and stale events do not consume the current operation's decision.
	if event.OperationID != intent.OperationID || event.Address != intent.Address || event.SourceRevision != intent.SourceRevision || event.SourceFingerprint != intent.SourceFingerprint {
		return state, nil, nil
	}
	if event.Kind == EditedRestartCancelled {
		state.Phase = EditedRestartPhaseCancelled
		return state, nil, nil
	}
	issue := func(shape StartShape) (EditedRestartState, *EditedRestartLaunch, error) {
		if event.Admission != EditedRestartSourceAdmitted || event.Occupancy != EditedRestartVacant || event.RecordRevision < intent.SourceRevision || event.NextNonce == "" {
			return refuse("fresh source admission and an unused attempt nonce are required")
		}
		request := &EditedRestartLaunch{Shape: shape, Profile: cloneLaunchProfile(intent.Profile), Nonce: event.NextNonce, ExpectedRevision: event.RecordRevision}
		state.intentDigest, state.AttemptNonce, state.issuedRevision, state.ClaimRevision = digest, event.NextNonce, event.RecordRevision, 0
		if shape == StartColdResume {
			state.Phase = EditedRestartResumeIssued
			request.RequiredNativeID = intent.ExpectedNativeID
		} else {
			state.Phase = EditedRestartFreshIssued
			request.Notice = EditedRestartFallbackNotice
		}
		return state, request, nil
	}
	switch event.Kind {
	case EditedRestartAdmission:
		if state.Phase != EditedRestartPrepared {
			return state, nil, nil
		}
		switch event.NativeKind {
		case EditedRestartNativeFound:
			if intent.ExpectedNativeID == "" || event.NativeID != intent.ExpectedNativeID {
				return refuse("native conversation changed")
			}
			return issue(StartColdResume)
		case EditedRestartNativeMissing:
			if event.NativeID != intent.ExpectedNativeID {
				return refuse("absence belongs to a different native conversation")
			}
			return issue(StartFreshExisting)
		default:
			return refuse("native conversation is unconfirmed")
		}
	case EditedRestartClaimRecorded:
		if state.Phase == EditedRestartPrepared || event.AttemptNonce != state.AttemptNonce {
			return state, nil, nil
		}
		if state.ClaimRevision != 0 {
			return state, nil, nil
		}
		if event.ClaimRevision != event.RecordRevision || event.ClaimRevision <= state.issuedRevision {
			return refuse("start claim revision is unconfirmed")
		}
		state.ClaimRevision = event.ClaimRevision
		return state, nil, nil
	case EditedRestartAttemptFailed:
		if state.Phase == EditedRestartPrepared || event.AttemptNonce != state.AttemptNonce || event.ClaimRevision != state.ClaimRevision {
			return state, nil, nil
		}
		if state.Phase == EditedRestartFreshIssued {
			return refuse("fresh attempt failed")
		}
		if event.NativeKind != EditedRestartNativeMissing || event.NativeID != intent.ExpectedNativeID {
			return refuse("failed resume did not prove the accepted conversation absent")
		}
		// A positive absence is not process cleanup. Leave the existing attempt
		// occupied until its owner positively confirms release of this exact claim.
		if state.ClaimRevision == 0 || event.Cleanup != EditedRestartCleanupConfirmed || event.Occupancy != EditedRestartVacant {
			return state, nil, nil
		}
		if event.CleanupRevision != event.RecordRevision || event.CleanupRevision <= state.ClaimRevision {
			return refuse("exact cleanup revision is unconfirmed")
		}
		if event.NextNonce == state.AttemptNonce {
			return refuse("fallback must use a different attempt nonce")
		}
		return issue(StartFreshExisting)
	case EditedRestartAdopted:
		if state.Phase != EditedRestartPrepared && state.ClaimRevision != 0 && event.AttemptNonce == state.AttemptNonce && event.ClaimRevision == state.ClaimRevision && event.RecordRevision >= state.ClaimRevision {
			state.Phase = EditedRestartPhaseAdopted
		}
		return state, nil, nil
	default:
		// An untagged observation is never positive evidence or fallback permission.
		return state, nil, nil
	}
}
