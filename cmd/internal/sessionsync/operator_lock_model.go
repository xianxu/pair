package sessionsync

import (
	"fmt"
	"strings"
	"time"
)

// OperatorLockRequest pins the registration and the editor's particular turn.
// The editor keeps Turn across lost replies, and uses a new nonce after release.
type OperatorLockRequest struct{ File, ID, Turn string }

// OperatorLockState describes a completed operator ownership operation.
type OperatorLockState string

const (
	OperatorAcquired        OperatorLockState = "acquired"
	OperatorAlreadyHeld     OperatorLockState = "already-held"
	OperatorReleased        OperatorLockState = "released"
	OperatorAlreadyReleased OperatorLockState = "already-released"
)

// OperatorLock is the acknowledged public ownership proof. Released results
// echo the requested identity but have neither a holder nor protective text.
type OperatorLock struct {
	ID, File string
	Holder   Holder
	Turn     string
	LockPath string
	LockText string
	State    OperatorLockState
}

type operatorTurn struct{ ID, Token string }
type operatorLockAction uint8

const (
	operatorAcquire operatorLockAction = iota
	operatorRelease
)

type operatorLockObservation struct {
	Path, Text string
	Exists     bool
}

type operatorLockDecision struct {
	State              OperatorLockState
	LockPath, LockText string
	Remove             []string
}

func validateOperatorTurn(turn operatorTurn) error {
	if !registryID(turn.ID) || !registryID(turn.Token) {
		return registryRefusal("operator-invalid", "operator ID and turn must each be 32 lowercase hexadecimal characters")
	}
	return nil
}

// operatorFields recognizes single known fields and leaves unknown lines
// uninterpreted. Existing proof text is never re-encoded during recovery.
func operatorFields(text string) (map[string]string, bool) {
	if len(text) > 4096 || strings.ContainsRune(text, 0) {
		return nil, false
	}
	fields := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch key {
		case "holder", "time", "document", "turn":
			if _, exists := fields[key]; exists {
				return nil, false
			}
			fields[key] = strings.TrimSpace(value)
		}
	}
	return fields, true
}

func parseOperatorProof(text string) (operatorTurn, bool) {
	fields, ok := operatorFields(text)
	turn := operatorTurn{ID: fields["document"], Token: fields["turn"]}
	_, timeErr := time.Parse(time.RFC3339Nano, fields["time"])
	return turn, ok && fields["holder"] == "operator" && timeErr == nil && validateOperatorTurn(turn) == nil
}

func operatorLockText(turn operatorTurn, at time.Time) string {
	return fmt.Sprintf("holder: operator\ntime: %s\ndocument: %s\nturn: %s\n", at.UTC().Format(time.RFC3339Nano), turn.ID, turn.Token)
}

func decideOperatorLock(turn operatorTurn, action operatorLockAction, locks []operatorLockObservation) (operatorLockDecision, error) {
	if err := validateOperatorTurn(turn); err != nil {
		return operatorLockDecision{}, err
	}
	decision := operatorLockDecision{State: OperatorAcquired}
	if action == operatorRelease {
		decision.State = OperatorAlreadyReleased
	}
	seen := map[string]bool{}
	for _, lock := range locks {
		if seen[lock.Path] || !lock.Exists {
			continue
		}
		seen[lock.Path] = true
		fields, valid := operatorFields(lock.Text)
		if ParseHolder(lock.Text, true) == HolderNone && valid && fields["holder"] == "agent" {
			continue
		}
		proof, ok := parseOperatorProof(lock.Text)
		if !ok || proof != turn {
			return operatorLockDecision{}, &Refusal{Exit: ExitOperatorLock, Code: "operator-lock", Reason: "operator protection belongs to another or unproven turn; inspect it without removing it"}
		}
		if action == operatorRelease {
			decision.State = OperatorReleased
			decision.Remove = append(decision.Remove, lock.Path)
		} else if decision.State != OperatorAlreadyHeld {
			decision.State = OperatorAlreadyHeld
			decision.LockPath, decision.LockText = lock.Path, lock.Text
		}
	}
	return decision, nil
}
