package couchcore

import (
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type ResumeDiagnosticCode string

const (
	ResumeLive               ResumeDiagnosticCode = "resume-live"
	ResumeCreating           ResumeDiagnosticCode = "resume-creating"
	ResumeUnknown            ResumeDiagnosticCode = "resume-unknown"
	ResumeParking            ResumeDiagnosticCode = "resume-parking"
	ResumeTombstoned         ResumeDiagnosticCode = "resume-tombstoned"
	ResumeLegacyUnverified   ResumeDiagnosticCode = "resume-legacy-unverified"
	ResumePathMissing        ResumeDiagnosticCode = "resume-path-missing"
	ResumeProfileMissing     ResumeDiagnosticCode = "resume-profile-missing"
	ResumeProfileInvalid     ResumeDiagnosticCode = "resume-profile-invalid"
	ResumeAgentUnsupported   ResumeDiagnosticCode = "resume-agent-unsupported"
	ResumeBindingProvisional ResumeDiagnosticCode = "resume-binding-provisional"
	ResumeBindingAmbiguous   ResumeDiagnosticCode = "resume-binding-ambiguous"
	ResumeBindingUnbound     ResumeDiagnosticCode = "resume-binding-unbound"
	ResumeBindingRootMissing ResumeDiagnosticCode = "resume-binding-root-missing"
)

type ResumeRefusal struct {
	Code       ResumeDiagnosticCode
	Diagnostic string
}

func (e *ResumeRefusal) Error() string {
	if e == nil {
		return "resume refused"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Diagnostic)
}

func ResumeDiagnosticOf(err error) ResumeDiagnosticCode {
	var refusal *ResumeRefusal
	if errors.As(err, &refusal) {
		return refusal.Code
	}
	return ""
}

type NativeBindingResolution struct {
	Status   sessioninventory.BindingStatus
	NativeID string
}

type ResumeEligibilityInput struct {
	Thread            ThreadRecord
	WorkingPathExists bool
	Binding           NativeBindingResolution
}

type ResumeEligibility struct {
	Address           ThreadAddress
	WorkingPath       string
	Profile           LaunchProfile
	RequiredSessionID string
}

func DecideResume(input ResumeEligibilityInput) (ResumeEligibility, error) {
	record := input.Thread
	if record.Park != nil {
		return ResumeEligibility{}, refuseResume(ResumeParking, "thread has an active park transaction")
	}
	for _, state := range []struct {
		value IncarnationState
		code  ResumeDiagnosticCode
	}{
		{IncarnationLive, ResumeLive},
		{IncarnationCreating, ResumeCreating},
		{IncarnationUnknown, ResumeUnknown},
	} {
		for _, incarnation := range record.Incarnations {
			if incarnation.State == state.value {
				return ResumeEligibility{}, refuseResume(state.code, "thread still has an occupied incarnation")
			}
		}
	}
	if record.VerifiedPark == nil {
		for i := len(record.ParkHistory) - 1; i >= 0; i-- {
			if record.ParkHistory[i].Tombstoned {
				return ResumeEligibility{}, refuseResume(ResumeTombstoned, "latest park transaction was abandoned")
			}
		}
		return ResumeEligibility{}, refuseResume(ResumeLegacyUnverified, "thread has no verified park completion")
	}
	if record.WorkingPath == "" || !input.WorkingPathExists {
		return ResumeEligibility{}, refuseResume(ResumePathMissing, "saved working path is unavailable")
	}
	if record.LatestLaunchProfile == nil {
		return ResumeEligibility{}, refuseResume(ResumeProfileMissing, "thread has no successful saved launch profile")
	}
	if record.LatestLaunchProfile.Agent == "" || record.LatestLaunchProfile.Argv == nil {
		return ResumeEligibility{}, refuseResume(ResumeProfileInvalid, "saved launch profile is incomplete")
	}
	profile := cloneLaunchProfile(*record.LatestLaunchProfile)
	if !launcher.IsSupportedAgent(profile.Agent) {
		return ResumeEligibility{}, refuseResume(ResumeAgentUnsupported, "saved launch agent is unsupported")
	}
	if code := bindingResumeDiagnostic(input.Binding); code != "" {
		return ResumeEligibility{}, refuseResume(code, "native session binding is not one exact established root")
	}
	return ResumeEligibility{
		Address: record.Address, WorkingPath: record.WorkingPath,
		Profile: profile, RequiredSessionID: input.Binding.NativeID,
	}, nil
}

func bindingResumeDiagnostic(binding NativeBindingResolution) ResumeDiagnosticCode {
	switch binding.Status {
	case sessioninventory.BindingProvisional:
		return ResumeBindingProvisional
	case sessioninventory.BindingAmbiguous:
		return ResumeBindingAmbiguous
	case sessioninventory.BindingUnbound:
		return ResumeBindingUnbound
	case sessioninventory.BindingEstablished:
		if binding.NativeID == "" {
			return ResumeBindingRootMissing
		}
		return ""
	default:
		return ResumeBindingUnbound
	}
}

func refuseResume(code ResumeDiagnosticCode, diagnostic string) error {
	return &ResumeRefusal{Code: code, Diagnostic: diagnostic}
}

type NativeBindingResolver interface {
	ResolveEstablished(repoScope, tag, agent string) (NativeBindingResolution, error)
}

type SessionInventoryNativeBindingResolver struct {
	Runtime sessioninventory.Runtime
}

func (r SessionInventoryNativeBindingResolver) ResolveEstablished(repoScope, tag, agent string) (NativeBindingResolution, error) {
	if r.Runtime == nil {
		return NativeBindingResolution{}, errors.New("native binding resolver has no runtime")
	}
	query, err := sessioninventory.QuerySession(r.Runtime, repoScope, tag, sessioninventory.Agent(agent))
	if err != nil {
		return NativeBindingResolution{}, err
	}
	resolution := NativeBindingResolution{Status: query.Status}
	if query.Root != nil {
		resolution.NativeID = query.Root.NativeID
	}
	if code := bindingResumeDiagnostic(resolution); code != "" {
		return resolution, refuseResume(code, "native session binding is not one exact established root")
	}
	return resolution, nil
}

var _ NativeBindingResolver = SessionInventoryNativeBindingResolver{}
