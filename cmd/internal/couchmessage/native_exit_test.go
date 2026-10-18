package couchmessage

import "testing"

func TestParseNativeExit(t *testing.T) {
	r, err := ParseNativeExit(`{"LaunchOrdinal":3,"AgentIdentity":"1792:start","ExitCode":3}`)
	if err != nil || r != (NativeExit{LaunchOrdinal: 3, AgentIdentity: "1792:start", ExitCode: 3}) {
		t.Fatalf("report %+v err=%v", r, err)
	}
	for _, bad := range []string{
		`{}`, `null`, `{"LaunchOrdinal":3,"AgentIdentity":"a"}`,
		`{"LaunchOrdinal":0,"AgentIdentity":"a","ExitCode":0}`,
		`{"LaunchOrdinal":1,"AgentIdentity":"","ExitCode":0}`,
		`{"LaunchOrdinal":1,"AgentIdentity":"a","ExitCode":256}`,
		`{"LaunchOrdinal":1,"AgentIdentity":"a","ExitCode":-1}`,
		`{"LaunchOrdinal":1,"AgentIdentity":"a","ExitCode":0,"Extra":1}`,
		`{"LaunchOrdinal":1,"LaunchOrdinal":2,"AgentIdentity":"a","ExitCode":0}`,
	} {
		if _, err := ParseNativeExit(bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestNativeExitRequestShape(t *testing.T) {
	good := Request{Op: "native-exit", Scope: "s", Tag: "t", Session: "session", Nonce: "nonce", ID: "native-exit-1-1", Body: `{"LaunchOrdinal":1,"AgentIdentity":"a","ExitCode":0}`}
	if err := ValidateRequest(good); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Request){func(r *Request) { r.Binding = &Binding{} }, func(r *Request) { r.Target = "pair:1" }, func(r *Request) { r.Agent = "codex" }, func(r *Request) { r.Confirmed = true }, func(r *Request) { r.Lines = 1 }, func(r *Request) { r.Scope = "" }, func(r *Request) { r.Nonce = "" }, func(r *Request) { r.ID = "" }, func(r *Request) { r.Body = "{}" }} {
		r := good
		mutate(&r)
		if err := ValidateRequest(r); err == nil {
			t.Errorf("accepted %+v", r)
		}
	}
}

func TestNativeExitIDExactIncarnationAndLaunch(t *testing.T) {
	binding := Binding{PID: 42, Nonce: "first", Start: "start"}
	id := NativeExitID(binding, 1)
	if id != NativeExitID(binding, 1) || !validMessageID(id) {
		t.Fatalf("unstable or invalid ID %q", id)
	}
	if id == NativeExitID(Binding{PID: 42, Nonce: "second", Start: "start"}, 1) || id == NativeExitID(binding, 2) {
		t.Fatal("ID reused across incarnations or launches")
	}
}
