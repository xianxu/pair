package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/netip"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RemoteSource contains only server-verified display facts. It stays comparable
// so immutable Message identity remains usable by the pure mailbox actor.
type RemoteSource struct{ Email, IP string }

func (s RemoteSource) Validate() error {
	if len(s.Email) == 0 || len(s.Email) > 254 || !utf8.ValidString(s.Email) || strings.ContainsAny(s.Email, "[]<>") || strings.IndexFunc(s.Email, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return errors.New("invalid remote email")
	}
	address, err := mail.ParseAddress(s.Email)
	if err != nil || address.Address != s.Email || address.Name != "" {
		return errors.New("invalid remote email")
	}
	ip, err := netip.ParseAddr(s.IP)
	if err != nil || ip.Zone() != "" || ip.Unmap().String() != s.IP {
		return errors.New("invalid remote client IP")
	}
	return nil
}

func (m Message) ValidateProvenance() error {
	if !m.Origin.Valid() {
		return errors.New("unknown message origin")
	}
	if m.DocumentID != "" {
		if len(m.DocumentID) != 32 || strings.IndexFunc(m.DocumentID, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) >= 0 {
			return errors.New("invalid registered document identity")
		}
	}
	if m.Origin == OriginRemote {
		if err := m.RemoteSource.Validate(); err != nil {
			return err
		}
		// The added remote display line shares the body's qualified byte budget.
		if len(m.Body)+len("[remote  ]\n")+len(m.RemoteSource.Email)+len(m.RemoteSource.IP) > MaxBodyBytes {
			return errors.New("remote header and body exceed message byte limit")
		}
	} else if m.RemoteSource != (RemoteSource{}) {
		return errors.New("remote source requires remote origin")
	}
	if m.Origin == OriginSystem && (m.RemoteEdited || m.DocumentID != "") {
		return errors.New("system message cannot carry document provenance")
	}
	return nil
}

// SameProvenance compares every trusted fact without comparing mutable status.
func (m Message) SameProvenance(other Message) bool {
	return m.Origin == other.Origin && m.RemoteSource == other.RemoteSource && m.RemoteEdited == other.RemoteEdited && m.DocumentID == other.DocumentID
}

func (m Message) needsRemoteProvenance() bool {
	return m.Origin == OriginRemote || m.RemoteEdited || m.DocumentID != ""
}

type messageProvenance struct {
	origin     Origin
	source     RemoteSource
	edited     bool
	documentID string
}

func (p messageProvenance) message() Message {
	return Message{Origin: p.origin, RemoteSource: p.source, RemoteEdited: p.edited, DocumentID: p.documentID}
}
func (p messageProvenance) matches(m Message) bool { return p.message().SameProvenance(m) }

// RemoteProvenanceEndpoint qualifies a receiver before privileged mail enters
// its mailbox. Absence of this explicit capability is a refusal, never downgrade.
type RemoteProvenanceEndpoint interface{ SupportsRemoteProvenance(context.Context) error }

func (b *Broker) requireRemoteProvenance(ctx context.Context, a *SlotActor) error {
	b.mu.Lock()
	supported := a.remoteProvenance
	b.mu.Unlock()
	if supported {
		return nil
	}
	endpoint, ok := a.endpoint.(RemoteProvenanceEndpoint)
	if !ok {
		return &RelaunchRequiredError{Cause: fmt.Errorf("receiver lacks remote-provenance-v1; relaunch the receiver: %w", ErrUnsupported)}
	}
	if err := endpoint.SupportsRemoteProvenance(ctx); err != nil {
		if errors.Is(err, ErrUnsupported) {
			return &RelaunchRequiredError{Cause: fmt.Errorf("receiver lacks remote-provenance-v1; relaunch the receiver: %w", err)}
		}
		return fmt.Errorf("receiver remote provenance query failed: %w", err)
	}
	b.mu.Lock()
	if b.actors[a.binding] == a && b.caller(a.binding) {
		a.remoteProvenance = true
	}
	b.mu.Unlock()
	return nil
}

// UnsettledDocument answers only exact registered-document work, without IO or
// expiry inference. Active delivery jobs include receipt-recording callbacks;
// an indeterminate receipt is never evidence that an effect did not happen.
func (b *Broker) UnsettledDocument(documentID string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if documentID == "" {
		return true
	}
	for _, admission := range b.admissions {
		if admission.provenance.documentID == documentID {
			return true
		}
	}
	for id, receipt := range b.receipts {
		if receipt.Message.DocumentID == documentID && (b.jobs[id] != nil || !receipt.Status.Terminal() || receipt.Status == Indeterminate) {
			return true
		}
	}
	for _, actor := range b.actors {
		for _, message := range actor.state.PendingMessages() {
			if message.DocumentID == documentID {
				return true
			}
		}
		for _, message := range actor.state.QueuedMessages() {
			if message.DocumentID == documentID {
				return true
			}
		}
	}
	return false
}
