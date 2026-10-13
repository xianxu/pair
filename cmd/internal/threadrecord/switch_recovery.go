package threadrecord

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

type SwitchContentKind string

const (
	SwitchContentFresh    SwitchContentKind = "fresh"
	SwitchContentAuthored SwitchContentKind = "authored"
	SwitchContentFallback SwitchContentKind = "fallback"
)

type SwitchContent struct {
	Kind SwitchContentKind `json:"kind"`
	Body string            `json:"body,omitempty"`
}

func ValidateSwitchContent(c SwitchContent) error {
	switch c.Kind {
	case SwitchContentFresh:
		if c.Body == "" {
			return nil
		}
		return errors.New("switch-agent: fresh content must have an empty body")
	case SwitchContentAuthored, SwitchContentFallback:
		return orientation.ValidateBody(c.Body)
	default:
		return errors.New("switch-agent: invalid content kind")
	}
}

type SwitchPhase string

const (
	SwitchWaitingBrief    SwitchPhase = "waiting-brief"
	SwitchWaitingSettled  SwitchPhase = "waiting-settled"
	SwitchSwitching       SwitchPhase = "switching"
	SwitchDeliveryPending SwitchPhase = "delivery-pending"
	SwitchSucceeded       SwitchPhase = "succeeded"
	SwitchRefused         SwitchPhase = "refused"
	SwitchFailure         SwitchPhase = "failed"
	SwitchUnconfirmed     SwitchPhase = "unconfirmed"
)

type SwitchSurvivor string

const (
	SwitchSurvivorNone           SwitchSurvivor = ""
	SwitchSurvivorSource         SwitchSurvivor = "source"
	SwitchSurvivorTarget         SwitchSurvivor = "target"
	SwitchSurvivorRestoredSource SwitchSurvivor = "restored-source"
)

type SwitchProcess struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}
type SwitchSourceSnapshot struct {
	Process       SwitchProcess                 `json:"process"`
	Binding       *couchidentity.SessionBinding `json:"binding,omitempty"`
	NativeSession string                        `json:"native_session,omitempty"`
	Profile       LaunchProfile                 `json:"profile"`
}

// SwitchLaunchWitness retains existing start facts after registration clears
// the current incarnation's Start claim. Helper.State uses the same existing
// creating/live/unknown vocabulary; it is evidence, never a liveness assertion.
type SwitchLaunchWitness struct {
	Claim  StartClaim   `json:"claim"`
	Helper *Incarnation `json:"helper,omitempty"`
}

