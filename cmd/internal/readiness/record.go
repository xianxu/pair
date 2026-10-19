package readiness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type ReadyRecord struct {
	AgentIdentity string `json:"agent_identity,omitempty"`
	LaunchOrdinal uint64 `json:"launch_ordinal,omitempty"`
	// DeprecatedOrientation is a TOMBSTONE: a wrapper before #522 published
	// its composer-paste delivery state here. A ready record written by such a
	// wrapper -- a slot launched by the old build, still running -- must still
	// decode, so it is accepted and ignored. Encode refuses it, so it is never
	// written. Delete it once every slot has been relaunched on a #522+ build.
	DeprecatedOrientation json.RawMessage `json:"orientation,omitempty"`
	Tag                   string          `json:"tag"`
	Agent                 string          `json:"agent"`
	Session               string          `json:"session"`
	Nonce                 string          `json:"nonce"`
	PID                   int             `json:"pid"`
	// TTY is the agent's controlling terminal as the kernel's device number in
	// decimal (procutil.Process.TTY: Darwin e_tdev, Linux tty_nr), published
	// only for a Couch-launched wrapper: Couch's teardown proves the agent gone
	// when no process holds it (pair#695 P1).
	TTY string `json:"tty,omitempty"`
}

var (
	readyKnownFields = strictjson.KnownFields(reflect.TypeOf(ReadyRecord{}))
	ttyNamePattern   = regexp.MustCompile(`^[A-Za-z0-9]{1,32}(/[0-9]{1,6})?$`)
)

func Encode(record ReadyRecord) (string, error) {
	if len(record.DeprecatedOrientation) != 0 {
		return "", fmt.Errorf("ready record: the orientation field is decode-only")
	}
	if err := Validate(record); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(record); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Decode accepts members it does not own and decodes its own strictly
// (pair#695 §2a). It keeps no unknown members: the wrapper is the record's
// only writer and always writes it whole, so no reader ever rewrites it.
func Decode(raw string) (ReadyRecord, error) {
	var record ReadyRecord
	if _, err := strictjson.DecodeKnown([]byte(raw), readyKnownFields, &record); err != nil {
		return ReadyRecord{}, err
	}
	if err := Validate(record); err != nil {
		return ReadyRecord{}, err
	}
	return record, nil
}

func Validate(record ReadyRecord) error {
	if len(record.AgentIdentity) > 256 || !utf8.ValidString(record.AgentIdentity) || strings.IndexFunc(record.AgentIdentity, unicode.IsControl) >= 0 {
		return fmt.Errorf("ready record: invalid agent process identity")
	}

	switch {
	case record.Tag == "":
		return fmt.Errorf("ready record: tag is empty")
	case record.Agent == "":
		return fmt.Errorf("ready record: agent is empty")
	case record.Session == "":
		return fmt.Errorf("ready record: session is empty")
	case record.Nonce == "":
		return fmt.Errorf("ready record: nonce is empty")
	case record.PID <= 0:
		return fmt.Errorf("ready record: pid must be positive")
	case record.TTY != "" && !ttyNamePattern.MatchString(record.TTY):
		return fmt.Errorf("ready record: invalid tty name")
	default:
		return nil
	}
}
