package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
)

const maxPolicyResponseBytes = 1 << 20

type PolicyCapacityKind string

const (
	CapacityUnknown   PolicyCapacityKind = ""
	CapacityBounded   PolicyCapacityKind = "bounded"
	CapacityUnbounded PolicyCapacityKind = "unbounded"
)

type CapacityAction string

const (
	CapacityActionUnknown     CapacityAction = ""
	CapacityReject            CapacityAction = "reject"
	CapacityProvisionWorktree CapacityAction = "provision-worktree"
)

type PolicyCapacity struct {
	Kind  PolicyCapacityKind `json:"kind"`
	Limit int                `json:"limit,omitempty"`
}

// PolicyResult is the Pair-owned normalized evidence used by admission. It
// intentionally contains no provider declaration or admission-key-kind model.
type PolicyResult struct {
	PolicyVersion int            `json:"policy_version"`
	PolicyDigest  string         `json:"policy_digest"`
	RepoIdentity  string         `json:"repo_identity"`
	AdmissionKey  string         `json:"admission_key"`
	Capacity      PolicyCapacity `json:"capacity"`
	OnCapacity    CapacityAction `json:"on_capacity,omitempty"`
}

type PolicyDiagnostic struct {
	Code          string
	Message       string
	Path          string
	PolicyVersion *int
}

type PolicyRefusal struct {
	Diagnostic PolicyDiagnostic
}

func (e *PolicyRefusal) Error() string {
	return fmt.Sprintf("fleet policy refused: %s: %s", e.Diagnostic.Code, e.Diagnostic.Message)
}

type PolicyResolver interface {
	ResolvePolicy(context.Context, string) (PolicyResult, error)
}

type policyEnvelopeWire struct {
	OK         *bool                  `json:"ok"`
	Value      *policyResultValueWire `json:"value,omitempty"`
	Diagnostic *policyDiagnosticWire  `json:"diagnostic,omitempty"`
}

type policyResultValueWire struct {
	PolicyVersion int                `json:"policy_version"`
	PolicyDigest  string             `json:"policy_digest"`
	RepoIdentity  string             `json:"repo_identity"`
	AdmissionKey  string             `json:"admission_key"`
	Capacity      policyCapacityWire `json:"capacity"`
	OnCapacity    string             `json:"on_capacity,omitempty"`
}

type policyCapacityWire struct {
	Kind  string `json:"kind"`
	Limit *int   `json:"limit,omitempty"`
}

type policyDiagnosticWire struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Path          string `json:"path,omitempty"`
	PolicyVersion *int   `json:"policy_version,omitempty"`
}

var (
	policyDigestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	diagnosticCodePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// DecodePolicyResponse validates the provider's process disposition and JSON
// envelope as one protocol. JSON that contradicts exit/stderr is never treated
// as policy evidence.
func DecodePolicyResponse(stdout, stderr []byte, exitCode int) (PolicyResult, error) {
	if len(stdout) > maxPolicyResponseBytes || len(stderr) > maxPolicyResponseBytes {
		return PolicyResult{}, fmt.Errorf("fleet policy response exceeds %d bytes", maxPolicyResponseBytes)
	}
	var envelope policyEnvelopeWire
	if err := strictPolicyJSON(stdout, &envelope); err != nil {
		return PolicyResult{}, fmt.Errorf("decode fleet policy response: %w", err)
	}
	if envelope.OK == nil {
		return PolicyResult{}, errors.New("decode fleet policy response: missing ok discriminator")
	}
	if *envelope.OK {
		if envelope.Value == nil || envelope.Diagnostic != nil {
			return PolicyResult{}, errors.New("decode fleet policy response: success requires value only")
		}
		if exitCode != 0 || len(stderr) != 0 {
			return PolicyResult{}, fmt.Errorf("fleet policy success has exit=%d stderr=%q", exitCode, stderr)
		}
		return normalizePolicyValue(*envelope.Value)
	}
	if envelope.Value != nil || envelope.Diagnostic == nil {
		return PolicyResult{}, errors.New("decode fleet policy response: refusal requires diagnostic only")
	}
	diagnostic, err := normalizePolicyDiagnostic(*envelope.Diagnostic)
	if err != nil {
		return PolicyResult{}, err
	}
	wantStderr := []byte("Error: fleet policy refused: " + diagnostic.Code + "\n")
	if exitCode != 1 || !bytes.Equal(stderr, wantStderr) {
		return PolicyResult{}, fmt.Errorf("fleet policy refusal has exit=%d stderr=%q, want exit=1 stderr=%q", exitCode, stderr, wantStderr)
	}
	return PolicyResult{}, &PolicyRefusal{Diagnostic: diagnostic}
}

func normalizePolicyValue(value policyResultValueWire) (PolicyResult, error) {
	if value.PolicyVersion != 1 {
		return PolicyResult{}, fmt.Errorf("unsupported fleet policy version %d", value.PolicyVersion)
	}
	if !policyDigestPattern.MatchString(value.PolicyDigest) {
		return PolicyResult{}, errors.New("fleet policy digest must be 64 lowercase hexadecimal characters")
	}
	if value.RepoIdentity == "" || value.AdmissionKey == "" {
		return PolicyResult{}, errors.New("fleet policy success requires repo identity and admission key")
	}
	result := PolicyResult{
		PolicyVersion: value.PolicyVersion,
		PolicyDigest:  value.PolicyDigest,
		RepoIdentity:  value.RepoIdentity,
		AdmissionKey:  value.AdmissionKey,
	}
	switch value.Capacity.Kind {
	case string(CapacityBounded):
		if value.Capacity.Limit == nil || *value.Capacity.Limit <= 0 {
			return PolicyResult{}, errors.New("bounded fleet capacity requires a positive limit")
		}
		action := CapacityAction(value.OnCapacity)
		if action != CapacityReject && action != CapacityProvisionWorktree {
			return PolicyResult{}, fmt.Errorf("bounded fleet capacity has unsupported on_capacity %q", value.OnCapacity)
		}
		result.Capacity = PolicyCapacity{Kind: CapacityBounded, Limit: *value.Capacity.Limit}
		result.OnCapacity = action
	case string(CapacityUnbounded):
		if value.Capacity.Limit != nil || value.OnCapacity != "" {
			return PolicyResult{}, errors.New("unbounded fleet capacity forbids limit and on_capacity")
		}
		result.Capacity = PolicyCapacity{Kind: CapacityUnbounded}
	default:
		return PolicyResult{}, fmt.Errorf("unknown fleet capacity kind %q", value.Capacity.Kind)
	}
	return result, nil
}

func normalizePolicyDiagnostic(value policyDiagnosticWire) (PolicyDiagnostic, error) {
	if !diagnosticCodePattern.MatchString(value.Code) || value.Message == "" {
		return PolicyDiagnostic{}, errors.New("fleet policy diagnostic requires a safe code and message")
	}
	if value.PolicyVersion != nil && *value.PolicyVersion != 1 {
		return PolicyDiagnostic{}, fmt.Errorf("unsupported diagnostic policy version %d", *value.PolicyVersion)
	}
	return PolicyDiagnostic{Code: value.Code, Message: value.Message, Path: value.Path, PolicyVersion: value.PolicyVersion}, nil
}

func strictPolicyJSON(raw []byte, target any) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := scanJSONValueNoDuplicates(dec); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func scanJSONValueNoDuplicates(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValueNoDuplicates(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := scanJSONValueNoDuplicates(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}
