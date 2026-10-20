package couchmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultPeerBatchMaxChars = 16384
	MaxPeerBatchChars        = 32768
	peerBatchSeparator       = "\n\n---\n\n"
)

// Envelope preserves the provenance header and original body of one message.
func Envelope(m Message) string {
	switch m.Origin {
	case OriginSystem:
		return fmt.Sprintf("[Couch system; delivery %s]\n%s", m.ID, m.Body)
	case OriginPeer:
	default:
		return ""
	}
	return fmt.Sprintf("[Couch peer from %s; delivery %s]\n%s", m.From.Slot, m.ID, m.Body)
}

// BatchEnvelope keeps every member's envelope intact and visibly separated.
func BatchEnvelope(messages []Message) string {
	envelopes := make([]string, len(messages))
	for i, m := range messages {
		envelopes[i] = Envelope(m)
	}
	return strings.Join(envelopes, peerBatchSeparator)
}

// SelectBatch takes a contiguous FIFO prefix. The head always proceeds alone
// when its envelope exceeds the configured character cap.
func SelectBatch(head Message, queued []Message, maxChars int) []Message {
	selected := []Message{head}
	for _, m := range queued {
		if len(selected) >= MaxQueued+1 {
			break
		}
		candidate := append(append([]Message(nil), selected...), m)
		if utf8.RuneCountInString(BatchEnvelope(candidate)) > maxChars || !batchFinalizationFits(candidate) {
			break
		}
		selected = candidate
	}
	return selected
}

// batchFinalizationFits measures the actual endpoint JSON, including repeated
// escaped bindings. AcceptedAt belongs to broker admission, not the wire commit.
func batchFinalizationFits(messages []Message) bool {
	if len(messages) == 0 {
		return false
	}
	wire := append([]Message(nil), messages...)
	for i := range wire {
		wire[i].AcceptedAt = time.Time{}
	}
	raw, err := json.Marshal(EndpointRequest{Op: "commit-batch", Binding: wire[0].To, ID: wire[0].ID, Messages: wire})
	return err == nil && len(raw) <= MaxFrameBytes
}

// ValidateBatch accepts old nonzero exchange deadlines: prepared mail may have
// waited for its connected receiver long after its admission exchange finished.
func ValidateBatch(messages []Message, to Binding, id string) error {
	if len(messages) == 0 || len(messages) > MaxQueued+1 {
		return errors.New("batch member count exceeds limit")
	}
	if err := to.Validate(); err != nil {
		return err
	}
	if !validMessageID(id) || messages[0].ID != id {
		return errors.New("batch head identity mismatch")
	}
	seen := make(map[string]bool, len(messages))
	for _, m := range messages {
		if !m.Origin.Valid() || !validMessageID(m.ID) || seen[m.ID] || m.To != to || m.From.Validate() != nil || m.Deadline.IsZero() {
			return errors.New("batch member identity mismatch")
		}
		if err := ValidateBody(m.Body); err != nil {
			return err
		}
		seen[m.ID] = true
	}
	if utf8.RuneCountInString(BatchEnvelope(messages)) > MaxPeerBatchChars {
		return errors.New("batch exceeds qualified composer character limit")
	}
	if !batchFinalizationFits(messages) {
		return errors.New("batch exceeds encoded endpoint frame limit")
	}
	return nil
}

type BatchPreparationPhase uint8

const (
	BatchPreparing BatchPreparationPhase = iota
	BatchReady
	BatchFrozen
)

type BatchPreparationEvent uint8

const (
	BatchAvailable BatchPreparationEvent = iota
	BatchUnavailable
	BatchFinalize
)

// AdvanceBatchPreparation grants sticky assembly permission once the writer
// offers readiness. Final paste eligibility remains the writer's responsibility.
func AdvanceBatchPreparation(phase BatchPreparationPhase, event BatchPreparationEvent) (BatchPreparationPhase, error) {
	if phase > BatchFrozen || event > BatchFinalize {
		return phase, errors.New("unknown batch preparation transition")
	}
	if phase == BatchFrozen {
		return phase, errors.New("batch membership is frozen")
	}
	switch event {
	case BatchAvailable:
		return BatchReady, nil
	case BatchUnavailable:
		return phase, nil
	case BatchFinalize:
		if phase == BatchReady {
			return BatchFrozen, nil
		}
	}
	return phase, errors.New("batch preparation is not ready")
}
