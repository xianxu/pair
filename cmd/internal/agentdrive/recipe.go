package agentdrive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ModelScope string

const (
	ScopeSession        ModelScope = "session"
	ScopeDefault        ModelScope = "default"
	MaxModelLabelBytes             = 128
	MaxEffortLabelBytes            = 32
	MaxResultTailBytes             = 32 * 1024
	MaxNavigationSteps             = 96
	MaxObservedLabels              = 128
)

type ModelRequest struct {
	Model  string
	Effort string     `json:",omitempty"`
	Scope  ModelScope `json:",omitempty"`
}

func (r ModelRequest) EffectiveScope() ModelScope {
	if r.Scope == "" {
		return ScopeSession
	}
	return r.Scope
}
func validLabel(s string, max int, optional bool) bool {
	return (optional || strings.TrimSpace(s) != "") && len(s) <= max && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func (r ModelRequest) Validate() error {
	if !validLabel(r.Model, MaxModelLabelBytes, false) {
		return fmt.Errorf("model must be nonempty printable UTF-8 at most %d bytes", MaxModelLabelBytes)
	}
	if !validLabel(r.Effort, MaxEffortLabelBytes, true) {
		return fmt.Errorf("effort must be printable UTF-8 at most %d bytes", MaxEffortLabelBytes)
	}
	if r.EffectiveScope() != ScopeSession && r.EffectiveScope() != ScopeDefault {
		return errors.New("scope must be session or default")
	}
	return nil
}

type Request struct {
	Keys  KeySequence   `json:",omitempty"`
	Model *ModelRequest `json:",omitempty"`
}

func (r Request) Validate() error {
	if (len(r.Keys) == 0) == (r.Model == nil) {
		return errors.New("drive requires exactly one key sequence or model request")
	}
	if r.Model != nil {
		return r.Model.Validate()
	}
	return r.Keys.Validate()
}

// IntentDigest records complete intent without retaining raw text in the
// operation receipt. Omitted session scope is canonicalized before hashing.
func (r Request) IntentDigest() string {
	if r.Model != nil {
		m := *r.Model
		m.Scope = m.EffectiveScope()
		r.Model = &m
	}
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type Outcome string

const (
	OutcomePending     Outcome = ""
	OutcomeConfirmed   Outcome = "confirmed"
	OutcomeRefused     Outcome = "refused"
	OutcomeFailed      Outcome = "failed"
	OutcomeUnconfirmed Outcome = "unconfirmed"
)

// Result names the knowledge retained after input stops. Unconfirmed never
// authorizes replay; Step names only writes acknowledged by the local writer.
type Result struct {
	ID      string `json:",omitempty"`
	Outcome Outcome
	Detail  string     `json:",omitempty"`
	Step    int        `json:",omitempty"`
	Model   string     `json:",omitempty"`
	Effort  string     `json:",omitempty"`
	Scope   ModelScope `json:",omitempty"`
	Tail    []string   `json:",omitempty"`
}

func (r Result) Terminal() bool { return r.Outcome != OutcomePending }
