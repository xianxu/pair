package threadrecord

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type nativeRecoverySourceWire NativeRecoverySource
type recoveryContextWire RecoveryContext
type recoveryCandidateWire RecoveryCandidate

var nativeRecoverySourceFields = strictjson.KnownFields(reflect.TypeOf(nativeRecoverySourceWire{}))
var recoveryContextFields = strictjson.KnownFields(reflect.TypeOf(recoveryContextWire{}))
var recoveryCandidateFields = strictjson.KnownFields(reflect.TypeOf(recoveryCandidateWire{}))

func (source *NativeRecoverySource) UnmarshalJSON(raw []byte) error {
	var wire nativeRecoverySourceWire
	owned, unknown, err := strictjson.SplitObject(raw, nativeRecoverySourceFields, nil)
	if err != nil {
		return err
	}
	encoded, err := strictjson.EncodeObject(owned, nil, nativeRecoverySourceFields)
	if err != nil {
		return err
	}
	err = strictjson.Decode(encoded, &wire)
	if err != nil {
		return err
	}
	*source = NativeRecoverySource(wire)
	if len(unknown) > 0 {
		source.Unknown = unknown
	}
	return nil
}

func (source NativeRecoverySource) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(nativeRecoverySourceWire(source))
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, source.Unknown, nativeRecoverySourceFields)
}

// A context owns the version discriminator. Candidates are decoded here so v1
// retains its strict nested boundary while v2 preserves additive evidence.
func (c *RecoveryContext) UnmarshalJSON(raw []byte) error {
	owned, unknown, err := strictjson.SplitObject(raw, recoveryContextFields, recoveryContextFields)
	if err != nil {
		return err
	}
	var version int
	if err := strictjson.Decode(owned["version"], &version); err != nil {
		return err
	}
	if version != 1 && version != 2 {
		return errors.New("unsupported recovery context version")
	}
	if version == 1 {
		if len(unknown) > 0 {
			return errors.New("unknown capture recovery context member")
		}
		if _, ok := owned["source"]; ok {
			return errors.New("capture recovery cannot name native source")
		}
	} else {
		for _, value := range owned {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return errors.New("native recovery member cannot be null")
			}
		}
		for _, field := range []string{"capture", "generation"} {
			if _, ok := owned[field]; ok {
				return errors.New("native recovery contains capture evidence")
			}
		}
		if value, ok := owned["source"]; !ok || string(value) == "null" {
			return errors.New("native recovery source required")
		}
	}
	candidates := owned["candidates"]
	delete(owned, "candidates")
	encoded, err := strictjson.EncodeObject(owned, nil, recoveryContextFields)
	if err != nil {
		return err
	}
	var wire recoveryContextWire
	if err := strictjson.Decode(encoded, &wire); err != nil {
		return err
	}
	if len(candidates) > 0 {
		var rows []json.RawMessage
		if err := strictjson.Decode(candidates, &rows); err != nil {
			return err
		}
		if rows != nil {
			wire.Candidates = make([]RecoveryCandidate, len(rows))
		}
		for i, row := range rows {
			if version == 1 {
				var candidate recoveryCandidateWire
				if err := strictjson.Decode(row, &candidate); err != nil {
					return err
				}
				wire.Candidates[i] = RecoveryCandidate(candidate)
			} else {
				values, extra, err := strictjson.SplitObject(row, recoveryCandidateFields, nil)
				if err != nil {
					return err
				}
				for _, field := range []string{"score", "matched", "total"} {
					if _, ok := values[field]; ok {
						return errors.New("native recovery contains tail scoring")
					}
				}
				for _, field := range []string{"cwd", "key", "agent", "native_id", "artifact_root", "artifact_relative_path", "stable_file_id", "mutation_token", "size", "mod_time", "last_activity", "eligible"} {
					if value, ok := values[field]; !ok || string(value) == "null" {
						return errors.New("native recovery candidate member required")
					}
				}
				encoded, err := strictjson.EncodeObject(values, nil, recoveryCandidateFields)
				if err != nil {
					return err
				}
				var candidate recoveryCandidateWire
				if err := strictjson.Decode(encoded, &candidate); err != nil {
					return err
				}
				wire.Candidates[i] = RecoveryCandidate(candidate)
				if len(extra) > 0 {
					wire.Candidates[i].Unknown = extra
				}
			}
		}
	}
	*c = RecoveryContext(wire)
	if len(unknown) > 0 {
		c.Unknown = unknown
	}
	return nil
}

func (c RecoveryContext) MarshalJSON() ([]byte, error) {
	// Preserve the established v1 member order and known-field bytes.
	if c.Version != 2 {
		return json.Marshal(recoveryContextWire(c))
	}
	raw, err := json.Marshal(recoveryContextWire(c))
	if err != nil {
		return nil, err
	}
	var owned map[string]json.RawMessage
	if err := json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	delete(owned, "capture")
	delete(owned, "generation")
	if c.Candidates != nil {
		rows := make([]json.RawMessage, len(c.Candidates))
		for i, candidate := range c.Candidates {
			encoded, err := json.Marshal(recoveryCandidateWire(candidate))
			if err != nil {
				return nil, err
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				return nil, err
			}
			delete(fields, "score")
			delete(fields, "matched")
			delete(fields, "total")
			rows[i], err = strictjson.EncodeObject(fields, candidate.Unknown, recoveryCandidateFields)
			if err != nil {
				return nil, err
			}
		}
		owned["candidates"], err = json.Marshal(rows)
		if err != nil {
			return nil, err
		}
	}
	return strictjson.EncodeObject(owned, c.Unknown, recoveryContextFields)
}
