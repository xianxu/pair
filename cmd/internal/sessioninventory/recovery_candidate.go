package sessioninventory

import (
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const RecoveryParserVersion = "recovery-v1"
const RecoveryRecordLimit = 2 << 20
const recoveryMessageWindow = 256 << 10
const recoveryMessageLimit = 256 // Also bound per-message structure overhead for tiny messages.

type InteractiveState string

const (
	InteractiveUnknown InteractiveState = "unknown"
	InteractiveYes     InteractiveState = "yes"
	InteractiveNo      InteractiveState = "no"
)

type InteractiveEvidence struct {
	State  InteractiveState
	Source string
	Reason string
}
type RecoveryMessage struct {
	Role      NativeEventKind
	Text      string
	Timestamp time.Time
	CWD       string
	Complete  bool
}

// RecoveryCandidate is evidence, never native binding proof or launch authority.
// Entry retains the collector's artifact identity/generation/mutation stamps.
type RecoveryCandidate struct {
	Agent         Agent
	NativeID      string
	Entry         FileEntry
	ParserVersion string
	RootEligible  bool
	Interactive   InteractiveEvidence
	CWD           string
	CreatedAt     time.Time
	SemanticLast  time.Time
	Messages      []RecoveryMessage
	Incomplete    bool
}

// ParseRecoveryCandidate consumes complete framed records. The collector reports
// any unframed tail separately; omitted or moving bytes cannot be inferred here.
func ParseRecoveryCandidate(agent Agent, entry FileEntry, records []FramedJSONLRecord, summary []byte) (RecoveryCandidate, []Diagnostic, error) {
	p := newRecoveryParser(agent, entry)
	for _, r := range records {
		if err := p.add(r); err != nil {
			return p.c, p.diagnostics, err
		}
	}
	return p.finish(summary)
}

type recoveryParser struct {
	c            RecoveryCandidate
	state        *ScannerState
	diagnostics  []Diagnostic
	bytesRead    int
	source       string
	seenMeta     bool
	windowBytes  int
	rootRejected bool
	cutoff       *time.Time
}

func newRecoveryParser(agent Agent, entry FileEntry) *recoveryParser {
	c := RecoveryCandidate{Agent: agent, Entry: entry, ParserVersion: RecoveryParserVersion, Interactive: InteractiveEvidence{State: InteractiveUnknown, Reason: "interactive provenance missing"}}
	if entry.BirthTime != nil {
		v := *entry.BirthTime
		c.Entry.BirthTime = &v
	}
	if entry.ModTime != nil {
		v := *entry.ModTime
		c.Entry.ModTime = &v
	}
	return &recoveryParser{c: c}
}
func (p *recoveryParser) add(r FramedJSONLRecord) error {
	c := &p.c
	agent := c.Agent
	entry := c.Entry
	p.bytesRead += len(r.Bytes)
	if len(r.Bytes) > RecoveryRecordLimit || p.bytesRead > 256<<20 {
		return ErrReadLimit
	}
	if len(r.Bytes) == 0 || strictjson.RejectDuplicateKeys(r.Bytes) != nil || !json.Valid(r.Bytes) {
		return errors.New("invalid recovery JSONL record")
	}
	var state ScannerState
	var diagnostics []Diagnostic
	var err error
	switch agent {
	case AgentCodex:
		state, diagnostics, err = ValidateCodexDelta(entry, p.state, []FramedJSONLRecord{r})
	case AgentClaude:
		state, diagnostics, err = ValidateClaudeDelta(entry, p.state, []FramedJSONLRecord{r})
	case AgentGrok:
		state, diagnostics, err = ValidateGrokDelta(entry, p.state, []FramedJSONLRecord{r})
	default:
		return errors.New("unsupported recovery parser")
	}
	p.diagnostics = append(p.diagnostics, diagnostics...)
	if err != nil {
		return err
	}
	p.state = &state
	c.NativeID = state.NativeID
	c.RootEligible = state.FirstRecordValidated && !state.Disputed && state.Role == RoleRoot && !p.rootRejected
	if state.Disputed {
		return errors.New("contradictory native root evidence")
	}
	priorCWD, priorMessages, priorBytes := c.CWD, c.Messages, p.windowBytes
	var timestamp time.Time
	var cwd string
	switch agent {
	case AgentCodex:
		var envelope struct {
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Payload   struct {
				ID     string          `json:"id"`
				Source json.RawMessage `json:"source"`
				CWD    string          `json:"cwd"`
				Parent *string         `json:"parent_thread_id"`
			} `json:"payload"`
		}
		if err = json.Unmarshal(r.Bytes, &envelope); err != nil {
			return err
		}
		timestamp, _ = time.Parse(time.RFC3339Nano, envelope.Timestamp)
		if envelope.Type == "session_meta" {
			if envelope.Payload.ID != c.NativeID {
				return errors.New("native ID disagrees with filename")
			}
			next := string(envelope.Payload.Source)
			if p.seenMeta && (p.source != next || c.CWD != envelope.Payload.CWD) {
				return errors.New("conflicting Codex session metadata")
			}
			p.seenMeta = true
			p.source = next
			cwd = envelope.Payload.CWD
			var label string
			_ = json.Unmarshal(envelope.Payload.Source, &label)
			c.Interactive = InteractiveEvidence{State: InteractiveUnknown, Source: label, Reason: "unrecognized Codex source"}
			if label == "cli" && envelope.Payload.Parent == nil && c.RootEligible {
				c.Interactive = InteractiveEvidence{State: InteractiveYes, Source: label}
			}
			if label == "exec" || label == "tool" || !c.RootEligible || envelope.Payload.Parent != nil {
				c.Interactive = InteractiveEvidence{State: InteractiveNo, Source: label, Reason: "not an interactive root"}
			}
			if c.CreatedAt.IsZero() {
				c.CreatedAt = timestamp
			}
		} else if envelope.Type == "turn_context" {
			cwd = envelope.Payload.CWD
		}
	case AgentClaude:
		var envelope struct {
			Type       string `json:"type"`
			SessionID  string `json:"sessionId"`
			Timestamp  string `json:"timestamp"`
			CWD        string `json:"cwd"`
			Entrypoint string `json:"entrypoint"`
			AgentID    string `json:"agentId"`
			Sidechain  *bool  `json:"isSidechain"`
		}
		if err = json.Unmarshal(r.Bytes, &envelope); err != nil {
			return err
		}
		timestamp, _ = time.Parse(time.RFC3339Nano, envelope.Timestamp)
		if envelope.Type == "assistant" || envelope.Type == "user" {
			cwd = envelope.CWD
			if envelope.SessionID == "" {
				c.RootEligible = false
				p.rootRejected = true
				c.Incomplete = true
			}
			next := InteractiveEvidence{State: InteractiveUnknown, Source: envelope.Entrypoint, Reason: "Claude message provenance missing"}
			if envelope.Entrypoint == "cli" && envelope.Sidechain != nil && !*envelope.Sidechain && envelope.AgentID == "" && c.RootEligible {
				next = InteractiveEvidence{State: InteractiveYes, Source: "cli"}
			}
			if envelope.Entrypoint == "sdk-cli" || envelope.AgentID != "" || envelope.Sidechain != nil && *envelope.Sidechain {
				next = InteractiveEvidence{State: InteractiveNo, Source: envelope.Entrypoint, Reason: "not an interactive root"}
			}
			if p.source != "" && p.source != string(next.State)+next.Source {
				return errors.New("conflicting Claude interactive provenance")
			}
			p.source = string(next.State) + next.Source
			c.Interactive = next
			if cwd == "" {
				c.Incomplete = true
				c.CWD = ""
				c.Messages = nil
				p.windowBytes = 0
			}
		}
		if c.CreatedAt.IsZero() && !timestamp.IsZero() {
			c.CreatedAt = timestamp
		}
	case AgentGrok:
		var envelope grokRecord
		_ = json.Unmarshal(r.Bytes, &envelope)
		if seconds, ok := grokEpochSeconds(envelope.Timestamp); ok {
			timestamp = time.Unix(seconds, 0).UTC()
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = timestamp
		}
	}
	if p.cutoff != nil && !timestamp.IsZero() && timestamp.After(*p.cutoff) {
		c.CWD = priorCWD
		c.Messages = priorMessages
		p.windowBytes = priorBytes
		return nil
	}
	if cwd != "" {
		if !strings.HasPrefix(cwd, "/") || path.Clean(cwd) != cwd {
			return errors.New("noncanonical recovery cwd")
		}
		if cwd != c.CWD {
			c.Messages = nil
			p.windowBytes = 0
		}
		c.CWD = cwd
	}
	events, disposition := NormalizeNativeEvent(agent, r.Bytes)
	if disposition == EventNearMiss {
		c.Incomplete = true
	}
	for _, event := range events {
		if event.Kind != EventAssistant && event.Kind != EventOperator || event.Text == "" {
			continue
		}
		// Do not score partial protocol chunks as complete assistant messages.
		if timestamp.IsZero() {
			c.Incomplete = true
		}
		if timestamp.After(c.SemanticLast) {
			c.SemanticLast = timestamp
		}
		if len(event.Text) > recoveryMessageWindow {
			return ErrReadLimit
		}
		c.Messages = append(c.Messages, RecoveryMessage{Role: event.Kind, Text: event.Text, Timestamp: timestamp, CWD: c.CWD, Complete: agent != AgentGrok})
		p.windowBytes += len(event.Text)
		for p.windowBytes > recoveryMessageWindow || len(c.Messages) > recoveryMessageLimit {
			p.windowBytes -= len(c.Messages[0].Text)
			c.Messages[0] = RecoveryMessage{}
			c.Messages = c.Messages[1:]
		}
	}
	return nil
}
func (p *recoveryParser) finish(summary []byte) (RecoveryCandidate, []Diagnostic, error) {
	c := p.c
	agent := c.Agent
	entry := c.Entry
	diagnostics := p.diagnostics
	var err error
	if p.state == nil || !p.state.FirstRecordValidated {
		return c, diagnostics, errors.New("missing native root records")
	}
	if agent == AgentGrok {
		if len(summary) > RecoveryRecordLimit {
			return c, diagnostics, ErrReadLimit
		}
		if len(summary) > 0 {
			if strictjson.RejectDuplicateKeys(summary) != nil {
				return c, diagnostics, errors.New("invalid Grok summary")
			}
			var s struct {
				Info struct {
					ID   string `json:"id"`
					CWD  string `json:"cwd"`
					Kind string `json:"session_kind"`
				} `json:"info"`
				Kind string `json:"session_kind"`
			}
			if err = json.Unmarshal(summary, &s); err != nil {
				return c, diagnostics, err
			}
			if s.Info.ID != "" && s.Info.ID != c.NativeID {
				return c, diagnostics, errors.New("Grok summary ID disagrees with root")
			}
			encodedCWD := strings.Split(entry.Artifact.RelativePath, "/")[0]
			pathCWD, _ := url.PathUnescape(encodedCWD)
			if s.Info.CWD != "" && s.Info.CWD != pathCWD {
				return c, diagnostics, errors.New("Grok summary cwd disagrees with path")
			}
			c.CWD = s.Info.CWD
			c.Interactive = InteractiveEvidence{State: InteractiveUnknown, Source: s.Info.Kind, Reason: "Grok has no verified interactive discriminator"}
			if s.Info.Kind == "headless" || s.Kind == "headless" {
				c.Interactive.State = InteractiveNo
			}
		}
	}
	if c.CWD == "" {
		c.Incomplete = true
	}
	return c, diagnostics, nil
}