// SwitchRecovery is replaced only after the previous request's recovery is
// resolved. Its immutable source snapshot survives target and restoration starts.
type SwitchRecovery struct {
	RequestID       string               `json:"request_id"`
	Owner           Address              `json:"owner"`
	SourceToken     string               `json:"source_token"`
	Source          SwitchSourceSnapshot `json:"source"`
	TargetProfile   LaunchProfile        `json:"target_profile"`
	Content         SwitchContent        `json:"content"`
	Phase           SwitchPhase          `json:"phase"`
	AdmittedAt      time.Time            `json:"admitted_at"`
	BriefDeadline   time.Time            `json:"brief_deadline"`
	OverallDeadline time.Time            `json:"overall_deadline"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Target          *SwitchLaunchWitness `json:"target,omitempty"`
	Restoration     *SwitchLaunchWitness `json:"restoration,omitempty"`
	Survivor        SwitchSurvivor       `json:"survivor,omitempty"`
	Error           string               `json:"error,omitempty"`
}

func switchText(s string) bool {
	return utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func switchProfile(p LaunchProfile) error {
	if err := validateLaunchProfile(p); err != nil {
		return err
	}
	if !componentPattern.MatchString(p.Agent) {
		return errors.New("invalid switch agent")
	}
	for _, arg := range p.Argv {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return errors.New("invalid switch argument")
		}
	}
	if len(launcher.FormatLaunchParameters(p.Argv)) > 4096 {
		return errors.New("switch startup parameters exceed 4096 bytes")
	}
	return nil
}
func (r SwitchRecovery) Validate(address Address) error {
	if r.RequestID == "" || !switchText(r.RequestID) || r.SourceToken == "" || !switchText(r.SourceToken) {
		return errors.New("switch recovery needs request and source identity")
	}
	if !componentPattern.MatchString(r.Owner.RepoScope) || !componentPattern.MatchString(r.Owner.Tag) {
		return errors.New("invalid switch receipt owner")
	}
	if r.Source.Process.PID < 0 || (r.Source.Process.PID == 0) != (r.Source.Process.Identity == "") || !switchText(r.Source.Process.Identity) {
		return errors.New("invalid switch source process")
	}
	if err := validateSessionBinding(r.Source.Binding, address); err != nil {
		return err
	}
	if !switchText(r.Source.NativeSession) {
		return errors.New("invalid source native session")
	}
	if err := switchProfile(r.Source.Profile); err != nil {
		return fmt.Errorf("switch source profile: %w", err)
	}
	if err := switchProfile(r.TargetProfile); err != nil {
		return fmt.Errorf("switch target profile: %w", err)
	}
	if r.AdmittedAt.IsZero() || !r.BriefDeadline.After(r.AdmittedAt) || !r.OverallDeadline.After(r.BriefDeadline) || r.UpdatedAt.Before(r.AdmittedAt) {
		return errors.New("invalid switch recovery timestamps")
	}
	switch r.Phase {
	case SwitchWaitingBrief, SwitchWaitingSettled, SwitchSwitching, SwitchDeliveryPending, SwitchSucceeded, SwitchRefused, SwitchFailure, SwitchUnconfirmed:
	default:
		return errors.New("invalid switch phase")
	}
	if r.Content == (SwitchContent{}) {
		switch r.Phase {
		case SwitchWaitingBrief, SwitchRefused, SwitchFailure, SwitchUnconfirmed:
		default:
			return errors.New("switch phase requires selected content")
		}
	} else if err := ValidateSwitchContent(r.Content); err != nil {
		return err
	}
	if r.Phase == SwitchWaitingBrief && r.Content != (SwitchContent{}) {
		return errors.New("waiting brief already has content")
	}
	if err := validateSwitchWitness(r.Target, address, r.TargetProfile); err != nil {
		return fmt.Errorf("target witness: %w", err)
	}
	if err := validateSwitchWitness(r.Restoration, address, r.Source.Profile); err != nil {
		return fmt.Errorf("restoration witness: %w", err)
	}
	if r.Target != nil && r.Restoration != nil && r.Target.Claim.Nonce == r.Restoration.Claim.Nonce {
		return errors.New("target and restoration share attempt nonce")
	}
	registered := func(w *SwitchLaunchWitness) bool { return w != nil && w.Helper != nil && w.Helper.State == "live" }
	if (r.Phase == SwitchSucceeded || r.Phase == SwitchDeliveryPending) && !registered(r.Target) {
		return errors.New("switch result requires registered target witness")
	}
	switch r.Survivor {
	case SwitchSurvivorNone, SwitchSurvivorSource:
	case SwitchSurvivorTarget:
		if !registered(r.Target) {
			return errors.New("selected target lacks registration")
		}
	case SwitchSurvivorRestoredSource:
		if !registered(r.Restoration) {
			return errors.New("selected restored source lacks registration")
		}
	default:
		return errors.New("invalid selected switch survivor")
	}
	if r.Phase == SwitchSucceeded && r.Survivor != SwitchSurvivorNone && r.Survivor != SwitchSurvivorTarget {
		return errors.New("successful target contradicts selected survivor")
	}
	if !utf8.ValidString(r.Error) {
		return errors.New("invalid switch diagnostic")
	}
	return nil
}
func validateSwitchWitness(w *SwitchLaunchWitness, address Address, profile LaunchProfile) error {
	if w == nil {
		return nil
	}
	claim := w.Claim
	if !componentPattern.MatchString(claim.Nonce) || claim.OwnerPID <= 0 || claim.OwnerIdentity == "" || !switchText(claim.OwnerIdentity) {
		return errors.New("invalid launch claim")
	}
	if claim.LaunchProfile == nil || !reflect.DeepEqual(*claim.LaunchProfile, profile) {
		return errors.New("launch claim profile mismatch")
	}
	if err := validateSessionBinding(claim.SessionBinding, address); err != nil {
		return err
	}
	if claim.SessionBinding != nil && claim.SessionBinding.StartNonce != claim.Nonce {
		return errors.New("launch binding nonce mismatch")
	}
	if w.Helper != nil {
		h := w.Helper
		if h.PID <= 0 || h.Identity == "" || !switchText(h.Identity) || claim.SessionBinding == nil {
			return errors.New("helper requires binding and exact process identity")
		}
		switch h.State {
		case "creating", "live", "unknown":
		default:
			return errors.New("invalid helper state")
		}
		if h.Start != nil && !reflect.DeepEqual(*h.Start, claim) {
			return errors.New("helper claim mismatch")
		}
		if h.LaunchProfile != nil && !reflect.DeepEqual(*h.LaunchProfile, profile) {
			return errors.New("helper profile mismatch")
		}
	}
	return nil
}
func cloneSwitchProfile(p LaunchProfile) LaunchProfile {
	if p.Argv != nil {
		p.Argv = append([]string{}, p.Argv...)
	}
	return p
}
func cloneSwitchClaim(c StartClaim) StartClaim {
	if c.SessionBinding != nil {
		b := *c.SessionBinding
		c.SessionBinding = &b
	}
	if c.LaunchProfile != nil {
		p := cloneSwitchProfile(*c.LaunchProfile)
		c.LaunchProfile = &p
	}
	return c
}
func cloneSwitchWitness(w *SwitchLaunchWitness) *SwitchLaunchWitness {
	if w == nil {
		return nil
	}
	copy := *w
	copy.Claim = cloneSwitchClaim(w.Claim)
	if w.Helper != nil {
		h := *w.Helper
		h.DeprecatedLegacyActorID = append([]byte(nil), h.DeprecatedLegacyActorID...)
		h.DeprecatedPolicy = append([]byte(nil), h.DeprecatedPolicy...)
		if h.Start != nil {
			claim := cloneSwitchClaim(*h.Start)
			h.Start = &claim
		}
		if h.LaunchProfile != nil {
			p := cloneSwitchProfile(*h.LaunchProfile)
			h.LaunchProfile = &p
		}
		copy.Helper = &h
	}
	return &copy
}
func CloneSwitchRecovery(r *SwitchRecovery) *SwitchRecovery {
	if r == nil {
		return nil
	}
	copy := *r
	copy.Source.Profile = cloneSwitchProfile(r.Source.Profile)
	copy.TargetProfile = cloneSwitchProfile(r.TargetProfile)
	if r.Source.Binding != nil {
		b := *r.Source.Binding
		copy.Source.Binding = &b
	}
	copy.Target = cloneSwitchWitness(r.Target)
	copy.Restoration = cloneSwitchWitness(r.Restoration)
	return &copy
}
