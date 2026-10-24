package zellijquery

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// ReadyIdentity is shared with the first-pane evidence; it certifies native
// initialization, independently of the later agent-ready contract.
type ReadyIdentity = panebirth.LaunchIdentity

type ReadyMarker struct {
	Identity                 ReadyIdentity
	Unknown, IdentityUnknown map[string]json.RawMessage
}

var ErrReadinessUnknown = errors.New("Zellij initialization is unverified")

var markerFields = map[string]bool{"identity": true}

func DecodeReadyMarker(raw []byte) (ReadyMarker, error) {
	var m ReadyMarker
	if len(raw) > panebirth.MaxRecordBytes {
		return m, errors.New("ready marker exceeds metadata bound")
	}
	owned, unknown, err := strictjson.SplitObjectPreserveNumbers(raw, markerFields, nil)
	if err != nil {
		return m, err
	}
	m.Identity, m.IdentityUnknown, err = panebirth.DecodeIdentity(owned["identity"])
	if err != nil {
		return ReadyMarker{}, err
	}
	m.Unknown = unknown
	return m, nil
}

func EncodeReadyMarker(m ReadyMarker) ([]byte, error) {
	identity, err := panebirth.EncodeIdentity(m.Identity, m.IdentityUnknown)
	if err != nil {
		return nil, err
	}
	raw, err := strictjson.EncodeObjectPreserveNumbers(map[string]json.RawMessage{"identity": identity}, m.Unknown, markerFields)
	if err == nil && len(raw) > panebirth.MaxRecordBytes {
		return nil, errors.New("ready marker exceeds metadata bound")
	}
	return raw, err
}

func QualifyReadyMarker(m ReadyMarker, namespace, session string) error {
	if err := m.Identity.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrReadinessUnknown, err)
	}
	if m.Identity.Namespace != namespace || m.Identity.Session != session {
		return fmt.Errorf("%w: marker names another namespace or session", ErrReadinessUnknown)
	}
	return nil
}

// PlanProbeAdmission makes one pass over the complete passive namespace set.
// A missing marker never becomes an empty native listing.
func PlanProbeAdmission(namespace string, candidates []string, markers map[string]ReadyMarker) error {
	for _, session := range candidates {
		marker, ok := markers[session]
		if !ok {
			return fmt.Errorf("%w: %s has no exact-launch marker", ErrReadinessUnknown, session)
		}
		if err := QualifyReadyMarker(marker, namespace, session); err != nil {
			return err
		}
	}
	return nil
}
