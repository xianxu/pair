package remotesession

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const maxLatchBytes = 4096

type latchPhase string

const (
	latchOff    latchPhase = "off"
	latchArmed  latchPhase = "armed"
	latchKilled latchPhase = "killed"
)

type latchRecord struct {
	Version    int
	Phase      latchPhase
	Generation uint64
	raw        map[string]json.RawMessage
}

func decodeLatch(raw []byte) (latchRecord, error) {
	var r latchRecord
	if err := validateRemoteJSON(raw, maxLatchBytes); err != nil {
		return r, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return r, err
	}
	if fields == nil {
		return r, errors.New("remote latch requires object")
	}
	for key := range fields {
		for _, known := range []string{"version", "phase", "generation"} {
			if key != known && strings.EqualFold(key, known) {
				return r, fmt.Errorf("remote latch case alias %q", key)
			}
		}
	}
	for _, f := range []struct {
		key    string
		target any
	}{{"version", &r.Version}, {"phase", &r.Phase}, {"generation", &r.Generation}} {
		v, ok := fields[f.key]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return r, fmt.Errorf("missing/null remote latch %s", f.key)
		}
		if err := json.Unmarshal(v, f.target); err != nil {
			return r, err
		}
	}
	if r.Version != 1 || (r.Phase != latchOff && r.Phase != latchArmed && r.Phase != latchKilled) {
		return r, errors.New("unsupported remote latch version or phase")
	}
	r.raw = fields
	return r, nil
}

func encodeLatch(r latchRecord) ([]byte, error) {
	fields := make(map[string]json.RawMessage, len(r.raw)+3)
	for key, value := range r.raw {
		fields[key] = value
	}
	fields["version"], _ = json.Marshal(r.Version)
	fields["phase"], _ = json.Marshal(r.Phase)
	fields["generation"], _ = json.Marshal(r.Generation)
	raw := marshalRemoteFields(fields)
	if _, err := decodeLatch(raw); err != nil {
		return nil, err
	}
	return raw, nil
}
