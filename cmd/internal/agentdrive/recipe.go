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

// RecoveryOnly grants only a bounded user-requested ESC/clear-line sequence
// access to a positively recognized occupied composer.
func (r Request) RecoveryOnly() bool {
	if r.Validate() != nil || r.Model != nil {
		return false
	}
	for _, key := range r.Keys {
		if key.Kind != KeyEsc && key.Kind != KeyClearLine {
			return false
		}
	}
	return true
}

// DismissOnly is exactly one Esc: the key a recognized panel declares as its
// own dismissal (pair#691). It is narrower than RecoveryOnly on purpose.
func (r Request) DismissOnly() bool {
	return r.Validate() == nil && r.Model == nil && len(r.Keys) == 1 && r.Keys[0] == KeyStep{Kind: KeyEsc}
}

// Admission is the wrapper's positive evidence about the current viewport.
type Admission struct{ Idle, Recoverable, Dismissable bool }

// Admits is the single admission rule. The Couch service, its pre-input
// guard, the wrapper and the reducer all call it, so no layer can admit a
// request another refuses (ARCH-DRY).
func (r Request) Admits(a Admission) bool {
	return a.Idle || r.RecoveryOnly() && a.Recoverable || r.DismissOnly() && a.Dismissable
}

// PanelRefusal is the refusal for any other request while a recognized panel
// shows: it names the one key that is admitted.
const PanelRefusal = "a slash-command panel is open; only a lone esc is admitted (couch --keys SLOT esc), then retry"

// Refusal explains why r was refused: the panel hint when a panel shows and r
// is not already the lone esc, otherwise the caller's own wording.
func (r Request) Refusal(a Admission, otherwise string) string {
	if a.Dismissable && !a.Idle && !a.Recoverable && !r.DismissOnly() {
		return PanelRefusal
	}
	return otherwise
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
