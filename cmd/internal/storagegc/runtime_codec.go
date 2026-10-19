package storagegc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// runtimeRegistry owns a typed projection and the opaque wire members of each
// surviving registration. Extensions never enter shared process identities.
type runtimeRegistry struct {
	Version   int                   `json:"version"`
	Processes []ProcessRegistration `json:"processes"`
	envelope  map[string]json.RawMessage
	rows      map[string]runtimeWireRow
}

type runtimeWireRow struct {
	fields  map[string]json.RawMessage
	process map[string]json.RawMessage
}

func runtimeObject(raw []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, errors.New("runtime registry requires an object")
	}
	return fields, nil
}

func runtimeField(fields map[string]json.RawMessage, key string, target any, required bool) error {
	raw, ok := fields[key]
	if !ok {
		if required {
			return fmt.Errorf("missing runtime field %s", key)
		}
		return nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("null runtime field %s", key)
	}
	return json.Unmarshal(raw, target)
}

func (r *runtimeRegistry) UnmarshalJSON(raw []byte) error {
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return err
	}
	fields, err := runtimeObject(raw)
	if err != nil {
		return err
	}
	registry := runtimeRegistry{envelope: fields, rows: make(map[string]runtimeWireRow)}
	if err := runtimeField(fields, "version", &registry.Version, true); err != nil {
		return err
	}
	var rows []json.RawMessage
	if err := runtimeField(fields, "processes", &rows, true); err != nil {
		return err
	}
	if registry.Version != 1 || rows == nil || len(rows) > 256 {
		return errors.New("invalid runtime registry schema")
	}
	registry.Processes = make([]ProcessRegistration, 0, len(rows))
	identities := make(map[string]bool)
	for _, rawRow := range rows {
		row, err := runtimeObject(rawRow)
		if err != nil {
			return err
		}
		var entry ProcessRegistration
		for _, field := range []struct {
			name     string
			target   any
			required bool
		}{
			{"id", &entry.ID, true}, {"role", &entry.Role, true}, {"target", &entry.Target, false}, {"diagnostic_protocol", &entry.DiagnosticProtocol, false},
		} {
			if err := runtimeField(row, field.name, field.target, field.required); err != nil {
				return err
			}
		}
		process, err := runtimeObject(row["process"])
		if err != nil {
			return err
		}
		if err := runtimeField(process, "pid", &entry.Process.PID, true); err != nil {
			return err
		}
		if err := runtimeField(process, "birth", &entry.Process.Birth, true); err != nil {
			return err
		}
		if entry.ID == "" || entry.Role == "" || len(entry.Role) > 128 || entry.Target != "" || entry.Process.PID <= 0 || !validRuntimeBirth(entry.Process.Birth) {
			return errors.New("invalid runtime registration")
		}
		key := strconv.Itoa(entry.Process.PID) + "/" + entry.Process.Birth + "/" + entry.Role
		if _, exists := registry.rows[entry.ID]; exists || identities[key] {
			return errors.New("duplicate runtime registration")
		}
		identities[key] = true
		registry.rows[entry.ID] = runtimeWireRow{fields: row, process: process}
		registry.Processes = append(registry.Processes, entry)
	}
	*r = registry
	return nil
}

// mergeRuntimeFields copies before merging, so marshaling does not mutate the
// retained wire snapshot (including a caller that retries failed publication).
func mergeRuntimeFields(original map[string]json.RawMessage, owned any) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(owned)
	if err != nil {
		return nil, err
	}
	fields, err := runtimeObject(raw)
	if err != nil {
		return nil, err
	}
	for key, value := range original {
		if _, owned := fields[key]; !owned {
			fields[key] = value
		}
	}
	return fields, nil
}

func (r runtimeRegistry) MarshalJSON() ([]byte, error) {
	rows := make([]json.RawMessage, 0, len(r.Processes))
	for _, entry := range r.Processes {
		wire := r.rows[entry.ID]
		row, err := mergeRuntimeFields(wire.fields, entry)
		if err != nil {
			return nil, err
		}
		process, err := mergeRuntimeFields(wire.process, entry.Process)
		if err != nil {
			return nil, err
		}
		row["process"], err = json.Marshal(process)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, raw)
	}
	root, err := mergeRuntimeFields(r.envelope, struct {
		Version   int               `json:"version"`
		Processes []json.RawMessage `json:"processes"`
	}{r.Version, rows})
	if err != nil {
		return nil, err
	}
	return json.Marshal(root)
}
