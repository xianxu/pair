package couchmessage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"path/filepath"
	"regexp"
	"unicode/utf8"
)

type DrainControlKind string

const (
	DrainNoticeControl    DrainControlKind = "notice"
	DrainInterruptControl DrainControlKind = "interrupt"
)

// DrainFence is a comparable capability bound to one namespace and incarnation.
// Zero is absence on ordinary messages; a present fence is always complete.
type DrainFence struct {
	Namespace  string
	Generation string
	Epoch      uint64
	Instance   string
	Kind       DrainControlKind
}

// UnmarshalJSON keeps generation authority closed: case aliases and partial
// objects must not override a known fence through encoding/json's field fold.
func (f *DrainFence) UnmarshalJSON(raw []byte) error {
	if len(raw) > 4096 || !utf8.Valid(raw) {
		return errors.New("invalid or oversized drain fence")
	}
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return errors.New("drain fence must be an object")
	}
	for _, key := range []string{"Namespace", "Generation", "Epoch", "Instance", "Kind"} {
		v, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return errors.New("drain fence lacks required canonical field")
		}
	}
	if len(fields) != 5 {
		return errors.New("drain fence has unknown or aliased fields")
	}
	type wire DrainFence
	var next wire
	if err := json.Unmarshal(raw, &next); err != nil {
		return err
	}
	candidate := DrainFence(next)
	if err := candidate.Validate(); err != nil {
		return err
	}
	*f = candidate
	return nil
}

var drainGenerationUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (f DrainFence) IsZero() bool { return f == DrainFence{} }
func (f DrainFence) Validate() error {
	if !validDrainNamespaceKey(f.Namespace) || !drainGenerationUUID.MatchString(f.Generation) || f.Epoch == 0 {
		return errors.New("invalid drain namespace/generation/epoch fence")
	}
	if !utf8.ValidString(f.Instance) {
		return errors.New("invalid drain wrapper instance encoding")
	}
	if err := (DriveObservation{Instance: f.Instance}).Validate(); err != nil {
		return err
	}
	if f.Kind != DrainNoticeControl && f.Kind != DrainInterruptControl {
		return errors.New("invalid drain control kind")
	}
	return nil
}
func validDrainNamespaceKey(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, c := range key {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// DrainNamespaceKey derives bounded authority from the same already canonical
// absolute namespace required by SocketPath; it performs no filesystem lookup.
func DrainNamespaceKey(namespace string) (string, error) {
	if !filepath.IsAbs(namespace) || namespace != filepath.Clean(namespace) || !utf8.ValidString(namespace) {
		return "", errors.New("drain requires a canonical absolute namespace")
	}
	digest := sha256.Sum256([]byte("couch-drain-v2\x00" + namespace))
	return hex.EncodeToString(digest[:]), nil
}

type DrainControlDisposition string

const (
	DrainControlUnknown      DrainControlDisposition = "unknown"
	DrainControlNotCommitted DrainControlDisposition = "not-committed"
	DrainControlCommitted    DrainControlDisposition = "committed"
)

type DrainGenerationEventKind string

const (
	DrainAdmitControl  DrainGenerationEventKind = "admit"
	DrainBeginInput    DrainGenerationEventKind = "begin-input"
	DrainSettleControl DrainGenerationEventKind = "settle"
	DrainRevokeControl DrainGenerationEventKind = "revoke"
)

type DrainGenerationEvent struct {
	Kind        DrainGenerationEventKind
	Fence       DrainFence
	ID          string
	Disposition DrainControlDisposition
}

type drainGenerationControl struct {
	id                              string
	begun, revoked, revocationKnown bool
	disposition                     DrainControlDisposition
}

// DrainGeneration owns a watermark and exactly two current control slots. Its
// fields cannot be mutated by the wrapper shell or old callback objects.
type DrainGeneration struct {
	namespace, instance, generation string
	epoch                           uint64
	controls                        [2]drainGenerationControl
}
type DrainControlView struct {
	ID                              string
	Kind                            DrainControlKind
	Begun, Revoked, RevocationKnown bool
	Disposition                     DrainControlDisposition
}
type DrainGenerationView struct {
	Namespace, Instance, Generation string
	Epoch                           uint64
	Controls                        [2]DrainControlView
}

func NewDrainGeneration(namespaceKey, instance string) (DrainGeneration, error) {
	if !validDrainNamespaceKey(namespaceKey) || !utf8.ValidString(instance) {
		return DrainGeneration{}, errors.New("invalid drain generation owner")
	}
	if err := (DriveObservation{Instance: instance}).Validate(); err != nil {
		return DrainGeneration{}, err
	}
	return DrainGeneration{namespace: namespaceKey, instance: instance}, nil
}
func drainControlIndex(kind DrainControlKind) int {
	if kind == DrainNoticeControl {
		return 0
	}
	return 1
}
func (s DrainGeneration) View() DrainGenerationView {
	v := DrainGenerationView{Namespace: s.namespace, Instance: s.instance, Generation: s.generation, Epoch: s.epoch}
	for i, c := range s.controls {
		kind := DrainNoticeControl
		if i == 1 {
			kind = DrainInterruptControl
		}
		v.Controls[i] = DrainControlView{ID: c.id, Kind: kind, Begun: c.begun, Revoked: c.revoked, RevocationKnown: c.revocationKnown, Disposition: c.disposition}
	}
	return v
}
func (s DrainGeneration) matches(f DrainFence) bool {
	return f.Validate() == nil && s.namespace == f.Namespace && s.instance == f.Instance && s.epoch == f.Epoch && s.generation == f.Generation
}
func (s DrainGeneration) Lookup(f DrainFence, id string) (DrainControlView, error) {
	if !s.matches(f) {
		return DrainControlView{}, errors.New("drain status fence is not current")
	}
	c := s.View().Controls[drainControlIndex(f.Kind)]
	if c.ID != id || id == "" {
		return DrainControlView{}, errors.New("unknown current drain control")
	}
	return c, nil
}
func (s DrainGeneration) Allows(f DrainFence, id string) bool {
	c, err := s.Lookup(f, id)
	return err == nil && !c.Revoked
}

// ReduceDrainGeneration grants admission or records a disposition without IO.
// Every failure returns the unchanged state. Status never calls this reducer.
func ReduceDrainGeneration(s DrainGeneration, e DrainGenerationEvent) (DrainGeneration, error) {
	original := s
	fail := func(reason string) (DrainGeneration, error) { return original, errors.New(reason) }
	if err := e.Fence.Validate(); err != nil {
		return original, err
	}
	if !validMessageID(e.ID) {
		return fail("invalid drain control ID")
	}
	if s.namespace != e.Fence.Namespace || s.instance != e.Fence.Instance {
		return fail("drain fence owner mismatch")
	}
	switch e.Kind {
	case DrainAdmitControl, DrainBeginInput, DrainSettleControl, DrainRevokeControl:
	default:
		return fail("invalid drain generation event")
	}
	if e.Kind == DrainSettleControl || e.Kind == DrainRevokeControl {
		switch e.Disposition {
		case DrainControlUnknown, DrainControlNotCommitted, DrainControlCommitted:
		default:
			return fail("invalid drain control disposition")
		}
	} else if e.Disposition != "" {
		return fail("drain admission/input carries no disposition")
	}
	if e.Fence.Epoch < s.epoch {
		return fail("retired drain epoch")
	}
	if e.Fence.Epoch == s.epoch && e.Fence.Generation != s.generation {
		return fail("drain epoch has a different generation")
	}
	if e.Fence.Epoch > s.epoch {
		if e.Kind != DrainAdmitControl && e.Kind != DrainRevokeControl {
			return fail("only admission/revocation can advance a drain epoch")
		}
		for _, c := range s.controls {
			if c.id != "" && (!c.revoked || !c.revocationKnown) {
				return fail("prior drain cancellation remains unknown")
			}
		}
		s.epoch = e.Fence.Epoch
		s.generation = e.Fence.Generation
		s.controls = [2]drainGenerationControl{}
	}
	i := drainControlIndex(e.Fence.Kind)
	c := s.controls[i]
	if c.id != "" && c.id != e.ID {
		return fail("drain control slot already occupied")
	}
	if other := s.controls[1-i]; other.id == e.ID {
		return fail("drain control ID belongs to another kind")
	}
	switch e.Kind {
	case DrainAdmitControl:
		if c.id == "" {
			c.id = e.ID
			c.disposition = DrainControlUnknown
		}
	case DrainBeginInput:
		if c.id == "" || c.revoked || c.begun {
			return fail("drain input is not authorized")
		}
		c.begun = true
		c.disposition = DrainControlUnknown
	case DrainSettleControl:
		if c.id == "" {
			return fail("drain control was not admitted")
		}
		if c.disposition != DrainControlUnknown && c.disposition != e.Disposition {
			return fail("conflicting drain input disposition")
		}
		c.disposition = e.Disposition
	case DrainRevokeControl:
		if c.id == "" {
			c.id = e.ID
			c.disposition = DrainControlNotCommitted
		}
		c.revoked = true
		if e.Disposition != DrainControlUnknown {
			if c.disposition != DrainControlUnknown && c.disposition != e.Disposition {
				return fail("conflicting drain revocation disposition")
			}
			c.disposition = e.Disposition
			c.revocationKnown = true
		}
	}
	s.controls[i] = c
	if !s.matches(e.Fence) {
		return fail(fmt.Sprintf("drain event failed current authority %d", e.Fence.Epoch))
	}
	return s, nil
}
