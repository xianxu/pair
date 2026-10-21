package couchschedule

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// stateObject requires every owned field and preserves additive members.
func stateObject(raw []byte, target any) (map[string]json.RawMessage, error) {
	typ := reflect.TypeOf(target).Elem()
	known := strictjson.KnownFields(typ)
	nullable := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Type.Kind() == reflect.Pointer {
			for key := range strictjson.KnownFields(reflect.StructOf([]reflect.StructField{field})) {
				nullable[key] = true
			}
		}
	}
	owned, unknown, err := strictjson.SplitObject(raw, known, nullable)
	if err != nil {
		return nil, err
	}
	for name := range known {
		if _, ok := owned[name]; !ok {
			return nil, fmt.Errorf("state requires %s", name)
		}
	}
	clean, err := strictjson.EncodeObject(owned, nil, known)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(clean, target); err != nil {
		return nil, err
	}
	return unknown, nil
}

// encodeStateObject never re-marshals unknown member values.
func encodeStateObject(value any, unknown map[string]json.RawMessage) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	known := strictjson.KnownFields(reflect.TypeOf(value))
	owned, _, err := strictjson.SplitObject(raw, known, known)
	if err != nil {
		return nil, err
	}
	return strictjson.EncodeObject(owned, unknown, known)
}

func validateStoredTask(id string, t TaskState) error {
	if id != t.Key.ID() || !filepath.IsAbs(t.Key.Hub) || filepath.Clean(t.Key.Hub) != t.Key.Hub || t.Key.Task == "" || t.Revision == "" {
		return fmt.Errorf("invalid task identity")
	}
	if t.Allowed {
		if !ValidTaskName(t.Key.Task) {
			return fmt.Errorf("invalid task name")
		}
		if t.Spec.Version != 1 || t.Spec.Timeout <= 0 || t.Spec.Timeout > 24*time.Hour || (t.Spec.Scope != "machine" && t.Spec.Scope != "team") || (t.Spec.Missed != "skip" && t.Spec.Missed != "once") {
			return fmt.Errorf("invalid stored task spec")
		}
		if _, err := NewCalendar(t.Spec.Cron, time.UTC); err != nil {
			return err
		}
		if t.Spec.Scope == "team" && !validOperatorSlug(t.Spec.Owner) {
			return fmt.Errorf("invalid stored owner")
		}
	}
	if a := t.Active; a != nil {
		if !ValidTaskName(a.ID) || a.Revision == "" || (a.Source != "manual" && a.Source != "scheduled") || a.ControllerPID < 0 || a.GroupPID < 0 || a.ParentPID < 0 || a.Deadline.IsZero() {
			return fmt.Errorf("invalid active attempt")
		}
		switch a.Stage {
		case "reserved", "queued", "building", "running", "notifying", "delivery", "stopping", "uncertain":
		default:
			return fmt.Errorf("invalid attempt stage")
		}
	}
	if last := t.Last; last != nil {
		if !ValidTaskName(last.ID) || (last.Source != "manual" && last.Source != "scheduled") || last.Outcome == "" {
			return fmt.Errorf("invalid last summary")
		}
	}
	return nil
}

// DecodeLedger refuses corruption rather than replacing authority with empty state.
func DecodeLedger(raw []byte) (Ledger, error) {
	var l Ledger
	if len(raw) > maxScheduleStateBytes || !utf8.Valid(raw) {
		return l, fmt.Errorf("schedule state exceeds UTF-8/byte bound")
	}
	known := map[string]bool{"version": true, "tasks": true}
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil {
		return l, err
	}
	if err = json.Unmarshal(owned["version"], &l.Version); err != nil {
		return l, err
	}
	if l.Version != 1 {
		return l, fmt.Errorf("unsupported schedule state version")
	}
	var rows map[string]json.RawMessage
	if err = json.Unmarshal(owned["tasks"], &rows); err != nil || rows == nil {
		return l, fmt.Errorf("schedule tasks object required")
	}
	if len(rows) > 256 {
		return l, fmt.Errorf("schedule task cap exceeded")
	}
	l.Tasks = map[string]TaskState{}
	l.Unknown = unknown
	l.original = rows
	for id, row := range rows {
		var task TaskState
		task.Unknown, err = stateObject(row, &task)
		if err != nil {
			return Ledger{}, fmt.Errorf("task %s: %w", id, err)
		}
		var fields map[string]json.RawMessage
		json.Unmarshal(row, &fields)
		if _, err = stateObject(fields["key"], &task.Key); err != nil {
			return Ledger{}, err
		}
		if _, err = stateObject(fields["spec"], &task.Spec); err != nil {
			return Ledger{}, err
		}
		if task.Active != nil {
			task.Active.Unknown, err = stateObject(fields["active"], task.Active)
			if err != nil {
				return Ledger{}, err
			}
		}
		if task.Last != nil {
			task.Last.Unknown, err = stateObject(fields["last"], task.Last)
			if err != nil {
				return Ledger{}, err
			}
		}
		if err = validateStoredTask(id, task); err != nil {
			return Ledger{}, err
		}
		l.Tasks[id] = task
	}
	return l, nil
}

func EncodeLedger(l Ledger) ([]byte, error) {
	if l.Version != 1 || l.Tasks == nil || len(l.Tasks) > 256 {
		return nil, fmt.Errorf("invalid schedule ledger version/tasks")
	}
	rows := map[string]json.RawMessage{}
	for id, task := range l.Tasks {
		if err := validateStoredTask(id, task); err != nil {
			return nil, err
		}
		raw, err := encodeStateObject(task, task.Unknown)
		if err != nil {
			return nil, err
		}
		var fields map[string]json.RawMessage
		json.Unmarshal(raw, &fields)
		var original map[string]json.RawMessage
		if l.original[id] != nil {
			json.Unmarshal(l.original[id], &original)
		}
		for name, value := range map[string]any{"key": task.Key, "spec": task.Spec} {
			var unknown map[string]json.RawMessage
			if old := original[name]; old != nil {
				_, unknown, err = strictjson.SplitObject(old, strictjson.KnownFields(reflect.TypeOf(value)), nil)
				if err != nil {
					return nil, err
				}
			}
			fields[name], err = encodeStateObject(value, unknown)
			if err != nil {
				return nil, err
			}
		}
		if task.Active != nil {
			fields["active"], err = encodeStateObject(*task.Active, task.Active.Unknown)
			if err != nil {
				return nil, err
			}
		}
		if task.Last != nil {
			fields["last"], err = encodeStateObject(*task.Last, task.Last.Unknown)
			if err != nil {
				return nil, err
			}
		}
		// Keep nested preserving encodings raw; json.Marshal would compact them.
		rows[id], err = strictjson.EncodeObject(fields, nil, strictjson.KnownFields(reflect.TypeOf(task)))
		if err != nil {
			return nil, err
		}
	}
	tasks, err := strictjson.EncodeObject(rows, nil, nil)
	if err != nil {
		return nil, err
	}
	raw, err := strictjson.EncodeObject(map[string]json.RawMessage{"version": json.RawMessage("1"), "tasks": tasks}, l.Unknown, map[string]bool{"version": true, "tasks": true})
	if err != nil {
		return nil, err
	}
	if len(raw) > maxScheduleStateBytes {
		return nil, fmt.Errorf("schedule state exceeds byte cap")
	}
	return raw, nil
}
