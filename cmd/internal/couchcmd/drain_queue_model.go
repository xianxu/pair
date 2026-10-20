package couchcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const maxDrainRecordBytes = 1 << 20

// QueueSnapshot is an observation, never a lease. Unknown/incomplete snapshots
// cannot prove an owner's absence or absence of admitted work.
type QueueSnapshot struct {
	SchemaVersion int               `json:"schema_version"`
	State         string            `json:"state"`
	Complete      bool              `json:"complete"`
	PauseOwners   []QueuePauseOwner `json:"pause_owners"`
	Active        []QueueActivity   `json:"active"`
	Detail        string            `json:"detail"`
}
type QueuePauseOwner struct {
	Owner         string `json:"owner"`
	Reason        string `json:"reason"`
	CancelRunning bool   `json:"cancel_running"`
}
type QueueActivity struct {
	ID        string `json:"id"`
	Workspace string `json:"workspace"`
	Kind      string `json:"kind"`
	State     string `json:"state"`
	Head      string `json:"head,omitempty"`
}

func (q QueueSnapshot) Quiescent() bool {
	return q.SchemaVersion == 1 && q.Complete && q.State == "paused" && len(q.PauseOwners) > 0 && len(q.Active) == 0
}

func decodeQueueSnapshot(data []byte) (QueueSnapshot, error) {
	unknown := QueueSnapshot{State: "unknown"}
	fail := func(err error) (QueueSnapshot, error) {
		unknown.Detail = drainQueueDiagnostic(err.Error(), 4096)
		return unknown, err
	}
	obj, err := drainJSONObject(data)
	if err != nil {
		return fail(err)
	}
	topFields := []string{"schema_version", "state", "complete", "pause_owners", "active", "detail"}
	if err := rejectDrainQueueAliases(obj, topFields...); err != nil {
		return fail(err)
	}
	for _, key := range topFields {
		if err := drainRequired(obj, key); err != nil {
			return fail(err)
		}
	}
	var q QueueSnapshot
	if err := json.Unmarshal(data, &q); err != nil {
		return fail(fmt.Errorf("queue schema: %w", err))
	}
	if q.SchemaVersion != 1 {
		return fail(fmt.Errorf("unsupported queue schema_version %d", q.SchemaVersion))
	}
	if q.State != "running" && q.State != "paused" && q.State != "unknown" {
		return fail(fmt.Errorf("invalid queue state %q", q.State))
	}
	if len(q.PauseOwners) > 128 || len(q.Active) > 4096 {
		return fail(fmt.Errorf("queue observation exceeds cardinality bound"))
	}
	if q.State == "running" && len(q.PauseOwners) != 0 || q.State == "paused" && len(q.PauseOwners) == 0 {
		return fail(fmt.Errorf("queue state contradicts pause owners"))
	}
	if !drainBoundedText(q.Detail, 4096) || (q.State == "unknown" || !q.Complete) && q.Detail == "" {
		return fail(fmt.Errorf("unknown/incomplete queue requires bounded detail"))
	}
	var owners []map[string]json.RawMessage
	if err := json.Unmarshal(obj["pause_owners"], &owners); err != nil {
		return fail(err)
	}
	seen := map[string]bool{}
	for i, o := range q.PauseOwners {
		ownerFields := []string{"owner", "reason", "cancel_running"}
		if err := rejectDrainQueueAliases(owners[i], ownerFields...); err != nil {
			return fail(err)
		}
		for _, key := range ownerFields {
			if err := drainRequired(owners[i], key); err != nil {
				return fail(err)
			}
		}
		if o.Owner == "" || !drainBoundedText(o.Owner, 256) || !drainBoundedText(o.Reason, 4096) || seen[o.Owner] {
			return fail(fmt.Errorf("invalid or duplicate pause owner"))
		}
		seen[o.Owner] = true
	}
	var active []map[string]json.RawMessage
	if err := json.Unmarshal(obj["active"], &active); err != nil {
		return fail(err)
	}
	seen = map[string]bool{}
	for i, a := range q.Active {
		if err := rejectDrainQueueAliases(active[i], "id", "workspace", "kind", "state", "head"); err != nil {
			return fail(err)
		}
		for _, key := range []string{"id", "workspace", "kind", "state"} {
			if err := drainRequired(active[i], key); err != nil {
				return fail(err)
			}
		}
		if raw, ok := active[i]["head"]; ok && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fail(fmt.Errorf("null active head"))
		}
		if a.ID == "" || !drainBoundedText(a.ID, 256) || !drainBoundedText(a.Workspace, 4096) || !drainBoundedText(a.Head, 256) {
			return fail(fmt.Errorf("invalid active text"))
		}
		if a.Kind != "suite" && a.Kind != "merge" && a.Kind != "landing" {
			return fail(fmt.Errorf("invalid active kind %q", a.Kind))
		}
		if a.State != "running" && a.State != "interrupting" && a.State != "unknown" {
			return fail(fmt.Errorf("invalid active state %q", a.State))
		}
		key := a.ID + "\x00" + a.Kind
		if seen[key] {
			return fail(fmt.Errorf("duplicate active operation kind"))
		}
		seen[key] = true
	}
	return q, nil
}

// rejectDrainQueueAliases prevents encoding/json case-insensitive struct field
// matching from shadowing known keys, including through Unicode folds.
// Additive fields are allowed only when they cannot shadow a known key.
// Optional known keys (head) participate even when their canonical key is absent.
func rejectDrainQueueAliases(obj map[string]json.RawMessage, known ...string) error {
	for key := range obj {
		for _, name := range known {
			if key != name && strings.EqualFold(key, name) {
				return fmt.Errorf("case alias shadows known queue field %s", name)
			}
		}
	}
	return nil
}

func drainBoundedText(s string, n int) bool { return len(s) <= n && utf8.ValidString(s) }
func drainRequired(m map[string]json.RawMessage, key string) error {
	v, ok := m[key]
	if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		return fmt.Errorf("missing/null required field %s", key)
	}
	return nil
}

// Validate objects before typed decoding: encoding/json otherwise accepts
// duplicate keys and invalid UTF-8. Bound nesting independently of byte size.
func drainJSONObject(data []byte) (map[string]json.RawMessage, error) {
	if len(data) == 0 || len(data) > maxDrainRecordBytes || !utf8.Valid(data) {
		return nil, fmt.Errorf("invalid or oversized drain JSON")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := drainJSONValue(d, 0); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing drain JSON")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return nil, fmt.Errorf("drain JSON must be an object")
	}
	return obj, nil
}
func drainJSONValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("drain JSON nesting exceeds bound")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			key, ok := k.(string)
			if !ok || seen[key] {
				return fmt.Errorf("duplicate/invalid drain JSON key")
			}
			seen[key] = true
			if err := drainJSONValue(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := drainJSONValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid drain JSON delimiter")
	}
	_, err = d.Token()
	return err
}

func drainQueueDiagnostic(text string, limit int) string {
	if len(text) > limit {
		text = text[:limit]
	}
	text = strings.ToValidUTF8(text, "?")
	if len(text) > limit {
		end := limit
		for end > 0 && !utf8.RuneStart(text[end]) {
			end--
		}
		text = text[:end]
	}
	return text
}
