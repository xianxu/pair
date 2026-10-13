package couchmessage

import (
	"encoding/json"
	"testing"
)

func switchWire(t *testing.T, op, target, agent, body, parent string) Request {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"Op": op, "Scope": "scope", "Tag": "tag", "Session": "session", "Nonce": "nonce", "ID": "id", "Target": target, "Agent": agent, "Body": body, "Confirmed": op == "switch-agent", "SwitchRequest": parent})
	var r Request
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestSwitchWireShapes(t *testing.T) {
	for _, target := range []string{"self", "pair:2"} {
		r := switchWire(t, "switch-agent", target, "codex", "context\n", "")
		if err := ValidateRequest(r); err != nil {
			t.Errorf("%s: %v", target, err)
		}
	}
	r := switchWire(t, "switch-agent", "self", "codex", "context\n", "parent")
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, r := range []Request{
		switchWire(t, "switch-agent", "pair", "codex", "", ""),
		switchWire(t, "switch-agent", "self", "bogus", "", ""),
		switchWire(t, "switch-agent", "self", "codex", "\r\n", ""),
		switchWire(t, "switch-agent", "self", "codex", "", "parent"),
		switchWire(t, "switch-agent", "pair:2", "codex", "body", "parent"),
		switchWire(t, "switch-agent", "self", "codex", "body", "bad id"),
	} {
		if ValidateRequest(r) == nil {
			t.Errorf("accepted %#v", r)
		}
	}
}
func TestSwitchParentRejectedBeforeEarlyShapes(t *testing.T) {
	binding := protocolBinding("pair:0")
	for _, r := range []Request{{Op: "broadcast-status"}, {Op: "tail", Target: "pair:2", Lines: 1}, {Op: "replay-status", Target: "record"}, {Op: "register", Binding: &binding}, {Op: "operator-submit", Binding: &binding}, switchWire(t, "operation-status", "", "", "", "")} {
		data, _ := json.Marshal(r)
		var fields map[string]any
		json.Unmarshal(data, &fields)
		fields["SwitchRequest"] = "parent"
		data, _ = json.Marshal(fields)
		json.Unmarshal(data, &r)
		if ValidateRequest(r) == nil {
			t.Errorf("accepted parent on %s", r.Op)
		}
	}
}

func FuzzSwitchParentClosedShape(f *testing.F) {
	for _, op := range []string{"send", "broadcast-status", "register", "tail", "operation-status", "actors", "resume"} {
		f.Add(op, "parent")
	}
	f.Fuzz(func(t *testing.T, op, parent string) {
		if op == "switch-agent" || parent == "" {
			return
		}
		r := switchWire(t, op, "", "", "", parent)
		if ValidateRequest(r) == nil {
			t.Fatalf("accepted parent on %q", op)
		}
	})
}
