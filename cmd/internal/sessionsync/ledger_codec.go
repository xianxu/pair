package sessionsync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Leave one byte for the JSONL newline within readRows' scanner bound.
const ledgerEntryMaxBytes = (1 << 20) - 1

var ledgerEntryKeys = map[string]bool{"ID": true, "From": true, "At": true, "AddressedAt": true, "Head": true, "Uncertain": true, "Origin": true, "RemoteEdited": true, "RemoteSource": true, "DocumentID": true}
var ledgerSourceKeys = map[string]bool{"Email": true, "IP": true}
var ledgerProvenanceKeys = map[string]bool{"Origin": true, "RemoteEdited": true, "RemoteSource": true, "DocumentID": true}

// entryWire avoids recursively invoking the retained-raw codec.
type entryWire Entry

func ledgerObject(raw []byte, keys map[string]bool) (map[string]json.RawMessage, error) {
	if len(raw) > ledgerEntryMaxBytes {
		return nil, errors.New("delivery row exceeds size limit")
	}
	if err := strictjson.RejectDuplicateKeysPreserveNumbers(raw); err != nil {
		return nil, err
	}
	depth := 0
	quoted, escaped := false, false
	for _, b := range raw {
		if quoted {
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				quoted = false
			}
			continue
		}
		switch b {
		case '"':
			quoted = true
		case '{', '[':
			depth++
			if depth > 32 {
				return nil, errors.New("delivery row exceeds depth limit")
			}
		case '}', ']':
			depth--
		}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, errors.New("delivery object required")
	}
	for key, value := range fields {
		for known := range keys {
			if key != known && strings.EqualFold(key, known) {
				return nil, fmt.Errorf("delivery member case alias %q", key)
			}
		}
		if keys[key] && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("delivery member %s cannot be null", key)
		}
	}
	return fields, nil
}

func (e *Entry) UnmarshalJSON(raw []byte) error {
	fields, err := ledgerObject(raw, ledgerEntryKeys)
	if err != nil {
		return err
	}
	var decoded entryWire
	owned := map[string]json.RawMessage{}
	unknown := map[string]json.RawMessage{}
	for key, value := range fields {
		if ledgerEntryKeys[key] {
			if key != "RemoteSource" {
				owned[key] = value
			}
		} else {
			unknown[key] = value
		}
	}
	encoded, err := strictjson.EncodeObject(owned, nil, nil)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	var sourceUnknown map[string]json.RawMessage
	if rawSource, present := fields["RemoteSource"]; present {
		source, err := ledgerObject(rawSource, ledgerSourceKeys)
		if err != nil {
			return err
		}
		sourceUnknown = map[string]json.RawMessage{}
		for key, value := range source {
			switch key {
			case "Email":
				if err := json.Unmarshal(value, &decoded.RemoteSource.Email); err != nil {
					return err
				}
			case "IP":
				if err := json.Unmarshal(value, &decoded.RemoteSource.IP); err != nil {
					return err
				}
			default:
				sourceUnknown[key] = value
			}
		}
	}
	hasProvenance := false
	for key := range ledgerProvenanceKeys {
		if _, ok := fields[key]; ok {
			hasProvenance = true
		}
	}
	if _, present := fields["RemoteSource"]; present && decoded.Origin != couchmessage.OriginRemote {
		return errors.New("remote source requires remote origin")
	}
	if hasProvenance {
		if decoded.ID == "" {
			return errors.New("delivery provenance requires ID")
		}
		if _, ok := fields["Origin"]; !ok {
			return errors.New("delivery provenance requires Origin")
		}
		if _, ok := fields["RemoteEdited"]; !ok {
			return errors.New("delivery provenance requires RemoteEdited")
		}
	}
	if err := (couchmessage.Message{Origin: decoded.Origin, RemoteEdited: decoded.RemoteEdited, RemoteSource: decoded.RemoteSource, DocumentID: decoded.DocumentID}).ValidateProvenance(); err != nil {
		return err
	}
	decoded.provenancePresent = hasProvenance
	decoded.rawFields = unknown
	decoded.rawSource = sourceUnknown
	*e = Entry(decoded)
	return nil
}

func (e Entry) MarshalJSON() ([]byte, error) { return encodeLedgerEntry(e) }

func encodeLedgerEntry(e Entry) ([]byte, error) {
	if err := (couchmessage.Message{Origin: e.Origin, RemoteEdited: e.RemoteEdited, RemoteSource: e.RemoteSource, DocumentID: e.DocumentID}).ValidateProvenance(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(entryWire(e))
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	// A struct-valued omitempty would emit an empty RemoteSource on legacy rows.
	delete(fields, "RemoteSource")
	if e.provenancePresent || e.Origin != "" || e.RemoteEdited || e.DocumentID != "" || e.RemoteSource != (couchmessage.RemoteSource{}) {
		fields["Origin"], _ = json.Marshal(e.Origin)
		fields["RemoteEdited"], _ = json.Marshal(e.RemoteEdited)
	}
	if e.RemoteSource != (couchmessage.RemoteSource{}) {
		source := strictjson.CloneRaw(e.rawSource)
		if source == nil {
			source = map[string]json.RawMessage{}
		}
		source["Email"], _ = json.Marshal(e.RemoteSource.Email)
		source["IP"], _ = json.Marshal(e.RemoteSource.IP)
		fields["RemoteSource"], err = strictjson.EncodeObject(source, nil, nil)
		if err != nil {
			return nil, err
		}
	}
	for key, value := range e.rawFields {
		if !ledgerEntryKeys[key] {
			fields[key] = value
		}
	}
	// All raw members are revalidated by the same bounded preserving reader.
	out, err := strictjson.EncodeObject(fields, nil, nil)
	if err != nil {
		return nil, err
	}
	var check Entry
	if err := check.UnmarshalJSON(out); err != nil {
		return nil, err
	}
	return out, nil
}

// Recognize keys even when a provenance value is malformed or torn. A legacy
// torn tail can be ignored; recognized security metadata must fail closed.
func hasLedgerProvenance(raw []byte) bool {
	// The main decoder has already rejected this bounded row. Scan string keys
	// lexically so a malformed earlier future value cannot hide later authority.
	// Quoted value contents are skipped whole, including escaped quotes.
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		start := i
		closed := false
		for i++; i < len(raw); i++ {
			if raw[i] == '\\' {
				i++
				continue
			}
			if raw[i] == '"' {
				closed = true
				break
			}
		}
		if !closed {
			return false
		}
		next := i + 1
		for next < len(raw) && (raw[next] == ' ' || raw[next] == '\t' || raw[next] == '\r' || raw[next] == '\n') {
			next++
		}
		if next >= len(raw) || raw[next] != ':' {
			continue
		}
		var key string
		if json.Unmarshal(raw[start:i+1], &key) != nil {
			continue
		}
		for known := range ledgerProvenanceKeys {
			if strings.EqualFold(key, known) {
				return true
			}
		}
	}
	return false
}
