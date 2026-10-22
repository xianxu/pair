package editorendpoint

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func cloneState(s EndpointState) EndpointState {
	s.Unknown = strictjson.CloneRaw(s.Unknown)
	if s.Records != nil {
		records := make([]EndpointRecord, len(s.Records))
		copy(records, s.Records)
		s.Records = records
	}
	for i := range s.Records {
		if s.Records[i].Current != nil {
			b := *s.Records[i].Current
			b.Unknown = strictjson.CloneRaw(b.Unknown)
			s.Records[i].Current = &b
		}
		s.Records[i].Unknown = strictjson.CloneRaw(s.Records[i].Unknown)
		s.Records[i].Binding.Unknown = strictjson.CloneRaw(s.Records[i].Binding.Unknown)
	}
	return s
}

// CurrentBinding returns the selected authority represented by this record.
func CurrentBinding(r EndpointRecord) *EndpointBinding {
	if r.Current != nil {
		return r.Current
	}
	if r.Status == "current" {
		b := r.Binding
		return &b
	}
	return nil
}
func clearCurrent(r *EndpointRecord, status string) {
	if r.Current != nil {
		r.Current = nil
	} else if r.Status == "current" {
		r.Status = status
	}
}
func sameRequest(b EndpointBinding, r RegisterRequest, identity, file string) bool {
	return b.Namespace == r.Namespace && b.DocumentID == r.DocumentID && b.EditorIncarnation == r.EditorIncarnation && b.AttachID == r.AttachID && b.PID == r.PID && b.ProcessIdentity == identity && b.Socket == r.Socket && b.Buffer == r.Buffer && b.Mode == r.Mode && b.File == file
}
func exact(b, c EndpointBinding) bool {
	return b.Namespace == c.Namespace && b.EditorIncarnation == c.EditorIncarnation && b.DocumentID == c.DocumentID && b.AttachID == c.AttachID && b.BindingGeneration == c.BindingGeneration
}

