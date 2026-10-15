package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// RecoveryAuthority is a locator and expected snapshot of one durable explicit
// choice. It is not sufficient authority until Pair rechecks that start claim.
type RecoveryAuthority struct {
	Version        int                         `json:"version"`
	Address        threadrecord.Address        `json:"address"`
	Attempt        string                      `json:"attempt"`
	RecordRevision uint64                      `json:"record_revision"`
	RecordPath     string                      `json:"record_path"`
	Intent         threadrecord.RecoveryIntent `json:"intent"`
}

func (a RecoveryAuthority) Validate() error {
	if a.Version != 1 || a.RecordRevision == 0 || ValidateRepoScopeKey(a.Address.RepoScope) != nil || ValidatePairTag(a.Address.Tag) != nil || !scopedComponentPattern.MatchString(a.Attempt) || len(a.Attempt) > 1024 {
		return errors.New("invalid recovery authority identity")
	}
	if len(a.RecordPath) > 4096 || !utf8.ValidString(a.RecordPath) || strings.IndexFunc(a.RecordPath, unicode.IsControl) >= 0 || !filepath.IsAbs(a.RecordPath) || filepath.Clean(a.RecordPath) != a.RecordPath {
		return errors.New("invalid recovery record locator")
	}
	return threadrecord.ValidateRecoveryIntent(a.Intent, a.Address)
}
func cloneRecoveryAuthority(a *RecoveryAuthority) *RecoveryAuthority {
	if a == nil {
		return nil
	}
	out := *a
	out.Intent = *threadrecord.CloneRecoveryIntent(&a.Intent)
	return &out
}
func BuildCouchRecoveryLaunchProfile(tag, agent string, argv []string, a RecoveryAuthority) (string, error) {
	p := TrustedLaunchProfile{SchemaVersion: 1, Tag: tag, Agent: agent, Argv: append([]string{}, argv...), Recovery: cloneRecoveryAuthority(&a), RequiredSessionID: a.Intent.NativeID}
	if a.Intent.NativeID == "" {
		p.FreshRequired = true
		p.FreshNonce = a.Attempt
		p.AgentSource = "explicit"
		p.ArgvSource = "explicit"
	} else {
		p.ResumeRequired = true
		p.AgentSource = "saved"
		p.ArgvSource = "saved"
	}
	if err := ValidateTrustedLaunchProfile(p); err != nil {
		return "", err
	}
	raw, err := json.Marshal(p)
	return string(raw), err
}

// RecoveryAuthorityRuntime is optional; a runtime without the complete second
// gate cannot execute recovery, and never falls back to ordinary or fresh start.
type RecoveryAuthorityRuntime interface {
	ValidateRecoveryAuthority(context.Context, RecoveryAuthority, RepoScope, string, string, string) error
}

func validateRecoveryProfile(p TrustedLaunchProfile) error {
	a := p.Recovery
	if a == nil {
		return nil
	}
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Address.Tag != p.Tag || p.RequiredSessionID != a.Intent.NativeID || (a.Intent.NativeID == "") != p.FreshRequired || (a.Intent.NativeID != "") != p.ResumeRequired {
		return errors.New("recovery profile does not match explicit intent")
	}
	if p.FreshRequired && p.FreshNonce != a.Attempt {
		return errors.New("recovery fresh nonce mismatch")
	}
	if a.Intent.NativeID != "" {
		for _, c := range a.Intent.Context.Candidates {
			if c.Key == a.Intent.SelectedKey && c.Agent != p.Agent {
				return errors.New("recovery candidate agent mismatch")
			}
		}
	}
	return nil
}
