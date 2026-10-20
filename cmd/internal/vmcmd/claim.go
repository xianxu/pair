package vmcmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type ClaimStage string

const (
	ClaimLaunching ClaimStage = "launching"
	ClaimLaunched  ClaimStage = "launched"
)

// LaunchClaim survives the gap between spawning Tart and observing its child
// identity. An absent identity preserves uncertainty rather than granting reuse.
type LaunchClaim struct {
	Version       int                        `json:"version"`
	Lane          Lane                       `json:"lane"`
	VM            string                     `json:"vm"`
	Nonce         string                     `json:"nonce"`
	Stage         ClaimStage                 `json:"stage"`
	PID           int                        `json:"pid"`
	StartIdentity string                     `json:"start_identity"`
	Unknown       map[string]json.RawMessage `json:"-"`
}

var claimFields = strictjson.KnownFields(reflect.TypeOf(LaunchClaim{}))
var claimNonce = regexp.MustCompile(`^[0-9a-f]{32}$`)

func ManagedVMName(lane Lane) string { return "pair-conformance-run-" + string(lane) }

func DecodeLaunchClaim(raw []byte) (LaunchClaim, error) {
	var c LaunchClaim
	owned, unknown, err := requiredObject(raw, claimFields)
	if err != nil {
		return c, err
	}
	known, err := strictjson.EncodeObject(owned, nil, claimFields)
	if err != nil {
		return c, err
	}
	if err := strictjson.Decode(known, &c); err != nil {
		return LaunchClaim{}, err
	}
	c.Unknown = unknown
	if err := validateLaunchClaim(c); err != nil {
		return LaunchClaim{}, err
	}
	return c, nil
}

func EncodeLaunchClaim(c LaunchClaim) ([]byte, error) {
	if err := validateLaunchClaim(c); err != nil {
		return nil, err
	}
	owned, err := ownedValues(c)
	if err != nil {
		return nil, err
	}
	return strictjson.EncodeObject(owned, c.Unknown, claimFields)
}

func validateLaunchClaim(c LaunchClaim) error {
	if c.Version != 1 || (c.Lane != TrainLane && c.Lane != AdhocLane) || c.VM != ManagedVMName(c.Lane) || !claimNonce.MatchString(c.Nonce) {
		return fmt.Errorf("invalid VM launch claim identity")
	}
	switch c.Stage {
	case ClaimLaunching:
		if c.PID != 0 || c.StartIdentity != "" {
			return fmt.Errorf("launching claim cannot contain an observed child")
		}
	case ClaimLaunched:
		if c.PID <= 0 {
			return fmt.Errorf("launched claim needs its child PID")
		}
	default:
		return fmt.Errorf("unsupported VM launch claim stage %q", c.Stage)
	}
	return nil
}

type ProcessObservation uint8

const (
	ProcessUnknown ProcessObservation = iota
	ProcessAlive
	ProcessGone
)

type VMObservation uint8

const (
	VMUnknown VMObservation = iota
	VMRunning
	VMStopped
)

// RecoveryEvidence is a native observation about the exact previously recorded
// child. ProcessGone is reserved for positive absence, not probe errors.
type RecoveryEvidence struct {
	PID      int
	Identity string
	Process  ProcessObservation
	VM       VMObservation
}

// RecoverLaunchClaim permits cleanup only with positive evidence that neither
// the previous child nor its guest can still start/use the position.
func RecoverLaunchClaim(c LaunchClaim, proof RecoveryEvidence) error {
	if err := validateLaunchClaim(c); err != nil {
		return err
	}
	if c.Stage != ClaimLaunched || c.StartIdentity == "" || proof.PID != c.PID || proof.Identity != c.StartIdentity || proof.Process != ProcessGone || proof.VM != VMStopped {
		return fmt.Errorf("VM position has unresolved startup; inspect managed VM %s and its launch claim before reuse", c.VM)
	}
	return nil
}
