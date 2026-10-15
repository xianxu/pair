package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"slices"
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

const NativeAdmissionEnv = "PAIR_NATIVE_ADMISSION"
const maxNativeAdmissions = 256
const maxNativeAdmissionBytes = 1 << 20
const maxNativeAdmissionGrantBytes = 64 << 10

type NativeAdmissionRequest struct {
	Agent, NativeID, Attempt, CommandDigest string
	CodexScope                              string `json:",omitempty"`
}

// NativeAdmissionClaim is an opaque exact wrapper/attempt CAS, not a lease.
type NativeAdmissionClaim struct {
	grant   nativeAdmissionGrant
	wrapper storagegc.ProcessIdentity
	home    string
}
type nativeAdmissionGrant struct {
	Version int
	Request NativeAdmissionRequest
	Helper  storagegc.ProcessIdentity
	Token   string
}

// NativeAdmissionOwners is an immutable view valid only during Reserve's check.
// No capability token is exposed to the observer.
type NativeAdmissionOwners struct {
	entries []NativeAdmissionEntry
	process func(int) (storagegc.ProcessIdentity, int, error)
	active  *atomic.Bool
}

type NativeAdmissionPhase string

const (
	NativeAdmissionReserved NativeAdmissionPhase = "reserved"
	NativeAdmissionSpawning NativeAdmissionPhase = "spawning"
	NativeAdmissionRunning  NativeAdmissionPhase = "running"
)

type NativeAdmissionEntry struct {
	Request                NativeAdmissionRequest
	Token                  string
	Phase                  NativeAdmissionPhase
	Helper, Wrapper, Child storagegc.ProcessIdentity
}
type NativeAdmissionTransition string

const (
	NativeAdmissionReserve NativeAdmissionTransition = "reserve"
	NativeAdmissionSpawn   NativeAdmissionTransition = "spawn"
	NativeAdmissionPublish NativeAdmissionTransition = "publish"
	NativeAdmissionCancel  NativeAdmissionTransition = "cancel-before-handoff"
	NativeAdmissionCollect NativeAdmissionTransition = "collect-both-dead"
	NativeAdmissionFence   NativeAdmissionTransition = "fence-dead-helper-reservation"
)

type NativeAdmissionChange struct {
	Transition                     NativeAdmissionTransition
	Request                        NativeAdmissionRequest
	Token                          string
	Helper, Process                storagegc.ProcessIdentity
	WrapperLiveness, ChildLiveness storagegc.Liveness
	HelperLiveness                 storagegc.Liveness
}