// ApplyEndpointEvent never mutates its input. Classified refusals return nil error;
// Changed can be true on a refusal because its terminal high-water must be durable.
func ApplyEndpointEvent(s EndpointState, e EndpointEvent) (EndpointState, EndpointReply, error) {
	p := EndpointReply{Version: 1, Code: "ok"}
	count := 0
	for _, yes := range []bool{e.Register != nil, e.Unregister != nil, e.Retire != nil, e.Collect != nil} {
		if yes {
			count++
		}
	}
	if count != 1 {
		return s, p, errors.New("exactly one event required")
	}
	if s.Version != 0 {
		if err := validateState(s); err != nil {
			return s, p, err
		}
	} else if s.Namespace != "" || s.Revision != 0 || len(s.Records) > 0 || len(s.Unknown) > 0 {
		return s, p, errors.New("invalid initial state")
	}
	n := cloneState(s)
	finish := func() (EndpointState, EndpointReply, error) {
		if p.Changed {
			if n.Revision == ^uint64(0) {
				return s, EndpointReply{Version: 1}, errors.New("endpoint revision exhausted")
			}
			n.Revision++
			if err := validateState(n); err != nil {
				return s, EndpointReply{Version: 1}, err
			}
		}
		return n, p, nil
	}
	if r := e.Register; r != nil {
		if err := ValidateEndpointRequest(r); err != nil {
			return s, p, err
		}
		if !validHex(e.BindingGeneration) || !canonicalPath(e.File) || e.ProcessIdentity == "" || len(e.ProcessIdentity) > 4096 {
			return s, p, errors.New("invalid derived registration identity")
		}
		if n.Version == 0 {
			n.Version = 1
			n.Namespace = r.Namespace
			n.Records = []EndpointRecord{}
		}
		if n.Namespace != r.Namespace {
			return s, p, errors.New("namespace mismatch")
		}
		pair := -1
		current := -1
		active := 0
		for i, x := range n.Records {
			b := x.Binding
			if b.EditorIncarnation == r.EditorIncarnation {
				if b.PID != r.PID || b.ProcessIdentity != e.ProcessIdentity {
					return s, p, errors.New("incarnation process identity conflict")
				}
				if CurrentBinding(x) != nil {
					active++
				}
				if b.DocumentID == r.DocumentID {
					pair = i
				}
			}
			if CurrentBinding(x) != nil && b.DocumentID == r.DocumentID {
				current = i
			}
		}
		if pair >= 0 {
			old := n.Records[pair]
			oldID, _ := attach(old.Binding.AttachID)
			newID, _ := attach(r.AttachID)
			if newID <= oldID {
				p.Code = "stale"
				if newID == oldID && old.Status != "current" {
					p.Code = "superseded"
				}
				if newID == oldID && old.Status == "current" {
					if sameRequest(old.Binding, *r, e.ProcessIdentity, e.File) {
						b := old.Binding
						p.EditorEndpoint = &b
						p.Code = "ok"
					} else {
						p.Code = "conflict"
					}
				}
				return finish()
			}
		}
		if pair < 0 && len(n.Records) >= MaxRecords {
			p.Code = "capacity"
			return finish()
		}
		blocked := current >= 0 && CurrentBinding(n.Records[current]).Mode == "desktop" && r.Mode == "headless"
		// Capacity refusals do not consume attempts: no authority was admitted.
		if !blocked && active >= MaxProcessActive && (pair < 0 || CurrentBinding(n.Records[pair]) == nil) {
			p.Code = "capacity"
			return finish()
		}
		b := EndpointBinding{Namespace: r.Namespace, DocumentID: r.DocumentID, EditorIncarnation: r.EditorIncarnation, AttachID: r.AttachID, PID: r.PID, ProcessIdentity: e.ProcessIdentity, Socket: r.Socket, Buffer: r.Buffer, Mode: r.Mode, BindingGeneration: e.BindingGeneration, File: e.File}
		record := EndpointRecord{Binding: b, Status: "current"}
		if pair >= 0 {
			record.Unknown = n.Records[pair].Unknown
			record.Binding.Unknown = n.Records[pair].Binding.Unknown
		}
		if blocked {
			record.Status = "retired"
			if current == pair {
				record.Current = CurrentBinding(n.Records[current])
			}
			p.Code = "desktop-active"
		} else {
			if current >= 0 {
				clearCurrent(&n.Records[current], "superseded")
			}
			p.EditorEndpoint = &b
		}
		if pair < 0 {
			n.Records = append(n.Records, record)
		} else {
			n.Records[pair] = record
		}
		p.Changed = true
		return finish()
	}
	if u := e.Unregister; u != nil {
		if err := ValidateEndpointRequest(u); err != nil {
			return s, p, err
		}
		if n.Namespace != "" && n.Namespace != u.Namespace {
			return s, p, errors.New("namespace mismatch")
		}
		for _, entry := range u.Entries {
			result := UnregisterResult{DocumentID: entry.DocumentID, AttachID: entry.AttachID, BindingGeneration: entry.BindingGeneration, Status: "absent"}
			for i, x := range n.Records {
				b := x.Binding
				if b.DocumentID == entry.DocumentID && result.Status != "removed" {
					result.Status = "stale"
				}
				cb := CurrentBinding(x)
				if cb != nil {
					b = *cb
				}
				if cb != nil && b.Namespace == u.Namespace && b.EditorIncarnation == u.EditorIncarnation && b.PID == u.PID && (u.ProcessIdentity == "" || u.ProcessIdentity == b.ProcessIdentity) && b.DocumentID == entry.DocumentID && b.AttachID == entry.AttachID && b.BindingGeneration == entry.BindingGeneration {
					clearCurrent(&n.Records[i], "retired")
					result.Status = "removed"
					result.Changed = true
					p.Changed = true
				}
			}
			p.Results = append(p.Results, result)
		}
		return finish()
	}
	if b := e.Retire; b != nil {
		for i, x := range n.Records {
			cb := CurrentBinding(x)
			if cb != nil && exact(*cb, *b) {
				clearCurrent(&n.Records[i], "retired")
				p.Changed = true
			}
		}
		return finish()
	}
	c := e.Collect
	if !validHex(c.EditorIncarnation) || c.PID <= 0 || c.ProcessIdentity == "" {
		return s, p, errors.New("invalid collection evidence")
	}
	kept := n.Records[:0]
	for _, x := range n.Records {
		b := x.Binding
		if b.EditorIncarnation == c.EditorIncarnation && b.PID == c.PID && b.ProcessIdentity == c.ProcessIdentity {
			p.Changed = true
			continue
		}
		kept = append(kept, x)
	}
	n.Records = kept
	return finish()
}
