package couchmessage

import (
	"strings"
	"testing"
)

func TestParseAutomaticInputStall(t *testing.T) {
	good := `{"Sequence":1,"Cause":"no-render","WaitMillis":30000,"Fence":3,"Epoch":3,"Released":true}`
	if r, err := ParseAutomaticInputStall(good); err != nil || r.Sequence != 1 || !r.Released {
		t.Fatalf("%+v %v", r, err)
	}
	for _, body := range []string{`null`, `{}`, good + ` {}`, strings.Replace(good, `"Sequence":1`, `"Sequence":0`, 1), strings.Replace(good, `"WaitMillis":30000`, `"WaitMillis":-1`, 1), strings.Replace(good, `"WaitMillis":30000`, `"WaitMillis":604800001`, 1), strings.Replace(good, `"Cause":"no-render"`, `"Cause":"unrecognized-composer"`, 1), strings.Replace(good, `"Cause":"no-render"`, `"Cause":"other"`, 1), strings.Replace(good, `"Fence":3`, `"Fence":-1`, 1), strings.Replace(good, `"Epoch":3`, `"Epoch":null`, 1), strings.Replace(good, `"Released":true`, `"Released":null`, 1), strings.Replace(good, `"Sequence":1`, `"Sequence":1,"Sequence":2`, 1), strings.Replace(good, `"Fence":3`, `"foreign":3`, 1), strings.Repeat(" ", 2049) + good} {
		if _, err := ParseAutomaticInputStall(body); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}
func TestAutomaticInputStallRequestShape(t *testing.T) {
	good := Request{Op: "automatic-input-stall", Scope: "s", Tag: "t", Session: "session", Nonce: "nonce", ID: "automatic-input-1-1", Body: `{"Sequence":1,"Cause":"unrecognized-composer","WaitMillis":30000,"Fence":3,"Epoch":3,"Released":false}`}
	if err := ValidateRequest(good); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Request){func(r *Request) { r.Binding = &Binding{} }, func(r *Request) { r.Target = "pair:1" }, func(r *Request) { r.Title = "custom" }, func(r *Request) { r.Agent = "codex" }, func(r *Request) { r.Confirmed = true }, func(r *Request) { r.SameBinary = true }, func(r *Request) { r.ForceUnknown = true }, func(r *Request) { r.ToHub = "pair:0" }, func(r *Request) { r.Persist = true }, func(r *Request) { r.TailScope = "s" }, func(r *Request) { r.TailTag = "t" }, func(r *Request) { r.Lines = 1 }, func(r *Request) { r.Scope = "" }, func(r *Request) { r.ID = "" }, func(r *Request) { r.Body = "{}" }} {
		r := good
		mutate(&r)
		if err := ValidateRequest(r); err == nil {
			t.Errorf("accepted %+v", r)
		}
	}
}

func FuzzParseAutomaticInputStall(f *testing.F) {
	for _, seed := range []string{`{}`, `null`, `{"Sequence":1,"Cause":"no-render","WaitMillis":30000,"Fence":0,"Epoch":0,"Released":true}`, `{"Sequence":1,"Cause":"unrecognized-composer","WaitMillis":0,"Fence":0,"Epoch":0,"Released":false}`, `{"Sequence":1,"Sequence":2}`, `{} {}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		r, err := ParseAutomaticInputStall(body)
		if err != nil {
			return
		}
		if r.Sequence == 0 || r.WaitMillis < 0 || r.WaitMillis > MaxAutomaticInputStallWaitMillis || len(body) > MaxAutomaticInputStallBytes || (r.Released && r.Cause != AutomaticInputStallNoRender) {
			t.Fatalf("unsafe report %+v", r)
		}
	})
}

func TestAutomaticInputStallIDExactIncarnation(t *testing.T) {
	binding := Binding{PID: 42, Nonce: "first", Start: "start"}
	id := AutomaticInputStallID(binding, 1)
	if id != AutomaticInputStallID(binding, 1) || !validMessageID(id) {
		t.Fatalf("unstable or invalid ID %q", id)
	}
	for _, other := range []Binding{{PID: 42, Nonce: "second", Start: "start"}, {PID: 42, Nonce: "first", Start: "later"}} {
		if id == AutomaticInputStallID(other, 1) {
			t.Fatalf("ID reused for different incarnation %+v", other)
		}
	}
	if id == AutomaticInputStallID(binding, 2) {
		t.Fatal("sequence omitted")
	}
}
