package panebirth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// MaxRecordBytes bounds the pane sidecar, including additive future fields.
const MaxRecordBytes = 8 << 10

// LaunchIdentity identifies one exact first-pane launch. A zero identity is a
// legacy sidecar and cannot authorize readiness.
type LaunchIdentity struct {
	Namespace string `json:"namespace"`
	Scope     string `json:"scope"`
	Tag       string `json:"tag"`
	Agent     string `json:"agent"`
	Session   string `json:"session"`
	Nonce     string `json:"nonce"`
}

func (i LaunchIdentity) Validate() error {
	for _, value := range []string{i.Namespace, i.Scope, i.Tag, i.Agent, i.Session, i.Nonce} {
		if !utf8.ValidString(value) || strings.TrimSpace(value) == "" || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return errors.New("invalid pane launch identity")
		}
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: i.Namespace, RepoScope: i.Scope, Tag: i.Tag})
	if err != nil {
		return err
	}
	if _, err = paths.PaneChecked(i.Agent); err != nil {
		return err
	}
	if i.Session == "." || i.Session == ".." || filepath.Base(i.Session) != i.Session {
		return errors.New("invalid pane launch session")
	}
	return nil
}

// Equal requires valid identities as well as equality: two legacy records do
// not become exact launch evidence merely because both identities are empty.
func (i LaunchIdentity) Equal(other LaunchIdentity) bool {
	return i == other && i.Validate() == nil
}

// Record is the existing pane sidecar with optional exact launch evidence.
// Unknown members are retained without interpreting future schemas.
type Record struct {
	PaneID          string
	Cwd             string
	Identity        LaunchIdentity
	Unknown         map[string]json.RawMessage
	IdentityUnknown map[string]json.RawMessage
}

// DecodeRecord accepts legacy pane records and additive future members, but
// rejects duplicates anywhere and requires strict types for every known field.
func DecodeRecord(raw []byte) (Record, error) {
	var record Record
	if len(raw) > MaxRecordBytes {
		return record, errors.New("pane record exceeds 8 KiB")
	}
	if err := strictjson.RejectDuplicateKeysPreserveNumbers(raw); err != nil {
		return record, err
	}
	fields, err := recordObject(raw)
	if err != nil {
		return record, err
	}
	if record.PaneID, err = recordString(fields, "pane_id", true); err != nil {
		return Record{}, err
	}
	if err = validatePaneID(record.PaneID); err != nil {
		return Record{}, err
	}
	if record.Cwd, err = recordString(fields, "cwd", false); err != nil {
		return Record{}, err
	}
	if identityRaw, ok := fields["identity"]; ok {
		record.Identity, record.IdentityUnknown, err = DecodeIdentity(identityRaw)
		if err != nil {
			return Record{}, err
		}
		delete(fields, "identity")
	}
	record.Unknown = fields
	return record, nil
}

// EncodeRecord validates direct values as strictly as decoded ones and retains
// unknown raw members. Callers cannot use retained fields to shadow known ones.
func EncodeRecord(record Record) ([]byte, error) {
	if err := validatePaneID(record.PaneID); err != nil {
		return nil, err
	}
	fields, err := copyRecordUnknown(record.Unknown, "pane_id", "cwd", "identity")
	if err != nil {
		return nil, err
	}
	fields["pane_id"], _ = json.Marshal(record.PaneID)
	fields["cwd"], _ = json.Marshal(record.Cwd)
	if record.Identity != (LaunchIdentity{}) || len(record.IdentityUnknown) > 0 {
		fields["identity"], err = EncodeIdentity(record.Identity, record.IdentityUnknown)
		if err != nil {
			return nil, err
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	// This also rejects malformed or duplicate members inside retained raw JSON.
	if _, err := DecodeRecord(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// DecodeIdentity decodes a complete launch identity while retaining additive
// members. It uses the same structural and size rules as pane records.
func DecodeIdentity(raw []byte) (LaunchIdentity, map[string]json.RawMessage, error) {
	var identity LaunchIdentity
	if len(raw) > MaxRecordBytes {
		return identity, nil, errors.New("pane identity exceeds 8 KiB")
	}
	if err := strictjson.RejectDuplicateKeysPreserveNumbers(raw); err != nil {
		return identity, nil, err
	}
	fields, err := recordObject(raw)
	if err != nil {
		return identity, nil, err
	}
	for key, dst := range map[string]*string{"namespace": &identity.Namespace, "scope": &identity.Scope, "tag": &identity.Tag, "agent": &identity.Agent, "session": &identity.Session, "nonce": &identity.Nonce} {
		*dst, err = recordString(fields, key, true)
		if err != nil {
			return LaunchIdentity{}, nil, err
		}
	}
	if err := identity.Validate(); err != nil {
		return LaunchIdentity{}, nil, err
	}
	return identity, fields, nil
}

// EncodeIdentity retains unknown raw members without letting them shadow the
// known launch identity. Invalid direct values and ambiguous JSON are rejected.
func EncodeIdentity(identity LaunchIdentity, unknown map[string]json.RawMessage) ([]byte, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	fields, err := copyRecordUnknown(unknown, "namespace", "scope", "tag", "agent", "session", "nonce")
	if err != nil {
		return nil, err
	}
	for key, value := range map[string]string{"namespace": identity.Namespace, "scope": identity.Scope, "tag": identity.Tag, "agent": identity.Agent, "session": identity.Session, "nonce": identity.Nonce} {
		fields[key], _ = json.Marshal(value)
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	if _, _, err := DecodeIdentity(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func recordObject(raw []byte) (map[string]json.RawMessage, error) {
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("pane record member must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func recordString(fields map[string]json.RawMessage, key string, required bool) (string, error) {
	raw, ok := fields[key]
	if !ok && !required {
		return "", nil
	}
	if !ok {
		return "", fmt.Errorf("missing pane record %s", key)
	}
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || trimmed[0] != '"' {
		return "", fmt.Errorf("pane record %s must be a string", key)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	delete(fields, key)
	return value, nil
}

func validatePaneID(value string) error {
	if value == "" {
		return errors.New("empty pane ID")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return errors.New("pane ID must be a nonnegative numeric string")
		}
	}
	if _, err := strconv.ParseUint(value, 10, 64); err != nil {
		return errors.New("pane ID out of range")
	}
	return nil
}

func copyRecordUnknown(source map[string]json.RawMessage, known ...string) (map[string]json.RawMessage, error) {
	target := make(map[string]json.RawMessage, len(source)+len(known))
	for key, value := range source {
		for _, reserved := range known {
			if key == reserved {
				return nil, fmt.Errorf("retained pane field shadows %s", key)
			}
		}
		target[key] = value
	}
	return target, nil
}