func admissionText(s string, max int) bool {
	return s != "" && len(s) <= max && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func validAdmissionProcess(p storagegc.ProcessIdentity) bool {
	return p.PID > 0 && p.PID <= 1<<31-1 && admissionText(p.Birth, 1024)
}
func validAdmissionDigest(s string) bool {
	raw, e := hex.DecodeString(s)
	return e == nil && len(raw) == sha256.Size && s == strings.ToLower(s)
}
func validateAdmissionRequest(r NativeAdmissionRequest) error {
	if r.CodexScope != "" {
		if r.Agent != "codex" {
			return errors.New("invalid native admission scope agent")
		}
		if _, err := decodeNativeAdmissionCodexScope(r.CodexScope); err != nil {
			return err
		}
	}
	if !slices.Contains(AgentInventory(), r.Agent) || !admissionText(r.NativeID, 4096) || !admissionText(r.Attempt, 1024) || !validAdmissionDigest(r.CommandDigest) {
		return errors.New("invalid native admission request")
	}
	return nil
}
func validateAdmissionEntry(e NativeAdmissionEntry) error {
	if err := validateAdmissionRequest(e.Request); err != nil {
		return err
	}
	if !validAdmissionDigest(e.Token) || !validAdmissionProcess(e.Helper) {
		return errors.New("invalid native admission identity")
	}
	zero := storagegc.ProcessIdentity{}
	switch e.Phase {
	case NativeAdmissionReserved:
		if e.Wrapper != zero || e.Child != zero {
			return errors.New("reserved admission has writer")
		}
	case NativeAdmissionSpawning:
		if !validAdmissionProcess(e.Wrapper) || e.Child != zero {
			return errors.New("invalid spawning admission")
		}
	case NativeAdmissionRunning:
		if !validAdmissionProcess(e.Wrapper) || !validAdmissionProcess(e.Child) || e.Wrapper.PID == e.Child.PID {
			return errors.New("invalid running admission")
		}
	default:
		return errors.New("unknown native admission phase")
	}
	return nil
}

// AdvanceNativeAdmission is the sole pure mutation authority. Absence is nil;
// observations never infer death from clocks, readiness or parent disappearance.
func AdvanceNativeAdmission(current *NativeAdmissionEntry, c NativeAdmissionChange) (*NativeAdmissionEntry, error) {
	if c.Transition == NativeAdmissionReserve {
		if current != nil {
			return nil, errors.New("native conversation already reserved")
		}
		next := NativeAdmissionEntry{Request: c.Request, Token: c.Token, Helper: c.Helper, Phase: NativeAdmissionReserved}
		if err := validateAdmissionEntry(next); err != nil {
			return nil, err
		}
		return &next, nil
	}
	if current == nil {
		return nil, errors.New("native admission unavailable or fenced")
	}
	if err := validateAdmissionEntry(*current); err != nil {
		return nil, err
	}
	next := *current
	if c.Transition == NativeAdmissionCollect {
		if next.Phase != NativeAdmissionRunning || c.WrapperLiveness != storagegc.ProcessDead || c.ChildLiveness != storagegc.ProcessDead {
			return nil, errors.New("native writer death unproved")
		}
		return nil, nil
	}
	if next.Request != c.Request || next.Token != c.Token || next.Helper != c.Helper {
		return nil, errors.New("native admission attempt changed")
	}
	switch c.Transition {
	case NativeAdmissionFence:
		// Claim must consume this exact capability before a wrapper may spawn.
		// Revocation under that same lock therefore fences every delayed Claim;
		// helper death never releases an already-consumed spawning admission.
		if next.Phase != NativeAdmissionReserved || c.HelperLiveness != storagegc.ProcessDead {
			return nil, errors.New("unconsumed native reservation death unproved")
		}
		return nil, nil
	case NativeAdmissionSpawn:
		if next.Phase != NativeAdmissionReserved || !validAdmissionProcess(c.Process) {
			return nil, errors.New("native admission already consumed or wrapper unavailable")
		}
		next.Phase = NativeAdmissionSpawning
		next.Wrapper = c.Process
	case NativeAdmissionPublish:
		if !validAdmissionProcess(c.Process) || c.Process.PID == next.Wrapper.PID {
			return nil, errors.New("native child identity unavailable")
		}
		if next.Phase == NativeAdmissionRunning && next.Child == c.Process {
			return &next, nil
		}
		if next.Phase != NativeAdmissionSpawning {
			return nil, errors.New("native child publication changed")
		}
		next.Phase = NativeAdmissionRunning
		next.Child = c.Process
	case NativeAdmissionCancel:
		if next.Phase != NativeAdmissionReserved || c.Process != next.Helper {
			return nil, errors.New("cannot cancel exposed native admission")
		}
		return nil, nil
	default:
		return nil, errors.New("unknown native admission transition")
	}
	if err := validateAdmissionEntry(next); err != nil {
		return nil, err
	}
	return &next, nil
}

// NativeAdmissionCommandDigest binds the exact final native argv, including its
// executable and one-time prompt. It never hashes shell-rendered arguments.
func NativeAdmissionCommandDigest(agent string, argv []string) (string, error) {
	if !slices.Contains(AgentInventory(), agent) || len(argv) == 0 || argv[0] == "" {
		return "", errors.New("invalid native invocation")
	}
	for _, arg := range argv {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return "", errors.New("invalid native invocation argument")
		}
	}
	raw, err := json.Marshal(struct {
		Version int
		Agent   string
		Argv    []string
	}{1, agent, argv})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func admissionChange(g nativeAdmissionGrant, kind NativeAdmissionTransition, p storagegc.ProcessIdentity) NativeAdmissionChange {
	return NativeAdmissionChange{Transition: kind, Request: g.Request, Token: g.Token, Helper: g.Helper, Process: p}
}
func sameAdmissionKey(a, b NativeAdmissionRequest) bool {
	return a.Agent == b.Agent && a.NativeID == b.NativeID
}

// EncodeNativeAdmissionCodexScope retains only typed selection metadata. The
// canonical encoding stays comparable inside the existing admission request.
func EncodeNativeAdmissionCodexScope(scope sessioninventory.CodexSelectionScope) (string, error) {
	if err := validateNativeAdmissionCodexScope(scope); err != nil {
		return "", err
	}
	raw, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	if len(raw) > 8192 {
		return "", errors.New("native admission Codex scope too large")
	}
	return string(raw), nil
}
func validateNativeAdmissionCodexScope(s sessioninventory.CodexSelectionScope) error {
	validPath := func(p string) bool { return admissionText(p, 4096) && filepath.IsAbs(p) && filepath.Clean(p) == p }
	if !validPath(s.CodexHome) || !validPath(s.SQLiteHome) || !validPath(s.CWD) || !admissionText(s.Provider, 256) || (s.Mode != sessioninventory.CodexModeTUI && s.Mode != sessioninventory.CodexModeExec) || len(s.CWDs) > 64 {
		return errors.New("invalid native admission Codex scope")
	}
	for _, p := range s.CWDs {
		if !validPath(p) {
			return errors.New("invalid native admission Codex cwd scope")
		}
	}
	return nil
}
func decodeNativeAdmissionCodexScope(encoded string) (sessioninventory.CodexSelectionScope, error) {
	var scope sessioninventory.CodexSelectionScope
	if len(encoded) > 8192 {
		return scope, errors.New("native admission Codex scope too large")
	}
	if err := strictjson.Decode([]byte(encoded), &scope); err != nil {
		return scope, errors.New("invalid native admission Codex scope")
	}
	canonical, err := EncodeNativeAdmissionCodexScope(scope)
	if err != nil || canonical != encoded {
		return scope, errors.New("invalid native admission Codex scope")
	}
	return scope, nil
}

// NativeAdmissionScopedCommandDigest extends the legacy digest only when a
// validated Codex namespace is present. Scope cannot downgrade to legacy argv.
func NativeAdmissionScopedCommandDigest(agent string, argv []string, encodedScope string) (string, error) {
	legacy, err := NativeAdmissionCommandDigest(agent, argv)
	if err != nil || encodedScope == "" {
		return legacy, err
	}
	if agent != "codex" {
		return "", errors.New("invalid native admission scope agent")
	}
	if _, err := decodeNativeAdmissionCodexScope(encodedScope); err != nil {
		return "", err
	}
	raw, err := json.Marshal(struct {
		Version                   int
		CommandDigest, CodexScope string
	}{2, legacy, encodedScope})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
