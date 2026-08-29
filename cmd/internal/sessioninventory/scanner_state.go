package sessioninventory

import "errors"

const ScannerStateVersion = 1

// ScannerState is the versioned agent-neutral envelope persisted after a
// scanner validates one native artifact through its parser-complete offset.
// Agent scanners own how records transition these fields.
// pair:156-concept pure new final
type ScannerState struct {
	Version              int         `json:"version"`
	Agent                Agent       `json:"agent"`
	NativeID             string      `json:"native_id"`
	IdentityAnchor       string      `json:"identity_anchor"`
	Role                 Role        `json:"role"`
	ParentID             *string     `json:"parent_id,omitempty"`
	ScannerSchema        string      `json:"scanner_schema"`
	Chronology           *NativeTime `json:"chronology,omitempty"`
	Disputed             bool        `json:"disputed"`
	FirstRecordValidated bool        `json:"first_record_validated"`
}

func ValidateScannerState(state ScannerState) error {
	if state.Version != ScannerStateVersion || !validAgent(state.Agent) || state.NativeID == "" || state.IdentityAnchor == "" || state.ScannerSchema == "" {
		return errors.New("invalid scanner state identity")
	}
	switch state.Role {
	case RoleRoot:
		if state.ParentID != nil || state.IdentityAnchor != state.NativeID {
			return errors.New("invalid root scanner state")
		}
	case RoleSubagent:
		if state.ParentID == nil || *state.ParentID == "" || state.IdentityAnchor != *state.ParentID {
			return errors.New("invalid subagent scanner state")
		}
	case RoleUnknown:
		if !state.Disputed || state.ParentID != nil {
			return errors.New("invalid unknown scanner state")
		}
	default:
		return errors.New("invalid scanner role")
	}
	if state.Chronology != nil {
		if state.Chronology.Value.IsZero() {
			return errors.New("invalid scanner chronology")
		}
		switch state.Chronology.Source {
		case TimeSourceMetadata, TimeSourceBirth, TimeSourceMTime:
		default:
			return errors.New("invalid scanner chronology source")
		}
	}
	return nil
}

func cloneScannerState(state ScannerState) ScannerState {
	state.ParentID = cloneString(state.ParentID)
	state.Chronology = cloneTime(state.Chronology)
	return state
}

func scannerStateFact(state ScannerState, artifacts []Artifact) Fact {
	return Fact{
		Agent: state.Agent, NativeID: state.NativeID, Role: state.Role, ParentID: cloneString(state.ParentID), Time: cloneTime(state.Chronology),
		Resumable: state.Role == RoleRoot && !state.Disputed, Disputed: state.Disputed, Artifacts: append([]Artifact(nil), artifacts...),
		EdgeProvenance: edgeProvenance(state.Role, state.ScannerSchema, artifacts[len(artifacts)-1]),
	}
}
